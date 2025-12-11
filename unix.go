//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
	"os"
)

const (
	DATA_HOME = ".local/share"
)

func cli() {}

func Exec(command string, args []string) error {
	err := unix.Exec(command, args, os.Environ())
	return err
}
