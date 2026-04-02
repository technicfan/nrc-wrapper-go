package launchers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"main/globals"
	"main/platform"
	"main/utils"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	MODRINTH_DIR     = "ModrinthApp"
	MODRINTH_FLATPAK = "com.modrinth.ModrinthApp"
	MODRINTH_CLASS   = "com.modrinth.theseus.MinecraftLaunch"
)

type modrinthapp struct /*implements Launcher*/ {
	*launcher_data
}

func NewModrinthApp(home string, path string, flatpak bool) Launcher {
	var name, flatpak_id string
	if path == "" {
		path = utils.LauncherDir(home, flatpak, MODRINTH_FLATPAK, MODRINTH_DIR)
	}
	if flatpak {
		flatpak_id = MODRINTH_FLATPAK
		name = "Modrinth App (Flatpak)"
	} else {
		name = "Modrinth App"
	}
	return modrinthapp{&launcher_data{name, path, filepath.Join(path, "profiles"), flatpak_id, false, ""}}
}

func (launcher modrinthapp) Exists() bool {
	_, err := os.Stat(filepath.Join(launcher.path, "app.db"))
	return err == nil
}

func (launcher modrinthapp) Id() string {
	return "modrinth"
}

func (launcher modrinthapp) IsRunning() bool {
	var pname string
	if platform.WINDOWS {
		pname = "Modrinth App.exe"
	} else if launcher.flatpak_id != "" {
		pname = "modrinth-app-wrapped"
	} else {
		pname = "modrinth-app"
	}
	return platform.IsRunning(pname)
}

func (launcher modrinthapp) GetCurrentInstanceDetails() (Minecraft, error) {
	var profile, version, loader, loader_version, token, username, uuid string

	db, err := sql.Open("sqlite3", filepath.Join(launcher.path, "app.db"))
	if err != nil {
		return Minecraft{}, err
	}
	defer db.Close()

	cwd, err := os.Getwd()
	if err != nil {
		return Minecraft{}, err
	}
	rows, err := db.Query(
		fmt.Sprintf(
			"SELECT name, game_version, mod_loader, mod_loader_version FROM profiles WHERE path = '%s'",
			filepath.Base(cwd),
		),
	)
	if err != nil {
		return Minecraft{}, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&profile, &version, &loader, &loader_version)
		if err != nil {
			return Minecraft{}, err
		}
	}

	rows, err = db.Query(
		"SELECT access_token, username, uuid FROM minecraft_users where active = 1",
	)
	if err != nil {
		return Minecraft{}, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&token, &username, &uuid)
		if err != nil {
			return Minecraft{}, err
		}
	}

	return Minecraft{profile, version, loader, loader_version, username, uuid, token}, nil
}

func (launcher modrinthapp) GetInstances(
	versions []string,
	loaders []string,
	ex string,
) ([]Instance, error) {
	var instances []Instance

	db, err := sql.Open("sqlite3", filepath.Join(launcher.path, "app.db"))
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
			var data [][]string

			pack := globals.DEFAULT_PACK
			mod_path := globals.DEFAULT_MOD_DIR
			notify := true
			if wrapper_ptr != nil {
				wrapper = *wrapper_ptr
			}
			nrc := strings.Contains(wrapper, filepath.Base(ex)) || strings.Contains(wrapper, ex)
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
			nrc_config := nrc_config{nrc, wrapper, pack, mod_path, notify, neofd}
			path := filepath.Join(launcher.instance_dir, instance_path)
			instances = append(instances, &modrinth_instance{&instance_data{
				name, version, loader, loader_version, path, path,
				vars, launcher.flatpak_id, nrc_config, true,
			}})
		}
	}
	slices.SortFunc(instances, func(a Instance, b Instance) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return instances, nil
}

type modrinth_instance struct /*implements Instance*/ {
	*instance_data
}

func (instance modrinth_instance) LauncherClass() string {
	return MODRINTH_CLASS
}

func (instance *modrinth_instance) Save(nrc bool, notify bool, neofd bool, pack string, ex string) error {
	if instance.instance_data.save(nrc, notify, neofd, pack, ex) {
		var env [][]string
		for k, v := range instance.env {
			env = append(env, []string{k, v})
		}
		raw, err := json.Marshal(env)
		if err != nil {
			return err
		}
		db, err := sql.Open(
			"sqlite3", fmt.Sprintf("%s/app.db", filepath.Dir(filepath.Dir(instance.path))),
		)
		if err != nil {
			return err
		}
		defer db.Close()
		sql_cmd := `UPDATE profiles SET override_hook_wrapper = ?, override_custom_env_vars = jsonb(?) WHERE path = ?;`
		_, err = db.Exec(sql_cmd, instance.config.command, raw, filepath.Base(instance.path))
		return err
	}
	return nil
}
