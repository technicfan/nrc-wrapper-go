package main

import (
	"os"
	"os/user"
	"path/filepath"
)

func get_config() map[string]string {
	config := make(map[string]string)
	usr, _ := user.Current()
	home := usr.HomeDir
	config["prism_dir"] = filepath.Join(home, PRISM_DIR)
	if value := os.Getenv("NRC_PACK"); value != "" {
		config["nrc-pack"] = value
	} else {
		config["nrc-pack"] = "norisk-prod"
	}

	return config
}
