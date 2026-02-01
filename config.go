package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	// _ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Launcher              string
	LauncherDir           string
	NrcPack               string
	Minecraft             Minecraft
	ModDir                string
	ErrorOnFailedDownload bool
	Notify                bool
}

func get_minecraft_details(
	path string,
	launcher string,
) (Minecraft, error) {
	switch launcher {
	case "prism":
		return get_prism_details(path)
	case "modrinth":
		return get_modrinth_details(path)
	default:
		return Minecraft{}, errors.New("Minecraft details not found")
	}
}

func get_config() Config {
	var config Config
	usr, _ := user.Current()
	home := usr.HomeDir
	data_home := os.Getenv("XDG_DATA_HOME")

	if value := os.Getenv("LAUNCHER"); value != "" {
		log.Printf("Set %s manually", value)
		config.Launcher = value
	} else if _, err := os.Open("../mmc-pack.json"); err == nil {
		log.Println("Detected Prism Launcher")
		config.Launcher = "prism"
	} else if _, err := os.Open("../../app.db"); err == nil {
		log.Println("Detected Modrinth Launcher")
		config.Launcher = "modrinth"
	}

	switch os.Getenv("NOTIFY") {
	case "true", "True", "1":
		config.Notify = true
	case "false", "False", "0":
		config.Notify = false
	default:
		config.Notify = config.Launcher == "modrinth"
	}

	if data_home == "" {
		if os.Getenv("container") == "flatpak" {
			app_id := os.Getenv("FLATPAK_ID")
			if app_id != "" {
				data_home = filepath.Join(home, ".var/app", app_id, "data")
			} else {
				notify(
					"Flatpak ID not set - you have to manually set the launcher directory",
					true,
					config.Notify,
				)
			}
		} else {
			data_home = filepath.Join(home, DATA_HOME)
		}
	}

	switch config.Launcher {
	case "prism":
		if value := os.Getenv("PRISM_DIR"); value != "" {
			config.LauncherDir = value
		} else {
			config.LauncherDir = filepath.Join(data_home, "PrismLauncher")
		}
	case "modrinth":
		if value := os.Getenv("MODRINTH_DIR"); value != "" {
			config.LauncherDir = value
		} else {
			config.LauncherDir = filepath.Join(data_home, "ModrinthApp")
		}
	default:
		notify("No valid launcher detected or set manually", true, config.Notify)
	}

	if value := os.Getenv("NRC_PACK"); value != "" {
		config.NrcPack = value
	} else {
		config.NrcPack = "norisk-prod"
	}

	minecraft, err := get_minecraft_details(config.LauncherDir, config.Launcher)
	if err != nil {
		notify(
			fmt.Sprintf("Failed to get Minecraft details: %s", err.Error()),
			true,
			config.Notify,
		)
	}
	config.Minecraft = minecraft

	if value := os.Getenv("NRC_MOD_DIR"); value != "" {
		config.ModDir = value
	} else if config.Minecraft.Loader == "fabric" {
		config.ModDir = "mods/NoRiskClient"
	} else {
		config.ModDir = "mods"
	}

	config.ErrorOnFailedDownload = os.Getenv("NO_ERROR_ON_FAILED_DOWNLOAD") == "" && os.Getenv("NEOFD") == ""

	return config
}
