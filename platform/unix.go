//go:build !windows

package platform

import (
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

const (
	DATA_HOME = ".local/share"
	WINDOWS = false
)

func Cli() {}

func Exec(command string, args []string) error {
	err := unix.Exec(command, args, os.Environ())
	return err
}

func IsRunning(pname string) bool {
	cmd := exec.Command("pgrep", "-x", pname)
	return cmd.Run() == nil
}

func RunningLaunchers() []string {
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
