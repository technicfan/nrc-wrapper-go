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

type token_store map[string]token_data

type token_data struct {
	Prod *string `json:"prod"`
	Exp *string `json:"exp"`
}

func token_store_from_old(old_store map[string]string) token_store {
	token_store := make(token_store)
	for uuid, token := range old_store {
		token_store[uuid] = token_data{&token, nil}
	}
	return token_store
}

func token_store_from_file(path string) (token_store, error) {
	file, err := os.Open(filepath.Join(path, globals.TOKEN_STORE))
	if err != nil {
		return token_store{}, err
	}

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return token_store{}, err
	}

	var data token_store
	json.Unmarshal(byte_data, &data)
	if data.empty() {
		var old_data map[string]string
		json.Unmarshal(byte_data, &old_data)
	    data = token_store_from_old(old_data)
	}
	return data, nil
}

func (store token_store) empty() bool {
	for _, data := range store {
		if data.Prod != nil || data.Exp != nil {
			return false
		}
	}
	return true
}

func (store token_store) get_token(
	uuid string,
	exp bool,
) (string, error) {
	if data, e := store[uuid]; e {
		if exp && data.Exp != nil {
			return *data.Exp, nil
		}
		if !exp && data.Prod != nil {
			return *data.Prod, nil
		}
	}
	return "", errors.New("requested token is not cached")
}

func (store token_store) add(
	uuid string,
	token string,
	exp bool,
) {
	data, e := store[uuid]
	if !e {
		data = token_data{}
	}
	if exp {
		data.Exp = &token
	} else {
		data.Prod = &token
	}
	store[uuid] = data
}

func (store token_store) write_store(path string) error {
	file, err := os.OpenFile(
		filepath.Join(path, globals.TOKEN_STORE), os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm,
	)
	if err != nil {
		return err
	}
	defer file.Close()

	json_string, err := json.MarshalIndent(store, "", "    ")
	if err != nil {
		return err
	}

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

	var nrc_token string
	token_store, err := token_store_from_file(config.Dir())
	if err == nil {
		nrc_token, err = token_store.get_token(uuid, config.Staging())
	}
	if err == nil {
		if result, err := is_token_expired(nrc_token); !result && err == nil {
			if config.Staging() {
				log.Println("Stored token (exp) is valid")
			} else {
				log.Println("Stored token is valid")
			}
			return nrc_token, nil
		}
	}

	if offline {
		return "offline", nil
	}

	if config.Staging() {
		log.Println("Requesting new token (exp)")
	} else {
		log.Println("Requesting new token")
	}
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

	token_store.add(uuid, nrc_token, config.Staging())
	err = token_store.write_store(config.Dir())
	if err != nil {
		return "", err
	}
	return nrc_token, nil
}
