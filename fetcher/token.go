package fetcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"main/api"
	"main/config"
	"main/globals"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func is_token_expired(
	token_string string,
) (bool, error) {
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

func read_token_from_file(
	path string,
	uuid string,
) (string, error) {
	file, err := os.Open(filepath.Join(path, globals.TOKEN_STORE))
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

func write_token_to_file(
	path string,
	uuid string,
	token string,
) error {
	var file *os.File
	var err error
	var data map[string]string
	file, err = os.Open(filepath.Join(path, globals.TOKEN_STORE))
	if err != nil {
		file, err = os.Create(filepath.Join(path, globals.TOKEN_STORE))
		if err != nil {
			return err
		}
	} else {
		byte_data, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		json.Unmarshal(byte_data, &data)
	}
	defer file.Close()

	if data == nil {
		data = make(map[string]string)
	}

	data[uuid] = token

	json_string, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	file, err = os.OpenFile(
		filepath.Join(path, globals.TOKEN_STORE), os.O_RDWR|os.O_TRUNC, os.ModePerm,
	)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(string(json_string))
	if err != nil {
		return err
	}

	return nil
}

func GetToken(
	config config.Config,
	offline bool,
) (string, error) {
	uuid := config.Uuid()
	if !strings.Contains(uuid, "-") {
		uuid = fmt.Sprintf("%s-%s-%s-%s-%s",
			uuid[0:8],
			uuid[8:12],
			uuid[12:16],
			uuid[16:20],
			uuid[20:32],
		)
	}

	if config.Token() == "offline" {
		return config.Token(), nil
	}

	nrc_token, err := read_token_from_file(config.Dir(), uuid)
	if err == nil {
		if result, err := is_token_expired(nrc_token); !result && err == nil {
			log.Println("Stored token is valid")
			return nrc_token, nil
		}
	}

	if offline {
		return "offline", nil
	}

	log.Println("Requesting new token")
	server_id, err := api.RequestServerId(config.ApiEndpoint())
	if err != nil {
		return "", err
	}
	err = api.JoinServerSession(config.Token(), uuid, server_id)
	if err != nil {
		return "", err
	}

	host, _ := os.Hostname()
	system_id := fmt.Sprintf("%s-%s-%s-%s-%s", config.Id(), config.Container(), runtime.GOOS, runtime.GOARCH, host)
	hash := sha256.Sum256([]byte(system_id))
	nrc_token, err = api.RequestToken(
		config.Username(),
		server_id,
		hex.EncodeToString(hash[:]),
		config.ApiEndpoint(),
	)
	if err != nil {
		return "", err
	}

	err = write_token_to_file(config.Dir(), uuid, nrc_token)
	if err != nil {
		return "", err
	}
	return nrc_token, nil
}
