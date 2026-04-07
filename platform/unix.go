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

func Exec(command string, args []string) error {
	err := unix.Exec(command, args, os.Environ())
	return err
}

func IsRunning(pname string) bool {
	if len(pname) > 15 {
		pname = pname[:15]
	}
	cmd := exec.Command("pgrep", "-x", pname)
	return cmd.Run() == nil
}
