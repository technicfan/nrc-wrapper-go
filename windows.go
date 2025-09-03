//go:build windows

package main

import (
	"os"
	"os/exec"
)

const (
	PRISM_DIR = "AppData/Roaming/PrismLauncher"
)

func Exec(command string, args []string) error {
	cmd := exec.Command(command, args[1:]...)
	cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout
	err := cmd.Run()

	return err
}
