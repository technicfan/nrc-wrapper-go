package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Assets struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int    `json:"size"`
	} `json:"objects"`
}

type ServerId struct {
	Id string `json:"serverId"`
}

type Downloadable interface {
	Url() string
	Path() string
	Filename() string
	ExpectedHash() string
	HashObj() hash.Hash
	Download() error
}

func download(
	resource Downloadable,
) error {
	os.MkdirAll(filepath.Dir(resource.Path()), os.ModePerm)

	response, err := http.Get(resource.Url())
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %v", response.StatusCode)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	expected_hash := resource.ExpectedHash()
	if expected_hash != "" {
		hash := resource.HashObj()

		if _, err := hash.Write(body); err != nil {
			return err
		}
		if hex.EncodeToString(hash.Sum(nil)) != expected_hash {
			return errors.New("wrong hash")
		}
	}

	file, err := os.Create(resource.Path())
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return err
	}

	return nil
}

func get_asset_metadata_async(
	index int,
	pack string,
	wg *sync.WaitGroup,
	data chan<- map[int]map[string]Asset,
) {
	defer wg.Done()

	response, err := http.Get(fmt.Sprintf("%s/launcher/pack/%s", NORISK_API_URL, pack))
	if err != nil {
		return
	}
	if response.StatusCode != http.StatusOK {
		return
	}
	defer response.Body.Close()

	var metadata Assets
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return
	}

	results := make(map[string]Asset)
	for path, obj := range metadata.Objects {
		asset := Asset{pack, path, obj.Hash}
		results[path] = asset
	}

	data <- map[int]map[string]Asset{index: results}
}

func request_token(
	username string,
	server_id string,
	hwid string,
) (string, error) {
	response, err := http.Post(
		fmt.Sprintf(
			"%s/launcher/auth/validate/v2?force=true&hwid=%s&username=%s&server_id=%s",
			NORISK_API_URL,
			hwid,
			username,
			server_id,
		),
		"application/json",
		bytes.NewBuffer([]byte{}),
	)
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %v", response.StatusCode)
	}
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
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
	response, err := http.Post(
		fmt.Sprintf("%s/launcher/auth/request-server-id", NORISK_API_URL),
		"",
		bytes.NewBuffer([]byte("")),
	)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %v", response.StatusCode)
	}
	defer response.Body.Close()

	var data ServerId
	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.Id, nil
}

func join_server_session(
	token string,
	selected_profile string,
	server_id string,
) error {
	params := make(map[string]string)
	params["accessToken"] = token
	params["selectedProfile"] = selected_profile
	params["serverId"] = server_id
	params_str, err := json.Marshal(params)
	if err != nil {
		return err
	}

	response, err := http.Post(
		fmt.Sprintf("%s/session/minecraft/join", MOJANG_SESSION_URL),
		"application/json",
		bytes.NewBuffer(params_str),
	)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP %v", response.StatusCode)
	}
	defer response.Body.Close()

	return nil
}

func get_norisk_versions(
	domain string,
) (Versions, error) {
	response, err := http.Get(fmt.Sprintf("%s/launcher/modpacks-v3", domain))
	if err != nil {
		return Versions{}, err
	}
	if response.StatusCode != http.StatusOK {
		return Versions{}, fmt.Errorf("HTTP %v", response.StatusCode)
	}

	var versions Versions
	if err := json.NewDecoder(response.Body).Decode(&versions); err != nil {
		return Versions{}, err
	}

	return versions, nil
}
