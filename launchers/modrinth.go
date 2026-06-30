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
	MODRINTH_ID      = "modrinth"
	MODRINTH_NAME    = "Modrinth App"
	MODRINTH_DIR     = "ModrinthApp"
	MODRINTH_FLATPAK = "com.modrinth.ModrinthApp"
	MODRINTH_CLASS   = "com.modrinth.theseus.MinecraftLaunch"
)

type modrinthapp struct /*implements Launcher*/ {
	*launcher_data
}

type launch_overrides struct {
	Env [][]string `json:"custom_env_vars"`
	Hooks struct {
		Wrapper *string `json:"wrapper"`
	} `json:"hooks"`
}

func NewModrinthApp(home string, path string, flatpak bool) Launcher {
	var flatpak_id string
	name := MODRINTH_NAME
	if path == "" {
		path = utils.LauncherDir(home, flatpak, MODRINTH_FLATPAK, MODRINTH_DIR)
	}
	if flatpak {
		flatpak_id = MODRINTH_FLATPAK
		name += " (Flatpak)"
	}
	return modrinthapp{&launcher_data{name, path, filepath.Join(path, "profiles"), flatpak_id, false}}
}

func (launcher modrinthapp) Exists() bool {
	_, err := os.Stat(filepath.Join(launcher.path, "app.db"))
	return err == nil
}

func (launcher modrinthapp) Id() string {
	return MODRINTH_ID
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

func (launcher modrinthapp) DefaultNotify() bool {
	return true
}

func (launcher modrinthapp) MergeNormal() Launcher {
	if launcher.flatpak_id != "" {
		launcher.replaced = true
	}
	return launcher
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
			"SELECT i.name, c.game_version, c.loader, c.loader_version FROM instances i INNER JOIN instance_content_sets c ON i.applied_content_set_id = c.id WHERE i.path = '%s'",
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
	support map[string][]string,
	ex string,
) ([]Instance, error) {
	var instances []Instance

	db, err := sql.Open("sqlite3", filepath.Join(launcher.path, "app.db"))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		"SELECT i.name, i.id, c.game_version, c.loader, c.loader_version, i.path, json(o.overrides) FROM instances i INNER JOIN instance_content_sets c ON c.id = i.applied_content_set_id LEFT JOIN instance_launch_overrides o ON o.instance_id = i.id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var raw_overrides []byte
		var name, id, version, loader, loader_version, instance_path, wrapper string

		err = rows.Scan(&name, &id, &version, &loader, &loader_version, &instance_path, &raw_overrides)
		if err == nil {
			if versions, e := support[loader]; !e || !slices.Contains(versions, version) {
				continue
			}

			var neofd, staging bool
			var overrides launch_overrides

			pack := globals.DEFAULT_PACK
			mod_path := globals.DEFAULT_MOD_DIR
			notify := true
			err := json.Unmarshal(raw_overrides, &overrides)
			if err != nil {
				return nil, err
			}
			if overrides.Hooks.Wrapper != nil {
				wrapper = *overrides.Hooks.Wrapper
			}
			nrc := strings.Contains(wrapper, filepath.Base(ex)) || strings.Contains(wrapper, ex)
			vars := make(map[string]string)
			for _, line := range overrides.Env {
				vars[line[0]] = line[1]
			}
			if v, e := vars["NRC_PACK"]; e {
				pack = v
			}
			if v, e := vars["STAGING"]; e && v != "" {
				staging = true
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
			nrc_config := nrc_config{nrc, wrapper, staging, pack, mod_path, notify, neofd}
			path := filepath.Join(launcher.instance_dir, instance_path)
			instances = append(instances, &modrinth_instance{&instance_data{
				name, version, loader, loader_version, path, path,
				vars, launcher.flatpak_id, nrc_config, launcher.DefaultNotify(),
			}, id})
		}
	}
	slices.SortFunc(instances, func(a Instance, b Instance) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return instances, nil
}

type modrinth_instance struct /*implements Instance*/ {
	*instance_data
	id string
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
		sql_cmd := `UPDATE instance_launch_overrides SET overrides = jsonb(json_set(overrides, '$.custom_env_vars', jsonb(?), '$.hooks.wrapper', ?)) WHERE instance_id = ?;`
		_, err = db.Exec(sql_cmd, raw, instance.config.command, instance.id)
		return err
	}
	return nil
}
