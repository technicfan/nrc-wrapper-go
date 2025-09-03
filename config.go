package main

import (
	"os/user"
	"path/filepath"
	"runtime"
)

func get_config() map[string]string {
	config := make(map[string]string)
	usr, _ := user.Current()
	home := usr.HomeDir
	switch runtime.GOOS {
	case "linux":
		config["prism_data"] = filepath.Join(home, PRISM_UNIX)
	case "windows":
		config["prism_data"] = filepath.Join(home, PRISM_WIN)
	}

	return config
}
