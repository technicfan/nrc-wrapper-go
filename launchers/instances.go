package launchers

import (
	"fmt"
	"main/globals"
	"os/exec"
	"path/filepath"
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
	LauncherClass() string
	Nrc() bool
	Staging() bool
	Notify() bool
	DefaultNotify() bool
	Neofd() bool
	Pack() string
	ModDir() string
	FixPack(string)
	Save(bool, bool, bool, string, string) error
}

type Minecraft struct {
	profile       string
	version       string
	loader        string
	loader_version string
	username      string
	uuid          string
	token         string
}

func NewMinecraft(
	instance Instance,
) Minecraft {
	return Minecraft{
		instance.Name(),
		instance.Version(),
		instance.Loader(),
		instance.LoaderVersion(),
		"", "", "",
	}
}

func (minecraft Minecraft) Profile() string {
	return minecraft.profile
}

func (minecraft Minecraft) Version() string {
	return minecraft.version
}

func (minecraft Minecraft) Loader() string {
	return minecraft.loader
}

func (minecraft Minecraft) LoaderVersion() string {
	return minecraft.loader_version
}

func (minecraft Minecraft) Username() string {
	return minecraft.username
}

func (minecraft Minecraft) Uuid() string {
	return minecraft.uuid
}

func (minecraft Minecraft) Token() string {
	return minecraft.token
}

type nrc_config struct {
	nrc     bool
	command string
	staging bool
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

func (instance instance_data) Staging() bool {
	return instance.config.staging
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
	if instance.loader == "fabric" {
		return instance.config.mod_dir
	} else {
		return "mods"
	}
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
			instance.config.pack = globals.DEFAULT_PACK
			delete(instance.env, "NRC_MOD_DIR")
			instance.config.mod_dir = globals.DEFAULT_MOD_DIR
			delete(instance.env, "STAGING")
			instance.config.staging = false
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
