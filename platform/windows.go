//go:build windows

package platform

import (
	"os"
	"os/exec"
	"strings"

	"github.com/kolesnikovae/go-winjob"
)

const (
	DATA_HOME = "AppData/Roaming"
	WINDOWS = true
)

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
