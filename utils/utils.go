package utils

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"main/globals"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gen2brain/beeep"
)

type Resource interface {
	Url() string
	Path() string
	Filename() string
	ExpectedHash() string
	HashObj() hash.Hash
	Download() error
}

func Download(resource Resource) error {
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

func Calc_hash(
	path string,
) (string, error) {
	var file, err = os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	var hash = md5.New()
	_, err = hash.Write(data)
	if err != nil {
		return "", err
	}

	var bytesHash = hash.Sum(nil)
	return hex.EncodeToString(bytesHash[:]), nil
}

func Make_unique(str string, index int) string {
	var builder strings.Builder
	builder.WriteString(str)
	for range index {
		builder.WriteRune('\u200d')
	}
	return builder.String()
}

func Cmp_versions(
	a string,
	b string,
) int {
	if a == b {
		return 0
	}

	a_split := strings.Split(a, ".")
	b_split := strings.Split(b, ".")

	length := min(len(a_split), len(b_split))

	for i := range length {
		if a_split[i] == b_split[i] {
			continue
		}

		a_int, err := strconv.ParseInt(a_split[i], 10, 32)
		if err != nil {
			return 1
		}
		b_int, err := strconv.ParseInt(b_split[i], 10, 32)
		if err != nil {
			return -1
		}
		if a_int > b_int {
			return 1
		} else {
			return -1
		}
	}

	if len(a) > len(b) {
		return 1
	} else {
		return -1
	}
}

func Notify(
	msg string,
	error bool,
	notify bool,
) {
	beeep.AppName = "nrc-wrapper-go"
	if error {
		if notify {
			err := beeep.Notify("Error", msg, "")
			log.Println(1)
			if err != nil {
				log.Fatalf("Notify failed: %s", err.Error())
			}
		}
		if globals.REFRESH {
			log.Println(msg)
		} else {
			log.Fatal(msg)
		}
	} else {
		if notify {
			err := beeep.Notify("Info", msg, "")
			if err != nil {
				log.Fatalf("Notify failed: %s", err.Error())
			}
		}
		log.Println(msg)
	}
}
