package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gen2brain/beeep"
)

func calc_hash(
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

func cmp_mc_versions(
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

func notify(
	msg string,
	error bool,
	notify bool,
) {
	beeep.AppName = "nrc-wrapper-go"
	if error {
		if notify {
			err := beeep.Notify("Error", msg, "")
			if err != nil {
				log.Fatalf("Notify failed: %s", err.Error())
			}
		}
		log.Fatal(msg)
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
