package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

	return false, errors.New("Invalid token")
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
	file, err = os.OpenFile(fmt.Sprintf("%s/norisk_data.json", path), os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(fmt.Sprintf("%s/norisk_data.json", path))
		if err != nil {
			return
		}
	} else {
		byte_data, err := io.ReadAll(file)
		if err != nil {
			return
		}

		json.Unmarshal(byte_data, &data)
	}
	defer file.Close()

	if data == nil {
		data = make(map[string]string)
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

func get_minecraft_data(path string, launcher string) (string, string, string, error) {
	switch launcher {
	case "prism":
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

		return "", "", "", errors.New("No active account found")

	case "modrinth":
		db, err := sql.Open("sqlite3", fmt.Sprintf("%s/app.db", path))
		if err != nil {
			return "", "", "", err
		}
		defer db.Close()

		rows, err := db.Query("SELECT access_token, username, uuid FROM minecraft_users where active = 1")
		if err != nil {
			return "", "", "", err
		}
		defer rows.Close()

		var token, username, uuid string
		for rows.Next() {
			err = rows.Scan(&token, &username, &uuid)
			if err != nil {
				return "", "", "", err
			}
		}
		return token, username, uuid, nil
	}

	return "", "", "", errors.New("No launcher detected")
}

func get_token(config map[string]string, wg *sync.WaitGroup, out chan <- string) {
	defer wg.Done()

	var err error
	var token, name, uuid string
	token, name, uuid, err = get_minecraft_data(config["launcher_dir"], config["launcher"])
	if err != nil {
		log.Fatal(err)
	}

	if token == "offline" {
		out <- token
		return
	}

	nrc_token, err := read_token_from_file(config["launcher_dir"], uuid)
	if err == nil {
		if result, err := is_token_expired(nrc_token); !result && err == nil {
			log.Println("Stored token is valid")
			out <- nrc_token
			return
		}
	}

	log.Println("Requesting new token")
	server_id, err := request_server_id()
	if err != nil {
		log.Fatal(err)
	}
	join_server_session(token, uuid, server_id)
	nrc_token, err = request_token(name, server_id, config["launcher"])
	if err != nil {
		log.Fatal(err)
	}
	write_token_to_file(config["launcher_dir"], uuid, nrc_token)
	out <- nrc_token
}
