//go:build windows

package platform

import (
	"os"
	"os/exec"
	"strings"

	"github.com/kolesnikovae/go-winjob"
	"golang.org/x/sys/windows"
)

const (
	DATA_HOME = "AppData/Roaming"
	WINDOWS = true
)

func Cli() {
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

func IsRunning(pname string) bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq " + pname)
	out, err := cmd.Output()
	return err == nil && strings.Contains(string(out), pname)
}

func RunningLaunchers() []string {
	var running []string
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq prismlauncher.exe")
	out, err := cmd.Output()
	if err == nil && strings.Contains(string(out), "prismlauncher.exe") {
		running = append(running, "Prism Launcher")
	}
	cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq Modrinth App.exe")
	out, err = cmd.Output()
	if err == nil && strings.Contains(string(out), "Modrinth App.exe") {
		running = append(running, "Modrinth App")
	}
	return running
}
