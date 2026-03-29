package launchers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"main/globals"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func get_modrinth_details(
	path string,
) (Minecraft, error) {
	var profile, version, loader, loader_version, token, username, uuid string

	db, err := sql.Open("sqlite3", fmt.Sprintf("%s/app.db", path))
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
	db, err := sql.Open(
		"sqlite3", fmt.Sprintf("%s/app.db", filepath.Dir(filepath.Dir(instance.Path))),
	)
	if err != nil {
		return err
	}
	defer db.Close()
	sql_cmd := `UPDATE profiles SET override_hook_wrapper = ?, override_custom_env_vars = jsonb(?) WHERE path = ?;`
	_, err = db.Exec(sql_cmd, instance.Config.Command, raw, filepath.Base(instance.Path))
	return err
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
			nrc_config := NrcConfig{nrc, wrapper, pack, mod_path, notify, neofd}
			path := filepath.Join(path, "profiles", instance_path)
			instances = append(instances, Instance{
				name, version, loader, loader_version, path, path,
				"modrinth", nil, vars, flatpak, nrc_config, nrc_config,
			})
		}
	}
	return instances, nil
}
