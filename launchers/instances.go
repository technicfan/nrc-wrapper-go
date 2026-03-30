package launchers

import (
	"errors"
	"fmt"
	"main/globals"
	"main/platform"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Instance interface {
	Name() string
	Version() string
	Loader() string
	LoaderVersion() string
	Path() string
	Env() []string
	FlatpakId() string
	Nrc() bool
	Notify() bool
	DefaultNotify() bool
	Neofd() bool
	Pack() string
	ModDir() string
	FixPack(string)
	Save(bool, bool, bool, string, string) error
}

type nrc_config struct {
	nrc     bool
	command string
	pack string
	mod_dir  string
	notify  bool
	neofd   bool
}

type instance_data struct {
	name           string
	version        string
	loader         string
	loader_version string
	path           string
	root           string
	env            map[string]string
	flatpak_id     string
	config         nrc_config
	default_notify bool
}

func (instance instance_data) Name() string {
	return instance.name
}

func (instance instance_data) Version() string {
	return instance.version
}

func (instance instance_data) Loader() string {
	return instance.loader
}

func (instance instance_data) LoaderVersion() string {
	return instance.loader_version
}

func (instance instance_data) Path() string {
	return instance.root
}

func (instance instance_data) Env() []string {
	var env []string
	for k, v := range instance.env {
		env = append(env, k + "=" + v)
	}
	return env
}

func (instance instance_data) FlatpakId() string {
	return instance.flatpak_id
}

func (instance *instance_data) FixPack(pack string) {
	instance.config.pack = pack
}

func (instance instance_data) Nrc() bool {
	return instance.config.nrc
}

func (instance instance_data) Notify() bool {
	return instance.config.notify
}

func (instance instance_data) DefaultNotify() bool {
	return instance.default_notify
}

func (instance instance_data) Neofd() bool {
	return instance.config.neofd
}

func (instance instance_data) Pack() string {
	return instance.config.pack
}

func (instance instance_data) ModDir() string {
	return instance.config.mod_dir
}

func (instance *instance_data) save(nrc bool, notify bool, neofd bool, pack string, ex string) bool {
	var changed bool
	if instance.config.nrc != nrc {
		instance.config.nrc = nrc
		changed = true
		if instance.config.nrc {
			if instance.config.command == "" {
				instance.config.command = ex
			} else {
				instance.config.command += " " + ex
			}
			if instance.flatpak_id != "" {
				cmd := exec.Command(
					"flatpak", "override", "--user", "--show", instance.flatpak_id,
				)
				if o, err := cmd.Output(); err == nil &&
					!strings.Contains(string(o), instance.config.command) {
					cmd = exec.Command(
						"flatpak", "override", "--user",
						fmt.Sprintf("--filesystem=%s", ex), instance.flatpak_id,
					)
					cmd.Run()
				}
			}
		} else {
			if instance.config.command == ex || instance.config.command == filepath.Base(ex) {
				instance.config.command = ""
			} else {
				cmd := ex
				if !strings.Contains(instance.config.command, ex) {
					cmd = filepath.Base(ex)
				}
				instance.config.command = strings.TrimSpace(
					strings.ReplaceAll(instance.config.command, cmd, ""),
				)
			}
			delete(instance.env, "NRC_PACK")
			instance.config.pack = ""
			delete(instance.env, "NRC_MOD_DIR")
			instance.config.mod_dir = ""
			delete(instance.env, "NOTIFY")
			instance.config.notify = instance.DefaultNotify()
			delete(instance.env, "NEOFD")
			delete(instance.env, "NO_ERROR_ON_FAILED_DOWNLOAD")
			instance.config.neofd = false
		}
	}
	if instance.config.nrc {
		if instance.config.pack != pack {
			instance.config.pack = pack
			changed = true
			if instance.config.pack == "" || instance.config.pack == globals.DEFAULT_PACK {
				delete(instance.env, "NRC_PACK")
			} else {
				instance.env["NRC_PACK"] = instance.config.pack
			}
		}
		// if instance.config.ModDir != instance.NewConfig.ModDir {
		// 	instance.config.ModDir = instance.NewConfig.ModDir
		// 	if instance.config.ModDir != "" && instance.Config.ModDir != globals.DEFAULT_MOD_DIR {
		// 		instance.Env["NRC_MOD_DIR"] = instance.config.ModDir
		// 	} else {
		// 		delete(instance.Env, "NRC_MOD_DIR")
		// 	}
		// }
		if instance.config.notify != notify {
			instance.config.notify = notify
			changed = true
			if instance.config.notify {
				instance.env["NOTIFY"] = "true"
			} else {
				instance.env["NOTIFY"] = "false"
			}
		}
		if instance.config.neofd != neofd {
			instance.config.neofd = neofd
			changed = true
			if instance.config.neofd {
				instance.env["NO_ERROR_ON_FAILED_DOWNLOAD"] = "true"
			} else {
				delete(instance.env, "NO_ERROR_ON_FAILED_DOWNLOAD")
			}
		}
	}
	return changed
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
			launchers = append(launchers, flatpak)
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

func GetInstances(
	versions []string,
	loaders []string,
	ex string,
) (map[string][]Instance, []string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	var launchers []Launcher
	launchers = appendLaunchers(launchers, NewPrismLauncher, home)
	launchers = appendLaunchers(launchers, NewModrinthApp, home)
	var order []string
	instances := make(map[string][]Instance)
	for i := range launchers {
		inst, err := launchers[i].GetInstances(versions, loaders, ex)
		if err != nil {
			continue
		}
		order = append(order, launchers[i].Name())
		slices.SortFunc(inst, func(a Instance, b Instance) int {
			return strings.Compare(a.Name(), b.Name())
		})
		instances[launchers[i].Name()] = inst
	}
	if len(order) == 0 {
		return nil, nil, errors.New("No instances found")
	}
	return instances, order, nil
}
