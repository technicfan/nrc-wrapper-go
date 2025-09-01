package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Asset struct {
	Hash string `json:"hash"`
	Size int `json:"size"`
}

type Assets struct {
	Objects map[string]Asset `json:"objects"`
}

type ServerId struct {
	Id string `json:"serverId"`
	Duration int `json:"expiresIn"`
}

type Mod struct {
	Id string `json:"id"`
	Name string `json:"displayName"`
	Source map[string]string `json:"source"`
	Compatibility map[string]map[string]*string `json:"compatibility"`
}

type Loader struct {
	Default map[string]map[string]string `json:"default"`
	Minecraft []string `json:"byMinecraft"`
}

type Pack struct {
	Name string `json:"displayName"`
	Desc string `json:"description"`
	Inherits []*string `json:"inheritsFrom"`
	Exclude []*string `json:"excludeMods"`
	Mods []Mod `json:"mods"`
	Assets []string `json:"assets"`
	Experimental bool `json:"isExperimental"`
	Auto_update bool `json:"autoUpdate"`
	Loader *Loader `json:"loaderPolicy"`
}

type Versions struct {
	Packs map[string]Pack `json:"packs"`
	Repositories map[string]string `json:"repositories"`
}

type ModFile struct {
	Hashes map[string]string `json:"hashes"`
	Url string `json:"url"`
	Filename string `json:"filename"`
	Primary bool `json:"primary"`
	Size int `json:"size"`
	File_type *string `json:"file_type"`
}

type ModrinthMod struct {
	Versions []string `json:"game_versions"`
	Loaders []string `json:"loaders"`
	Id string `json:"id"`
	Project_id string `json:"project_id"`
	Author_id string `json:"author_id"`
	Featured bool `json:"featured"`
	Name string `json:"name"`
	Version string `json:"version_number"`
	Changelog string `json:"changelog"`
	Changelog_url *string `json:"changelog_url"`
	Date string `json:"date_published"`
	Downloads int `json:"downloads"`
	Version_type string `json:"version_type"`
	Status string `json:"status"`
	Requested_status *string `json:"requested_status"`
	Files map[string]ModFile `json:"files"`
	Dependencies any `json:"dependencies"`
}



func download_jar(url string, name string) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatal(response.StatusCode)
		return
	}

	file, err := os.Create(fmt.Sprintf("./mods/%s", name))
	if err != nil  {
		log.Fatal(err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func download_single_asset(id string, path string, metadata Asset, token string) (error) {
	os.MkdirAll(fmt.Sprintf("NoRiskClient/assets/%s", filepath.Dir(path)), 0600)

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://cdn.norisk.gg/assets/%s/assets/%s", id, path),
		nil,
	)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, response.Body); err != nil {
		return err
	}

	if hex.EncodeToString(hash.Sum(nil)) != metadata.Hash {
		return errors.New("hash mismatch")
	}

	file, err := os.Create(fmt.Sprintf("NoRiskClient/assets/%s", path))
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, response.Body); err != nil {
		return err
	}

	return nil
}

func get_asset_metadata(id string) (Assets, error) {
	response, err := http.Get(fmt.Sprintf("https://api.norisk.gg/api/v1/launcher/pack/%s", id))
	if err != nil {
		return Assets{}, err
	}
	defer response.Body.Close()

	var metadata Assets
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		log.Fatal(err)
		return Assets{}, err
	}

	return metadata, nil
}

func request_token(username string, server_id string) (string, error) {
	params := make(map[string]string)
	params["force"] = "False"
	params["hwid"] = "null"
	params["username"] = username
	params["server_id"] = server_id
	params_str, err := json.Marshal(params)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	response, err := http.Post(
		fmt.Sprintf("%s/launcher/auth/validate/v2", NORISK_API_URL),
		"application/json",
		bytes.NewBuffer(params_str),
	)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer response.Body.Close()

	var data map[string]string
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		log.Fatal(err)
		return "", err
	}

	token, exists := data["value"]
	if exists {
		return token, nil
	}

	return "", errors.New("got no token")
}

func request_server_id() (string, error) {
	response, err := http.Post(fmt.Sprintf("%s/launcher/auth/request-server-id", NORISK_API_URL), "", bytes.NewBuffer([]byte("")))
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	var data ServerId
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		log.Fatal(err)
		return "", err
	}

	return data.Id, nil
}

func join_server_session(token string, selected_profile string, server_id string) {
	params := make(map[string]string)
	params["accessToken"] = token
	params["selectedProfile"] = selected_profile
	params["serverId"] = server_id
	params_str, err := json.Marshal(params)
	if err != nil {
		log.Fatal(err)
	}

	response, err := http.Post(
		fmt.Sprintf("%s/session/minecraft/join", MOJANG_SESSION_URL),
		"application/json",
		bytes.NewBuffer(params_str),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatal(response.StatusCode)
	}
}

func get_norisk_versions() (Versions, error) {
	response, err := http.Get(fmt.Sprintf("%s/launcher/modpacks", NORISK_API_URL))
	if err != nil {
		log.Fatal(err)
		return Versions{}, err
	}

	var versions Versions
	if err := json.NewDecoder(response.Body).Decode(&versions); err != nil {
		log.Fatal(err)
		return Versions{}, err
	}

	return versions, nil
}

func get_modrinth_versions(project string) (ModrinthMod, error) {
	response, err := http.Get(fmt.Sprintf("%s/project/%s/version", MODRINTH_API_URL, project))
	if err != nil {
		log.Fatal(err)
		return ModrinthMod{}, err
	}

	var mod ModrinthMod
	if err := json.NewDecoder(response.Body).Decode(&mod); err != nil {
		log.Fatal(err)
		return ModrinthMod{}, err
	}

	return mod, nil
}
