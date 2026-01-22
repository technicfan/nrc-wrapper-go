package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Minecraft struct {
	Profile       string
	Version       string
	Loader        string
	LoaderVersion string
	Username      string
	Uuid          string
	Token         string
}

// Prism Launcher

type PrismData struct {
	Accounts []struct {
		Active  any `json:"active"`
		Profile struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"profile"`
		Type string `json:"type"`
		Ygg  struct {
			Token string `json:"token"`
		} `json:"ygg"`
	} `json:"accounts"`
	FormatVersion int `json:"formatVersion"`
}

func (data *PrismData) get_active() (string, string, string, error) {
	var token string
	for _, v := range data.Accounts {
		if v.Active != nil && v.Active.(bool) {
			if v.Type == "Offline" {
				token = "offline"
			} else {
				token = v.Ygg.Token
			}
			return token, v.Profile.Name, v.Profile.Id, nil
		}
	}

	return "", "", "", errors.New("No active account found")
}

type PrismInstance struct {
	Components []struct {
		Uid     string `json:"uid"`
		Version string `json:"version"`
	} `json:"components"`
}

func (instance *PrismInstance) get_details() (string, string, string) {
	var version, loader, loader_version string
	for _, entry := range instance.Components {
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
	return version, loader, loader_version
}

func get_prism_details(
	path string,
) (Minecraft, error) {
	var profile, version, loader, loader_version, token, username, uuid string

	file, err := os.OpenFile("../mmc-pack.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return Minecraft{}, err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return Minecraft{}, err
	}
	defer file.Close()

	var instance PrismInstance
	err = json.Unmarshal(content, &instance)
	if err != nil {
		return Minecraft{}, err
	}
	version, loader, loader_version = instance.get_details()

	file, err = os.OpenFile("../instance.cfg", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return Minecraft{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if match, _ := regexp.MatchString("^name=.*", scanner.Text()); match {
			profile = regexp.MustCompile("^name=").ReplaceAllString(scanner.Text(), "")
		}
	}

	file, err = os.Open(fmt.Sprintf("%s/accounts.json", path))
	if err != nil {
		return Minecraft{}, err
	}
	defer file.Close()

	content, err = io.ReadAll(file)
	if err != nil {
		return Minecraft{}, err
	}

	var data PrismData
	err = json.Unmarshal(content, &data)
	if err != nil {
		return Minecraft{}, err
	}
	token, username, uuid, err = data.get_active()
	if err != nil {
		return Minecraft{}, err
	}

	return Minecraft{
		profile,
		version,
		loader,
		loader_version,
		username,
		uuid,
		token,
	}, nil
}

// Modrinth App

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
