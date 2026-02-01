package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type NrcConfig struct {
	Nrc bool
	Command string
	NrcPack string
	ModDir string
	Notify bool
	Staging bool
	Neofd bool
}

type Instance struct {
	Name string
	Version string
	Loader string
	LoaderVersion string
	Path string
	Launcher string
	Cfg Cfg
	Env map[string]string
	FlatpakId string
	Config NrcConfig
	NewConfig NrcConfig
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

func (instance *Instance) update(ex string) error {
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
					cmd := exec.Command("flatpak", "override", "--user", fmt.Sprintf("filesystem=%s", ex), instance.FlatpakId)
					cmd.Run()
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
				delete(instance.Env, "STAGING")
				instance.Config.Staging = false
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
				instance.Env["NRC_PACK"] = instance.Config.NrcPack
			}
			if instance.Config.Staging != instance.NewConfig.Staging {
				instance.Config.Staging = instance.NewConfig.Staging
				if instance.Config.Staging {
					instance.Env["STAGING"] = "true"
				} else {
					delete(instance.Env, "STAGING")
				}
			}
			if instance.Config.ModDir != instance.NewConfig.ModDir {
				instance.Config.ModDir = instance.NewConfig.ModDir
				if instance.Config.ModDir != "" {
					instance.Env["NRC_MOD_DIR"] = instance.Config.ModDir
				} else {
					delete(instance.Env, "NRC_MOD_DIR")
				}
			}
			if instance.Config.Notify != instance.NewConfig.Notify {
				instance.Config.Notify = instance.NewConfig.Notify
				if instance.Config.Notify {
					instance.Env["NOTIFY"] = "true"
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
		switch(instance.Launcher) {
			case "prism": err = update_prism_instance(instance)
			case "modrinth": err = update_modrinth_instance(instance)
			default: return fmt.Errorf("%s is an invalid launcher", instance.Launcher)
		}
		if err != nil {
			return err
		}
		instance.NewConfig = instance.Config
	}
	return nil
}

func update_prism_instance(
	instance *Instance,
) error {
	instance.Cfg["General"]["WrapperCommand"] = instance.Config.Command
	if instance.Config.Nrc {
		instance.Cfg["General"]["OverrideEnv"] = "true"
	}
	raw_env, err := json.Marshal(instance.Env)
	if err != nil {
		return err
	}
	instance.Cfg["General"]["Env"] = strings.ReplaceAll(string(raw_env), `"`, `\"`)
	return instance.Cfg.write(filepath.Join(instance.Path, "instance.cfg"))
}

func update_modrinth_instance(
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
		return instances, nil
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
			var staging, notify, neofd bool
			var mod_path string
			pack := "norisk-prod"
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
			if v, e := vars["STAGING"]; e && v != "" {
				staging = true
			}
			v, e := vars["NEOFD"]
			v2, e2 := vars["NO_ERROR_ON_FAILED_DOWNLOAD"]
			if (e && v != "") || (e2 && v2 != "") {
				neofd = true
			}
			nrc_config := NrcConfig{nrc, wrapper, pack, mod_path, notify, staging, neofd}
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
			var staging, neofd bool
			var mod_path string
			pack := "norisk-prod"
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
			if v, e := vars["STAGING"]; e && v != "" {
				staging = true
			}
			v, e := vars["NEOFD"]
			v2, e2 := vars["NO_ERROR_ON_FAILED_DOWNLOAD"]
			if (e && v != "") || (e2 && v2 != "") {
				neofd = true
			}
			nrc_config := NrcConfig{nrc, wrapper, pack, mod_path, notify, staging, neofd}
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
	Name string
	Desc string
	Versions []string
	Loaders map[string]string
}

type MetaPacks struct {
	Packs map[string]MetaPack
	Versions []string
	Loaders []string
}
