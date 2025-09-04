package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func is_token_expired(token_string string) (bool, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(token_string, jwt.MapClaims{})

	if err != nil {
		return false, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		exp := int64(claims["exp"].(float64))
		current := time.Now().Unix()

		return current > exp, nil
	}

	return false, errors.New("invalid token")
}

func read_token_from_file(path string, uuid string) (string, error) {
	file, err := os.Open(fmt.Sprintf("%s/norisk_data.json", path))
	if err != nil {
		return "", err
	}

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	var data map[string]string
	json.Unmarshal(byte_data, &data)

	token, exists := data[uuid]
	if exists {
		return token, nil
	}

	return "", errors.New("uuid not cached")
}

func write_token_to_file(path string, uuid string, token string) {
	var file *os.File
	var err error
	var data map[string]string
	file, err = os.OpenFile(fmt.Sprintf("%s/norisk_data.json", path), os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(fmt.Sprintf("%s/norisk_data.json", path))
		if err != nil {
			return
		}
		data = make(map[string]string)
	} else {
		byte_data, err := io.ReadAll(file)
		if err != nil {
			return
		}

		json.Unmarshal(byte_data, &data)
	}
	defer file.Close()

	if _, exists := data[uuid]; !exists {
		data[uuid] = token

		json_string, err := json.Marshal(data)
		if err != nil {
			return
		}

		_, err = file.WriteString(string(json_string))
		if err != nil {
			log.Println(err)
			log.Fatal("failed to write data")
		}
	}
}

func get_prism_data(path string) (string, string, string, error) {
	file, err := os.Open(fmt.Sprintf("%s/accounts.json", path))
	if err != nil {
		return "", "", "", err
	}
	defer file.Close()

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return "", "", "", err
	}

	var data PrismData
	err = json.Unmarshal(byte_data, &data)
	if err != nil {
		return "", "", "", err
	}

	for _, v := range data.Accounts {
		if v.Active != nil && *v.Active {
			if v.Type == "Offline" {
				return "offline", v.Profile.Name, v.Profile.Id, nil
			} else {
				return v.Ygg.Token, v.Profile.Name, v.Profile.Id, nil
			}
		}
	}

	return "", "", "", errors.New("no active account found")
}

func get_token(prism_data string) (string, error) {
	var err error
	var token, name, uuid string
	token, name, uuid, err = get_prism_data(prism_data)
	if err != nil {
		return "", err
	}

	if token == "offline" {
		return token, nil
	}

	nrc_token, err := read_token_from_file(prism_data, uuid)
	if err == nil {
		if result, err := is_token_expired(nrc_token); !result && err == nil {
			log.Print("stored token is valid")
			return nrc_token, nil
		}
	}

	log.Print("requesting new token")
	server_id, err := request_server_id()
	if err != nil {
		return "", err
	}
	join_server_session(token, uuid, server_id)
	nrc_token, err = request_token(name, server_id)
	if err != nil {
		return "", err
	}
	write_token_to_file(prism_data, uuid, nrc_token)
	return nrc_token, nil
}
