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
