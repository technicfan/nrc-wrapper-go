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
	token, err := jwt.Parse(token_string, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok && token.Method.Alg() != "none" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return nil, nil
	})

	if err != nil {
		return false, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		expTime := int64(claims["exp"].(float64))
		currentTime := time.Now().Unix()

		if currentTime > expTime {
			log.Print("Stored Token is expired")
			return true, nil
		} else {
			log.Print("Stored Token is valid")
			return false, nil
		}
	}

	return false, errors.New("invalid token")
}

func read_token_from_file(path string, uuid string) (string, error) {
	file, err := os.Open(fmt.Sprintf("%snorisk_data.json", path))
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
	created := false
	file, err = os.Open(fmt.Sprintf("%snorisk_data.json", path))
	if err != nil {
		created = true
		file, err = os.Create(fmt.Sprintf("%snorisk_data.json", path))
		if err != nil {
			return
		}
	}
	defer file.Close()

	var data map[string]string

	if !created {
		byte_data, err := io.ReadAll(file)
		if err != nil {
			return
		}

		json.Unmarshal(byte_data, &data)
	}

	data[uuid] = token

	json_string, err := json.Marshal(data)
	if err != nil {
		return
	}

	_, err = file.WriteString(string(json_string))
	if err != nil {
		log.Fatal("failed to write data")
	}
}

func get_prism_data(path string) (string, string, string, error) {
	file, err := os.Open(fmt.Sprintf("%saccounts.json", path))
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

func get_token() (string, error) {
	var err error
	var token, name, uuid string
	token, name, uuid, err = get_prism_data(PRISM_UNIX)
	if err != nil {
		return "", err
	}

	nrc_token, err := read_token_from_file(PRISM_UNIX, uuid)
	if err == nil {
		if result, err := is_token_expired(nrc_token); !result && err == nil {
			return nrc_token, nil
		}
	}

	server_id, err := request_server_id()
	if err != nil {
		return "", err
	}
	join_server_session(token, uuid, server_id)
	nrc_token, err = request_token(name, server_id)
	if err != nil {
		return "", err
	}
	write_token_to_file(PRISM_UNIX, uuid, nrc_token)
	return nrc_token, nil
}
