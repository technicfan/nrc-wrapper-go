//go:build !windows

package main

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	DATA_HOME = ".local/share"
)

func cli() {}

func get_launcher_dirs() map[string][]string {
	usr, _ := user.Current()
	home := usr.HomeDir
	dirs := map[string][]string{
		"Modrinth App" : {filepath.Join(home, DATA_HOME, "ModrinthApp"), ""},
		"Modrinth App (Flatpak)": {filepath.Join(home, ".var/app/com.modrinth.ModrinthApp/data/ModrinthApp"), "com.modrinth.ModrinthApp"},
		"Prism Launcher": {filepath.Join(home, DATA_HOME, "PrismLauncher"), ""},
		"Prism Launcher (Flatpak)": {filepath.Join(home, ".var/app/org.prismlauncher.PrismLauncher/data/PrismLauncher"), "org.prismlauncher.PrismLauncher"},
	}
	for l := range dirs {
		_, err := os.Stat(dirs[l][0])
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			delete(dirs, l)
		} else if err == nil && strings.HasPrefix(l, "Prism") {
			dir, err := get_prism_instance_dir(dirs[l][0])
			if dirs["Prism Launcher"][0] == dir {
				delete(dirs, "Prism Launcher")
			}
			if err != nil {
				delete(dirs, l)
			}
			dirs[l] = []string{dir, dirs[l][1]}
		}
	}

	return dirs
}

func Exec(command string, args []string) error {
	err := unix.Exec(command, args, os.Environ())
	return err
}
