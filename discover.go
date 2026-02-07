package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type NrcConfig struct {
	Nrc     bool
	Command string
	NrcPack string
	ModDir  string
	Notify  bool
	Neofd   bool
}

type Instance struct {
	Name          string
	Version       string
	Loader        string
	LoaderVersion string
	Path          string
	Launcher      string
	Cfg           Cfg
	Env           map[string]string
	FlatpakId     string
	Config        NrcConfig
	NewConfig     NrcConfig
}

type Cfg map[string]map[string]string

func (cfg *Cfg) write(
	filename string,
) error {
	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for s, kv := range *cfg {
		fmt.Fprintf(writer, "[%s]\n", s)
		for k, v := range kv {
			fmt.Fprintf(writer, "%s=%s\n", k, v)
		}
	}
	writer.Flush()
	return nil
}

func (instance *Instance) save(ex string) error {
	if instance.Config != instance.NewConfig {
		if instance.Config.Nrc != instance.NewConfig.Nrc {
			instance.Config.Nrc = instance.NewConfig.Nrc
			if instance.Config.Nrc {
				if instance.Config.Command == "" {
					instance.Config.Command = ex
				} else {
					instance.Config.Command += " " + ex
				}
				if instance.FlatpakId != "" {
					cmd := exec.Command("flatpak", "override", "--user", "--show", instance.FlatpakId)
					if o, err := cmd.Output(); err == nil && !strings.Contains(string(o), instance.Config.Command) {
						cmd = exec.Command("flatpak", "override", "--user", fmt.Sprintf("--filesystem=%s", ex), instance.FlatpakId)
						cmd.Run()
					}
				}
			} else {
				if instance.Config.Command == ex || instance.Config.Command == filepath.Base(ex) {
					instance.Config.Command = ""
				} else {
					cmd := ex
					if !strings.Contains(instance.Config.Command, ex) {
						cmd = filepath.Base(ex)
					}
					instance.Config.Command = strings.TrimSpace(strings.ReplaceAll(instance.Config.Command, cmd, ""))
				}
				delete(instance.Env, "NRC_PACK")
				instance.Config.NrcPack = ""
				delete(instance.Env, "NRC_MOD_DIR")
				instance.Config.ModDir = ""
				delete(instance.Env, "NOTIFY")
				instance.Config.Notify = instance.Launcher == "modrinth"
				delete(instance.Env, "NEOFD")
				delete(instance.Env, "NO_ERROR_ON_FAILED_DOWNLOAD")
				instance.Config.Neofd = false
			}
		}
		if instance.Config.Nrc {
			if instance.Config.NrcPack != instance.NewConfig.NrcPack {
				instance.Config.NrcPack = instance.NewConfig.NrcPack
				if instance.Config.NrcPack == "" || instance.Config.NrcPack == DEFAULT_PACK {
				    delete(instance.Env, "NRC_PACK")
				} else {
					instance.Env["NRC_PACK"] = instance.Config.NrcPack
				}
			}
			if instance.Config.ModDir != instance.NewConfig.ModDir {
				instance.Config.ModDir = instance.NewConfig.ModDir
				if instance.Config.ModDir != "" && instance.Config.ModDir != DEFAULT_MOD_DIR {
					instance.Env["NRC_MOD_DIR"] = instance.Config.ModDir
				} else {
					delete(instance.Env, "NRC_MOD_DIR")
				}
			}
			if instance.Config.Notify != instance.NewConfig.Notify {
				instance.Config.Notify = instance.NewConfig.Notify
				if instance.Config.Notify {
					if instance.Launcher == "modrinth" {
						delete(instance.Env, "NOTIFY")
					} else {
						instance.Env["NOTIFY"] = "true"
					}
				} else {
					instance.Env["NOTIFY"] = "false"
				}
			}
			if instance.Config.Neofd != instance.NewConfig.Neofd {
				instance.Config.Neofd = instance.NewConfig.Neofd
				if instance.Config.Neofd {
					instance.Env["NO_ERROR_ON_FAILED_DOWNLOAD"] = "true"
				} else {
					delete(instance.Env, "NO_ERROR_ON_FAILED_DOWNLOAD")
				}
			}
		}
		var err error
		switch instance.Launcher {
		case "prism":
			err = save_prism_instance(instance)
		case "modrinth":
			err = save_modrinth_instance(instance)
		default:
			return fmt.Errorf("%s is an invalid launcher", instance.Launcher)
		}
		if err != nil {
			return err
		}
		instance.NewConfig = instance.Config
	}
	return nil
}

