package launchers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"main/globals"
	"main/utils"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func Get_prism_instance_dir(
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

type PrismData struct {
	Accounts []struct {
		Active  any `json:"active"`
		Profile struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"profile"`
		Type string `json:"type"`
		Ygg  struct {
			Token string `json:"token"`
		} `json:"ygg"`
	} `json:"accounts"`
	FormatVersion int `json:"formatVersion"`
}

func (data PrismData) get(
	id *string,
) (string, string, string, error) {
	var token string
	for _, v := range data.Accounts {
		if (id != nil && v.Profile.Id == *id) || (id == nil && v.Active != nil && v.Active.(bool)) {
			if v.Type == "Offline" {
				token = "offline"
			} else {
				token = v.Ygg.Token
			}
			return token, v.Profile.Name, v.Profile.Id, nil
		}
	}

	var err error
	if id != nil {
		err = fmt.Errorf("Account with id %s not found", id)
	} else {
		err = errors.New("No active account found")
	}
	return "", "", "", err
}

func (data PrismData) get_active() (string, string, string, error) {
	return data.get(nil)
}

type prism_instance_config struct {
	Components []struct {
		Uid     string `json:"uid"`
		Version string `json:"version"`
	} `json:"components"`
}

func (instance *prism_instance_config) get_details() (string, string, string) {
	var version, loader, loader_version string
	for _, entry := range instance.Components {
		switch entry.Uid {
		case "net.minecraft":
			version = entry.Version
		case "net.fabricmc.fabric-loader":
			loader = "fabric"
			loader_version = entry.Version
		case "org.quiltmc.quilt-loader":
			loader = "quilt"
			loader_version = entry.Version
		case "net.minecraftforge":
			loader = "forge"
			loader_version = entry.Version
		case "net.neoforged":
			loader = "neoforge"
			loader_version = entry.Version
		}
	}
	return version, loader, loader_version
}

func get_prism_details(
	path string,
) (Minecraft, error) {
	var profile, version, loader, loader_version, token, username, uuid string

	instance, err := get_prism_instance("../")
	if err != nil {
		return Minecraft{}, err
	}
	version, loader, loader_version = instance.get_details()

	config, err := parse_cfg("../instance.cfg")
	if err != nil {
		return Minecraft{}, err
	}

	if name, e := config["General"]["name"]; e {
		profile = name
	}

	file, err := os.Open(fmt.Sprintf("%s/accounts.json", path))
	if err != nil {
		return Minecraft{}, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return Minecraft{}, err
	}

	var data PrismData
	err = json.Unmarshal(content, &data)
	if err != nil {
		return Minecraft{}, err
	}
	if id, e := config["General"]["InstanceAccountId"]; e && config["General"]["UseAccountForInstance"] == "true" {
		token, username, uuid, err = data.get(&id)
	} else {
		token, username, uuid, err = data.get_active()
	}
	if err != nil {
		return Minecraft{}, err
	}

	return Minecraft{
		profile,
		version,
		loader,
		loader_version,
		username,
		uuid,
		token,
	}, nil
}

func get_prism_instance(
	path string,
) (prism_instance_config, error) {
	file, err := os.OpenFile(filepath.Join(path, "mmc-pack.json"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return prism_instance_config{}, err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return prism_instance_config{}, err
	}
	defer file.Close()

	var instance prism_instance_config
	err = json.Unmarshal(content, &instance)
	if err != nil {
		return prism_instance_config{}, err
	}

	return instance, nil
}

type prism_instance struct {
	*instance_data
	cfg cfg
}

func (instance *prism_instance) Save(nrc bool, notify bool, neofd bool, pack string, ex string) error {
	if (instance.instance_data.save(nrc, notify, neofd, pack, ex)) {
		if instance.config.command != "" {
			instance.cfg["General"]["OverrideCommands"] = "true"
			instance.cfg["General"]["WrapperCommand"] = instance.config.command
		} else {
			instance.cfg["General"]["OverrideCommands"] = "false"
			instance.cfg["General"]["WrapperCommand"] = ""
		}
		if len(instance.env) != 0 {
			instance.cfg["General"]["OverrideEnv"] = "true"
		} else {
			instance.cfg["General"]["OverrideEnv"] = "false"
		}
		raw_env, err := json.Marshal(instance.env)
		if err != nil {
			return err
		}
		env := strings.ReplaceAll(strings.Trim(string(raw_env), `"`), `"`, `\"`)
		if len(instance.env) >= 2 {
			env = `"` + env + `"`
		}
		instance.cfg["General"]["Env"] = env
		return instance.cfg.write(filepath.Join(instance.path, "instance.cfg"))
	}
	return nil
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
			var vars map[string]string
			var notify, neofd bool
			var wrapper string

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
			if config["General"]["OverrideCommands"] == "true" {
				wrapper = config["General"]["WrapperCommand"]
			}
			name := config["General"]["name"]
			env := strings.Trim(strings.ReplaceAll(config["General"]["Env"], `\"`, `"`), `"`)
			if config["General"]["OverrideEnv"] == "true" {
				err = json.Unmarshal([]byte(env), &vars)
				if err != nil {
					continue
				}
			} else {
				vars = make(map[string]string)
			}

			pack := globals.DEFAULT_PACK
			mod_path := globals.DEFAULT_MOD_DIR
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
			nrc_config := nrc_config{nrc, wrapper, pack, mod_path, notify, neofd}
			mc_root := filepath.Join(instance_path, "minecraft")
			_, err = os.Stat(mc_root)
			if err != nil && errors.Is(err, fs.ErrNotExist) {
				mc_root = filepath.Join(instance_path, ".minecraft")
			}
			instances = append(instances, &prism_instance{&instance_data{
				name, version, loader, loader_version, instance_path, mc_root,
				vars, flatpak, nrc_config,
			}, config})
		}
	}
	return instances, nil
}

type cfg map[string]map[string]string

func (cfg cfg) write(
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
	for s, kv := range cfg {
		fmt.Fprintf(writer, "[%s]\n", s)
		for k, v := range kv {
			fmt.Fprintf(writer, "%s=%s\n", k, v)
		}
	}
	writer.Flush()
	return nil
}

func parse_cfg(
	filename string,
) (cfg, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
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
		if strings.HasPrefix(scanner.Text(), "[") && strings.HasSuffix(scanner.Text(), "]") {
			current_section = strings.Trim(scanner.Text(), "[]")
		} else if k, v, e := strings.Cut(scanner.Text(), "="); e && k != "" {
			if _, exists := config[current_section]; !exists {
				config[current_section] = make(map[string]string)
			}
			config[current_section][k] = v
		} else {
			return nil, errors.New("Invalid config")
		}
	}

	if v, e := config["General"]["ConfigVersion"]; !e || utils.Cmp_versions(v, "1.3") < 0 {
		return nil, fmt.Errorf("%s is too old. Only config version >= 1.3 compatible", v)
	}

	return config, nil
}
