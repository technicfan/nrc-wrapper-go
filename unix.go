//go:build !windows

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	PRISM_DIR = ".local/share/PrismLauncher"
)

func Exec(command string, args []string) error {
	err := unix.Exec(command, args, os.Environ())
	return err
}
