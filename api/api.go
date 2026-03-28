package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"main/globals"
	"main/packs"
	"net/http"
)

type Versions struct {
	Packs        packs.Packs             `json:"packs"`
	Repositories map[string]string `json:"repositories"`
}

type ServerId struct {
	Id string `json:"serverId"`
}

func Request_token(
	username string,
	server_id string,
	hwid string,
) (string, error) {
	response, err := http.Post(
		fmt.Sprintf(
			"%s/launcher/auth/validate/v2?force=true&hwid=%s&username=%s&server_id=%s",
			globals.NORISK_API_URL,
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

func Request_server_id() (string, error) {
	response, err := http.Post(
		fmt.Sprintf("%s/launcher/auth/request-server-id", globals.NORISK_API_URL),
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

func Join_server_session(
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
		fmt.Sprintf("%s/session/minecraft/join", globals.MOJANG_SESSION_URL),
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

func Get_norisk_versions() (Versions, error) {
	response, err := http.Get(fmt.Sprintf("%s/launcher/modpacks-v3", globals.NORISK_API_URL))
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
