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
			config["launcher-dir"] = value
		} else {
			config["launcher-dir"] = filepath.Join(home, PRISM_DIR)
		}
	case "modrinth":
		if value := os.Getenv("MODRINTH_DIR"); value != "" {
			config["launcher-dir"] = value
		} else {
			config["launcher-dir"] = filepath.Join(home, MODRINTH_DIR)
		}
	default:
		log.Fatal("No valid launcher detected or set manually")
	}

	if value := os.Getenv("NRC_PACK"); value != "" {
		config["nrc-pack"] = value
	} else {
		config["nrc-pack"] = "norisk-prod"
	}

	v, l, lv, err := get_minecraft_details(config["launcher-dir"], config["launcher"])
	if err != nil {
		log.Fatalf("Failed to get Minecraft details: %s", err.Error())
	}
	config["mc-version"], config["loader"], config["loader-version"] = v, l, lv

	if value := os.Getenv("NRC_MODS_DIR"); value != "" {
		config["mods-dir"] = value
	} else if config["loader"] == "fabric" {
		config["mods-dir"] = "mods/NoRiskClient"
	} else {
		config["mods-dir"] = "mods"
	}

	config["error-on-failed-download"] = os.Getenv("NO_ERROR_ON_FAILED_DOWNLOAD")

	return config
}