func save_prism_instance(
	instance *Instance,
) error {
	if instance.Config.Command != "" {
		instance.Cfg["General"]["OverrideCommands"] = "true"
		instance.Cfg["General"]["WrapperCommand"] = instance.Config.Command
	} else {
		instance.Cfg["General"]["OverrideCommands"] = "false"
		instance.Cfg["General"]["WrapperCommand"] = ""
	}
	if len(instance.Env) != 0 {
		instance.Cfg["General"]["OverrideEnv"] = "true"
	} else {
		instance.Cfg["General"]["OverrideEnv"] = "false"
	}
	raw_env, err := json.Marshal(instance.Env)
	if err != nil {
		return err
	}
	env := strings.ReplaceAll(strings.Trim(string(raw_env), `"`), `"`, `\"`)
	if len(instance.Env) >= 2 {
		env = `"` + env + `"`
	}
	instance.Cfg["General"]["Env"] = env
	return instance.Cfg.write(filepath.Join(instance.Path, "instance.cfg"))
}

func save_modrinth_instance(
	instance *Instance,
) error {
	var env [][]string
	for k, v := range instance.Env {
		env = append(env, []string{k, v})
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s/app.db", filepath.Dir(filepath.Dir(instance.Path))))
	if err != nil {
		return err
	}
	defer db.Close()
	sql_cmd := `UPDATE profiles SET override_hook_wrapper = ?, override_custom_env_vars = jsonb(?) WHERE path = ?;`
	_, err = db.Exec(sql_cmd, instance.Config.Command, raw, filepath.Base(instance.Path))
	return err
}

func get_prism_instance_dir(
	path string,
) (string, error) {
	config, err := parse_cfg(filepath.Join(path, "prismlauncher.cfg"))
	if err != nil {
		return "", err
	}
	if instances, exists := config["General"]["InstanceDir"]; exists {
		if regexp.MustCompile("^([A-Z]:|/).*").MatchString(instances) {
			return instances, nil
		}
		return filepath.Join(path, instances), nil
	}
	return filepath.Join(path, "instances"), nil
}

func get_instances(
	path string,
	flatpak string,
	versions []string,
	loaders []string,
	ex string,
) ([]Instance, error) {
	if strings.Contains(path, "ModrinthApp") {
		return get_modrinth_instances(path, flatpak, versions, loaders, ex)
	} else {
		return get_prism_instances(path, flatpak, versions, loaders, ex)
	}
}

func get_prism_instances(
	path string,
	flatpak string,
	versions []string,
	loaders []string,
	ex string,
) ([]Instance, error) {
	var instances []Instance

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, dir := range files {
		if dir.IsDir() {
			instance_path := filepath.Join(path, dir.Name())
			instance, err := get_prism_instance(instance_path)
			if err != nil {
				continue
			}
			version, loader, loader_version := instance.get_details()
			if !slices.Contains(versions, version) || !slices.Contains(loaders, loader) {
				continue
			}
			config, err := parse_cfg(filepath.Join(instance_path, "instance.cfg"))
			if err != nil {
				continue
			}
			wrapper := config["General"]["WrapperCommand"]
			name := config["General"]["name"]
			env := strings.Trim(strings.ReplaceAll(config["General"]["Env"], `\"`, `"`), `"`)
			var vars map[string]string
			if config["General"]["OverrideEnv"] == "true" {
				err = json.Unmarshal([]byte(env), &vars)
				if err != nil {
					continue
				}
			} else {
				vars = make(map[string]string)
			}
			var notify, neofd bool
			var mod_path string
			pack := DEFAULT_PACK
			nrc := strings.Contains(wrapper, filepath.Base(ex)) || strings.Contains(wrapper, ex)
			if v, e := vars["NRC_PACK"]; e {
				pack = v
			}
			if v, e := vars["NRC_MOD_DIR"]; e {
				mod_path = v
			}
			if v, e := vars["NOTIFY"]; e {
				notify = !(v == "False" || v == "false" || v == "0")
			}
			v, e := vars["NEOFD"]
			v2, e2 := vars["NO_ERROR_ON_FAILED_DOWNLOAD"]
			if (e && v != "") || (e2 && v2 != "") {
				neofd = true
			}
			nrc_config := NrcConfig{nrc, wrapper, pack, mod_path, notify, neofd}
			instances = append(instances, Instance{
				name, version, loader, loader_version, instance_path, "prism", config, vars,
				flatpak, nrc_config, nrc_config,
			})
		}
	}
	return instances, nil
}

func get_modrinth_instances(
	path string,
	flatpak string,
	versions []string,
	loaders []string,
	ex string,
) ([]Instance, error) {
	var instances []Instance

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s/app.db", path))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		"SELECT name, game_version, mod_loader, mod_loader_version, path, override_hook_wrapper, json(override_custom_env_vars) FROM profiles",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var env []byte
		var name, version, loader, loader_version, instance_path, wrapper string
		var wrapper_ptr *string
		err = rows.Scan(&name, &version, &loader, &loader_version, &instance_path, &wrapper_ptr, &env)
		if err == nil && slices.Contains(versions, version) && slices.Contains(loaders, loader) {
			var neofd bool
			var mod_path string
			pack := DEFAULT_PACK
			notify := true
			if wrapper_ptr != nil {
				wrapper = *wrapper_ptr
			}
			nrc := strings.Contains(wrapper, filepath.Base(ex)) || strings.Contains(wrapper, ex)
			var data [][]string
			err := json.Unmarshal(env, &data)
			if err != nil {
				return nil, err
			}
			vars := make(map[string]string)
			for _, line := range data {
				vars[line[0]] = line[1]
			}
			if v, e := vars["NRC_PACK"]; e {
				pack = v
			}
			if v, e := vars["NRC_MOD_DIR"]; e {
				mod_path = v
			}
			if v, e := vars["NOTIFY"]; e {
				notify = !(v == "False" || v == "false" || v == "0")
			}
			v, e := vars["NEOFD"]
			v2, e2 := vars["NO_ERROR_ON_FAILED_DOWNLOAD"]
			if (e && v != "") || (e2 && v2 != "") {
				neofd = true
			}
			nrc_config := NrcConfig{nrc, wrapper, pack, mod_path, notify, neofd}
			instances = append(instances, Instance{
				name, version, loader, loader_version, filepath.Join(path, "profiles", instance_path), "modrinth", nil, vars,
				flatpak, nrc_config, nrc_config,
			})
		}
	}
	return instances, nil
}

