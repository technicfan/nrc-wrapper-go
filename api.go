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
	"sync"
)

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

	file, err := os.Create(fmt.Sprintf("mods/%s", name))
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

func download_single_asset(id string, path string, metadata Asset, token string, wg *sync.WaitGroup) {
	defer wg.Done()

	os.MkdirAll(fmt.Sprintf("NoRiskClient/assets/%s", filepath.Dir(path)), 0600)

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://cdn.norisk.gg/assets/%s/assets/%s", id, path),
		nil,
	)
	if err != nil {
		return
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, response.Body); err != nil {
		return
	}

	if hex.EncodeToString(hash.Sum(nil)) != metadata.Hash {
		return
	}

	file, err := os.Create(fmt.Sprintf("NoRiskClient/assets/%s", path))
	if err != nil {
		return
	}

	if _, err := io.Copy(file, response.Body); err != nil {
		return
	}
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
