package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"main/globals"
	"main/platform"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gen2brain/beeep"
)

func Hash(
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

func Unique(str string, index int) string {
	var builder strings.Builder
	builder.WriteString(str)
	for range index {
		builder.WriteRune('\u200d')
	}
	return builder.String()
}

func LauncherDir(
	home string,
	flatpak bool,
	id string,
	dir string,
) string {
	data_home := os.Getenv("XDG_DATA_HOME")
	if data_home == "" {
		if flatpak {
			data_home = filepath.Join(".var/app", id, "data")
		} else {
			data_home = platform.DATA_HOME
		}
		data_home = filepath.Join(home, data_home)
	}
	return filepath.Join(data_home, dir)
}

func CmpVersions(
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
