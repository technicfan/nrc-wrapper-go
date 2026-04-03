package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"main/assets"
	"main/globals"
	"main/packs"
	"net/http"
	"sync"
)

type Versions struct {
	Packs        packs.Packs       `json:"packs"`
	Repositories map[string]string `json:"repositories"`
}

type ServerId struct {
	Id string `json:"serverId"`
}

func RequestToken(
	username string,
	server_id string,
	hwid string,
	api_endpoint string,
) (string, error) {
	response, err := http.Post(
		fmt.Sprintf(
			"%s/launcher/auth/validate/v2?force=true&hwid=%s&username=%s&server_id=%s",
			api_endpoint,
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

func RequestServerId(api_endpoint string) (string, error) {
	response, err := http.Post(
		fmt.Sprintf("%s/launcher/auth/request-server-id", api_endpoint),
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

func JoinServerSession(
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
		fmt.Sprintf("%s/session/minecraft/join", globals.MOJANG_SESSION_ENDPOINT),
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

func GetVersions(api_endpoint string) (Versions, error) {
	response, err := http.Get(fmt.Sprintf("%s/launcher/modpacks-v3", api_endpoint))
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

func GetAssets(
	index int,
	pack string,
	api_endpoint string,
	wg *sync.WaitGroup,
	data chan<- map[int]map[string]assets.Asset,
) {
	defer wg.Done()

	response, err := http.Get(fmt.Sprintf("%s/launcher/pack/%s", api_endpoint, pack))
	if err != nil {
		return
	}
	if response.StatusCode != http.StatusOK {
		return
	}
	defer response.Body.Close()

	var pack_data assets.Assets
	if err := json.NewDecoder(response.Body).Decode(&pack_data); err != nil {
		return
	}

	data <- map[int]map[string]assets.Asset{index: pack_data.Assets(pack)}
}
