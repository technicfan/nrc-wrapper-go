package main

import (
	"os/user"
	"path/filepath"
)

func get_config() map[string]string {
	config := make(map[string]string)
	usr, _ := user.Current()
	home := usr.HomeDir
	config["prism_dir"] = filepath.Join(home, PRISM_DIR)

	return config
}
