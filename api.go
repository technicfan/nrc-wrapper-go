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
	"strings"
	"sync"
)

func check_connection() bool {
	_, err := http.Get(strings.ReplaceAll(NORISK_API_URL, "/api/v1", ""))
	if err != nil {
		return false
	}

	return true
}

func download_jar(url string, name string, path string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to download %s: %v", name, response.StatusCode)
	}
	defer response.Body.Close()

	file, err := os.Create(filepath.Join(path, name))
	if err != nil  {
		return "", err
	}
	defer file.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	_, err = file.Write(body)
	if err != nil {
		return "", err
	}

	log.Printf("Downloaded %s", name)

	return name, nil
}

func download_single_asset(pack string, path string, expected_hash string, wg *sync.WaitGroup, limiter chan struct{}) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <- limiter }()

	os.MkdirAll(fmt.Sprintf("NoRiskClient/assets/%s", filepath.Dir(path)), os.ModePerm)

	response, err := http.Get(fmt.Sprintf("https://cdn.norisk.gg/assets/%s/assets/%s", pack, path))
	if err != nil {
		log.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		log.Fatalf("Failed to download %s: %v", filepath.Base(path), response.StatusCode)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	hash := md5.New()
	if _, err := hash.Write(body); err != nil {
		log.Fatal(err)
	}

	if hex.EncodeToString(hash.Sum(nil)) != expected_hash {
		log.Fatalf("%s/%s has wrong hash", pack, filepath.Base(path))
	}

	file, err := os.Create(fmt.Sprintf("NoRiskClient/assets/%s", path))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if _, err := file.Write(body); err != nil {
		log.Fatal(err)
	}

	log.Printf("Downloaded %s/%s", pack, filepath.Base(path))
}

func get_asset_metadata(index int, pack string, wg *sync.WaitGroup, data chan <- map[int]map[string]map[string]string) {
	defer wg.Done()

	response, err := http.Get(fmt.Sprintf("%s/launcher/pack/%s", NORISK_API_URL, pack))
	if err != nil {
		return
	}
	defer response.Body.Close()

	var metadata Assets
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return
	}

	results := make(map[string]map[string]string)
	for i, v := range metadata.Objects {
		asset := make(map[string]string)
		asset["hash"] = v.Hash
		asset["pack"] = pack
		results[i] = asset
	}

	data <- map[int]map[string]map[string]string{index: results}
}

func request_token(username string, server_id string, hwid string) (string, error) {
	response, err := http.Post(
		fmt.Sprintf("%s/launcher/auth/validate/v2?force=false&hwid=%s&username=%s&server_id=%s", NORISK_API_URL, hwid, username, server_id),
		"application/json",
		bytes.NewBuffer([]byte{}),
	)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", nil
	}

	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
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
		return "", err
	}
	defer response.Body.Close()

	var data ServerId
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
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
}

func get_norisk_versions() (Versions, error) {
	response, err := http.Get(fmt.Sprintf("%s/launcher/modpacks", NORISK_API_URL))
	if err != nil {
		return Versions{}, err
	}

	var versions Versions
	if err := json.NewDecoder(response.Body).Decode(&versions); err != nil {
		return Versions{}, err
	}

	return versions, nil
}

func get_modrinth_versions(project string, wg *sync.WaitGroup, results chan <- []ModrinthMod) {
	defer wg.Done()
	response, err := http.Get(fmt.Sprintf("%s/project/%s/version", MODRINTH_API_URL, project))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var mod []ModrinthMod
	if err := json.Unmarshal(body, &mod); err != nil {
		log.Fatal(err)
		return
	}

	results <- mod
}
