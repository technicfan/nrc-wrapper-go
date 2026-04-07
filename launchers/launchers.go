package launchers

import (
	"errors"
	"main/platform"
	"os"
)

var LAUNCHERS = map[string]LauncherSupport{
	PRISM_ID:    LauncherSupport{NewPrismLauncher, PRISM_NAME, PRISM_CLASS},
	MODRINTH_ID: LauncherSupport{NewModrinthApp, MODRINTH_NAME, MODRINTH_CLASS},
}

type LauncherSupport struct {
	New       func(string, string, bool) Launcher
	Name      string
	JavaClass string
}

type Launcher interface {
	Id() string
	Name() string
	Dir() string
	Container() string
	InstanceDir() string
	GetCurrentInstanceDetails() (Minecraft, error)
	GetInstances(map[string][]string, string) ([]Instance, error)
	Exists() bool
	IsRunning() bool
	DefaultNotify() bool
	MergeNormal() Launcher
}

type launcher_data struct {
	name         string
	path         string
	instance_dir string
	flatpak_id   string
	replaced     bool
}

func (data launcher_data) Name() string {
	return data.name
}

func (data launcher_data) Dir() string {
	return data.path
}

func (data launcher_data) Container() string {
	if data.flatpak_id != "" {
		return "flatpak"
	} else {
		return ""
	}
}

func (data launcher_data) InstanceDir() string {
	return data.instance_dir
}

func (data launcher_data) ReplaceNormal() {
	if data.flatpak_id != "" {
		data.replaced = true
	}
}

func appendLaunchers(
	launchers []Launcher,
	ctor func(string, string, bool) Launcher,
	home string,
) []Launcher {
	regular := ctor(home, "", false)
	if platform.WINDOWS {
		if regular.Exists() {
			launchers = append(launchers, regular)
		}
	} else {
		flatpak := ctor(home, "", true)
		if regular.Exists() && flatpak.Exists() && regular.InstanceDir() == flatpak.InstanceDir() {
			launchers = append(launchers, flatpak.MergeNormal())
		} else {
			if regular.Exists() {
				launchers = append(launchers, regular)
			}
			if flatpak.Exists() {
				launchers = append(launchers, flatpak)
			}
		}
	}
	return launchers
}

func GetLaunchers() ([]Launcher, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var launchers []Launcher
	for _, l := range LAUNCHERS {
		launchers = appendLaunchers(launchers, l.New, home)
	}
	if len(launchers) == 0 {
		return nil, errors.New("No launchers installed")
	}
	return launchers, nil
}
