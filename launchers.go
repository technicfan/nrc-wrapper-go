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

func get_prism_details(
	path string,
) (Minecraft, error) {
	var profile, version, loader, loader_version, token string

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

	for _, v := range data.Accounts {
		if v.Active != nil && v.Active.(bool) {
			if v.Type == "Offline" {
				token = "offline"
			} else {
				token = v.Ygg.Token
			}
			return Minecraft{
				profile,
				version,
				loader,
				loader_version,
				v.Profile.Name,
				v.Profile.Id,
				token,
			}, nil
		}
	}

	return Minecraft{}, errors.New("No active account found")
}

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
