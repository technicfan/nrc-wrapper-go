package launchers

import (
	"errors"
	"fmt"
	"io/fs"
	"main/globals"
	"main/platform"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type NrcConfig struct {
	Nrc     bool
	Command string
	NrcPack string
	ModDir  string
	Notify  bool
	Neofd   bool
}

type TestInstance interface {
	Save() error
}

type Instance struct {
	Name          string
	Version       string
	Loader        string
	LoaderVersion string
	Path          string
	McRoot        string
	Launcher      string
	Cfg           Cfg
	Env           map[string]string
	FlatpakId     string
	Config        NrcConfig
	NewConfig     NrcConfig
}

func (instance *Instance) Save(ex string) error {
	if instance.Config != instance.NewConfig {
		if instance.Config.Nrc != instance.NewConfig.Nrc {
			instance.Config.Nrc = instance.NewConfig.Nrc
			if instance.Config.Nrc {
				if instance.Config.Command == "" {
					instance.Config.Command = ex
				} else {
					instance.Config.Command += " " + ex
				}
				if instance.FlatpakId != "" {
					cmd := exec.Command(
						"flatpak", "override", "--user", "--show", instance.FlatpakId,
					)
					if o, err := cmd.Output(); err == nil &&
						!strings.Contains(string(o), instance.Config.Command) {
						cmd = exec.Command(
							"flatpak", "override", "--user",
							fmt.Sprintf("--filesystem=%s", ex), instance.FlatpakId,
						)
						cmd.Run()
					}
				}
			} else {
				if instance.Config.Command == ex || instance.Config.Command == filepath.Base(ex) {
					instance.Config.Command = ""
				} else {
					cmd := ex
					if !strings.Contains(instance.Config.Command, ex) {
						cmd = filepath.Base(ex)
					}
					instance.Config.Command = strings.TrimSpace(
						strings.ReplaceAll(instance.Config.Command, cmd, ""),
					)
				}
				delete(instance.Env, "NRC_PACK")
				instance.Config.NrcPack = ""
				delete(instance.Env, "NRC_MOD_DIR")
				instance.Config.ModDir = ""
				delete(instance.Env, "NOTIFY")
				instance.Config.Notify = instance.Launcher == "modrinth"
				delete(instance.Env, "NEOFD")
				delete(instance.Env, "NO_ERROR_ON_FAILED_DOWNLOAD")
				instance.Config.Neofd = false
			}
		}
		if instance.Config.Nrc {
			if instance.Config.NrcPack != instance.NewConfig.NrcPack {
				instance.Config.NrcPack = instance.NewConfig.NrcPack
				if instance.Config.NrcPack == "" || instance.Config.NrcPack == globals.DEFAULT_PACK {
					delete(instance.Env, "NRC_PACK")
				} else {
					instance.Env["NRC_PACK"] = instance.Config.NrcPack
				}
			}
			if instance.Config.ModDir != instance.NewConfig.ModDir {
				instance.Config.ModDir = instance.NewConfig.ModDir
				if instance.Config.ModDir != "" && instance.Config.ModDir != globals.DEFAULT_MOD_DIR {
					instance.Env["NRC_MOD_DIR"] = instance.Config.ModDir
				} else {
					delete(instance.Env, "NRC_MOD_DIR")
				}
			}
			if instance.Config.Notify != instance.NewConfig.Notify {
				instance.Config.Notify = instance.NewConfig.Notify
				if instance.Config.Notify {
					if instance.Launcher == "modrinth" {
						delete(instance.Env, "NOTIFY")
					} else {
						instance.Env["NOTIFY"] = "true"
					}
				} else {
					instance.Env["NOTIFY"] = "false"
				}
			}
			if instance.Config.Neofd != instance.NewConfig.Neofd {
				instance.Config.Neofd = instance.NewConfig.Neofd
				if instance.Config.Neofd {
					instance.Env["NO_ERROR_ON_FAILED_DOWNLOAD"] = "true"
				} else {
					delete(instance.Env, "NO_ERROR_ON_FAILED_DOWNLOAD")
				}
			}
		}
		var err error
		switch instance.Launcher {
		case "prism":
			err = save_prism_instance(instance)
		case "modrinth":
			err = save_modrinth_instance(instance)
		default:
			return fmt.Errorf("%s is an invalid launcher", instance.Launcher)
		}
		if err != nil {
			return err
		}
		instance.NewConfig = instance.Config
	}
	return nil
}

func Get_instances(
	path string,
	launcher string,
	flatpak string,
	versions []string,
	loaders []string,
	ex string,
) ([]Instance, error) {
	if strings.Contains(launcher, "Modrinth App") {
		return get_modrinth_instances(path, flatpak, versions, loaders, ex)
	} else {
		return get_prism_instances(path, flatpak, versions, loaders, ex)
	}
}

func Get_launcher_dirs() (map[string][]string, []string) {
	dirs, order := platform.Get_const_dirs()
	for i, l := range order {
		if l == "" {
			continue
		}
		_, err := os.Stat(dirs[l][0])
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			order = slices.Delete(order, i, i+1)
		} else if err == nil && strings.HasPrefix(l, "Prism") {
			dir, err := Get_prism_instance_dir(dirs[l][0])
			if dirs["Prism Launcher"][0] == dir {
				i := slices.Index(order, "Prism Launcher")
				order = slices.Delete(order, i, i+1)
			}
			if err != nil {
				order = slices.Delete(order, i, i+1)
			}
			dirs[l][0] = dir
		}
	}

	return dirs, order
}
