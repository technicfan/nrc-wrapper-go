package main

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
)

func get_config() map[string]string {
	config := make(map[string]string)
	usr, _ := user.Current()
	home := usr.HomeDir

	if value := os.Getenv("LAUNCHER"); value != "" {
		log.Printf("Set %s manually", value)
		config["launcher"] = value
	} else if _, err := os.Open("../mmc-pack.json"); err == nil {
		log.Println("Detected Prism Launcher")
		config["launcher"] = "prism"
	} else if _, err := os.Open("../../app.db"); err == nil {
		log.Println("Detected Modrinth Launcher")
		config["launcher"] = "modrinth"
	}

	switch config["launcher"] {
	case "prism":
		if value := os.Getenv("PRISM_DIR"); value != "" {
			config["launcher_dir"] = filepath.Join(home, value)
		} else {
			config["launcher_dir"] = filepath.Join(home, PRISM_DIR)
		}
	case "modrinth":
		if value := os.Getenv("MODRINTH_DIR"); value != "" {
			config["launcher_dir"] = filepath.Join(home, value)
		} else {
			config["launcher_dir"] = filepath.Join(home, MODRINTH_DIR)
		}
	default:
		log.Fatal("No valid launcher detected or set manually")
	}

	if value := os.Getenv("NRC_PACK"); value != "" {
		config["nrc-pack"] = value
	} else {
		config["nrc-pack"] = "norisk-prod"
	}

	return config
}
