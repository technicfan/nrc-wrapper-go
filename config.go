package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func get_minecraft_details(
	path string,
	launcher string,
) (string, string, string, error) {
	var version, loader, loader_version string

	switch launcher {
	case "prism":
		file, err := os.OpenFile("../mmc-pack.json", os.O_RDONLY, os.ModePerm)
		if err != nil {
			return "", "", "", err
		}
		content, err := io.ReadAll(file)
		if err != nil {
			return "", "", "", err
		}
		defer file.Close()

		var data PrismInstance
		err = json.Unmarshal(content, &data)
		if err != nil {
			return "", "", "", err
		}

		for _, entry := range data.Components {
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

	case "modrinth":
		db, err := sql.Open("sqlite3", fmt.Sprintf("%s/app.db", path))
		if err != nil {
			return "", "", "", err
		}
		defer db.Close()

		cwd, err := os.Getwd()
		if err != nil {
			return "", "", "", err
		}
		rows, err := db.Query(
			fmt.Sprintf(
				"SELECT game_version, mod_loader, mod_loader_version FROM profiles WHERE path = '%s'",
				filepath.Base(cwd),
			),
		)
		if err != nil {
			return "", "", "", err
		}
		defer rows.Close()

		for rows.Next() {
			err = rows.Scan(&version, &loader, &loader_version)
			if err != nil {
				return "", "", "", err
			}
		}
	default:
		return "", "", "", errors.New("Minecraft details not found")
	}

	return version, loader, loader_version, nil
}

func get_config() Config {
	var config Config
	usr, _ := user.Current()
	home := usr.HomeDir

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

	switch (os.Getenv("NOTIFY")) {
	case "true", "True", "1":
		config.Notify = true			
	case "false", "False", "0":
		config.Notify = false
	default:
		config.Notify = config.Launcher == "modrinth"
	}

	switch config.Launcher {
	case "prism":
		if value := os.Getenv("PRISM_DIR"); value != "" {
			config.LauncherDir = value
		} else {
			config.LauncherDir = filepath.Join(home, PRISM_DIR)
		}
	case "modrinth":
		if value := os.Getenv("MODRINTH_DIR"); value != "" {
			config.LauncherDir = value
		} else {
			config.LauncherDir = filepath.Join(home, MODRINTH_DIR)
		}
	default:
		notify("No valid launcher detected or set manually", true, config.Notify)
	}

	if value := os.Getenv("NRC_PACK"); value != "" {
		config.NrcPack = value
	} else {
		config.NrcPack = "norisk-prod"
	}

	v, l, lv, err := get_minecraft_details(config.LauncherDir, config.Launcher)
	if err != nil {
		notify(
			fmt.Sprintf("Failed to get Minecraft details: %s", err.Error()),
			true,
			config.Notify,
		)
	}
	config.McVersion, config.Loader, config.LoaderVersion = v, l, lv

	if value := os.Getenv("NRC_MOD_DIR"); value != "" {
		config.ModDir = value
	} else if config.Loader == "fabric" {
		config.ModDir = "mods/NoRiskClient"
	} else {
		config.ModDir = "mods"
	}

	config.ErrorOnFailedDownload = os.Getenv("NO_ERROR_ON_FAILED_DOWNLOAD") == ""

	return config
}
