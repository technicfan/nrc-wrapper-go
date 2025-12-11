//go:build windows

package main

import (
	"os"
	"os/exec"

	"github.com/kolesnikovae/go-winjob"
	"golang.org/x/sys/windows"
)

const (
	DATA_HOME = "AppData/Roaming"
)

func cli() {
	const ATTACH_PARENT_PROCESS = ^uintptr(0)

	windows.NewLazyDLL("kernel32.dll").NewProc("AttachConsole").Call(ATTACH_PARENT_PROCESS)

	stdoutHandle, _ := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	stderrHandle, _ := windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	os.Stdout = os.NewFile(uintptr(stdoutHandle), "/dev/stdout")
	os.Stderr = os.NewFile(uintptr(stderrHandle), "/dev/stderr")
}

func Exec(command string, args []string) error {
	cmd := exec.Command(command, args[1:]...)
	cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout

	job, err := winjob.Start(cmd, winjob.LimitKillOnJobClose, winjob.LimitBreakawayOK)
	if err != nil {
		return err
	}
	defer job.Close()

	if err := cmd.Wait(); err != nil {
		return err
	}

	return err
}
