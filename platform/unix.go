//go:build !windows

package platform

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const (
	DATA_HOME = ".local/share"
)

func Cli() {}

func Get_const_dirs() (map[string][]string, []string) {
	usr, _ := user.Current()
	home := usr.HomeDir
	dirs := map[string][]string{
		"Prism Launcher": {filepath.Join(home, DATA_HOME, "PrismLauncher"), ""},
		"Prism Launcher (Flatpak)": {filepath.Join(home, ".var/app/org.prismlauncher.PrismLauncher/data/PrismLauncher"), "org.prismlauncher.PrismLauncher"},
		"Modrinth App": {filepath.Join(home, DATA_HOME, "ModrinthApp"), ""},
		"Modrinth App (Flatpak)": {filepath.Join(home, ".var/app/com.modrinth.ModrinthApp/data/ModrinthApp"), "com.modrinth.ModrinthApp"},
	}

	return dirs, []string{"Prism Launcher", "Prism Launcher (Flatpak)", "Modrinth App", "Modrinth App (Flatpak)"}
}

func Exec(command string, args []string) error {
	err := unix.Exec(command, args, os.Environ())
	return err
}

func Get_running_launchers() []string {
	var running []string
	cmd := exec.Command("pgrep", "-x", "prismlauncher")
	err := cmd.Run()
	if err == nil {
		running = append(running, "Prism Launcher")
	}
	cmd = exec.Command("pgrep", "-x", "modrinth-app")
	err = cmd.Run()
	if err == nil {
		running = append(running, "Modrinth App")
	}
	return running
}