func parse_cfg(
	filename string,
) (Cfg, error) {
	file, err := os.OpenFile(filepath.Join(filename), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var current_section string
	config := make(map[string]map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == "" || strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		if match, _ := regexp.MatchString(`^\[.+\]$`, scanner.Text()); match {
			current_section = regexp.MustCompile(`^\[|\]$`).ReplaceAllString(scanner.Text(), "")
		} else if match, _ := regexp.MatchString("^.+=.*$", scanner.Text()); match && current_section != "" {
			k := regexp.MustCompile("=.*$").ReplaceAllString(scanner.Text(), "")
			v := regexp.MustCompile("^.+=").ReplaceAllString(scanner.Text(), "")
			if _, exists := config[current_section]; !exists {
				config[current_section] = make(map[string]string)
			}
			config[current_section][k] = v
		} else {
			return nil, errors.New("Invalid config")
		}
	}

	if v, e := config["General"]["ConfigVersion"]; !e || cmp_mc_versions(v, "1.3") < 0 {
		return nil, fmt.Errorf("%s is too old. Only config version >= 1.3 compatible", v)
	}

	return config, nil
}

type MetaPack struct {
	Name     string
	Desc     string
	Versions []string
	Loaders  map[string]string
}

type MetaPacks struct {
	Packs    map[string]MetaPack
	Versions []string
	Loaders  []string
	Names    []string
}

func (packs *MetaPacks) get_compatible_packs(version string, loader string) ([]string, []string) {
	var index int
	var unique_pack_names []string
	pack_ids := MAIN_PACKS
	for _, i := range MAIN_PACKS {
		if _, e := packs.Packs[i].Loaders[loader]; e && slices.Contains(packs.Packs[i].Versions, version) {
			unique_pack_names = append(unique_pack_names, make_unique(packs.Packs[i].Name, index))
			index++
		}
	}
	for i := range packs.Packs {
		if _, e := packs.Packs[i].Loaders[loader]; e && slices.Contains(packs.Packs[i].Versions, version) && !slices.Contains(MAIN_PACKS, i) {
			unique_pack_names = append(unique_pack_names, make_unique(packs.Packs[i].Name, index))
			pack_ids = append(pack_ids, i)
			index++
		}
	}
	return unique_pack_names, pack_ids
}

func make_unique(str string, index int) string {
	var builder strings.Builder
	builder.WriteString(str)
	for range index {
		builder.WriteRune('\u200d')
	}
	return builder.String()
}

func get_launcher_dirs() (map[string][]string, []string) {
	dirs, order := get_const_dirs()
	for i, l := range order {
		if l == "" {
			continue
		}
		_, err := os.Stat(dirs[l][0])
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			order = slices.Delete(order, i, i+1)
		} else if err == nil && strings.HasPrefix(l, "Prism") {
			dir, err := get_prism_instance_dir(dirs[l][0])
			if dirs["Prism Launcher"][0] == dir {
				i := slices.Index(order, "Prism Launcher")
				order = slices.Delete(order, i, i+1)
			}
			if err != nil {
				order = slices.Delete(order, i, i+1)
			}
			dirs[l][0] = dir
		}
	}

	return dirs, order
}
