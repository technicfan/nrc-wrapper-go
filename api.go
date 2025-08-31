package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Asset struct {
    Hash string
    Size int
}

type Assets struct {
    Objects map[string]Asset
}

type Mod struct {
	id string
	name string
	source map[string]string
	compatibility map[string]map[string]*string
}

type Loader struct {
	Default map[string]map[string]string
	Minecraft []string
}

type Pack struct {
	name string
	desc string
	inherits []*string
	exclude []*string
	mods []Mod
	asset []string
	experimental bool
	auto_update bool
	loader *Loader
}

type Versions struct {
	Packs map[string]Pack
	Repositories map[string]string
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

func download_single_asset(id string, path string, metadata Asset, token string) {

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

func validate(username string, server_id string) (string, error) {
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

	var data Asset
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		log.Fatal(err)
		return "", err
	}

	return data.Hash, nil
}

func join_server_session(token string, selected_profile string, server_id string) {

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
