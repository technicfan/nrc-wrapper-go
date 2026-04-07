package config

import (
	"fmt"
	"log"
	"main/globals"
	"main/launchers"
	"main/utils"
	"os"
	"strings"
)

type Config struct {
	launchers.Launcher
	launchers.Minecraft
	api_endpoint string
	root         string
	pack         string
	mod_dir      string
	eofd         bool
	notify       bool
}

func NewConfigFromGui(
	launcher launchers.Launcher,
	instance launchers.Instance,
) Config {
	var api_endpoint string
	if instance.Staging() {
		api_endpoint = globals.STAGING_NORISK_API_ENDPOINT
	} else {
		api_endpoint = globals.NORISK_API_ENDPOINT
	}
	return Config{
		launcher,
		launchers.NewMinecraft(instance),
		api_endpoint,
		instance.Path(),
		instance.Pack(),
		instance.ModDir(),
		false,
		true,
	}
}

func (config Config) ApiEndpoint() string {
	return config.api_endpoint
}

func (config Config) Root() string {
	return config.root
}

func (config Config) Pack() string {
	return config.pack
}

func (config Config) ModDir() string {
	return config.mod_dir
}

func (config Config) ErrorOnFailedDownload() bool {
	return config.eofd
}

func (config Config) Notify() bool {
	return config.notify
}

func GetConfig() Config {
	var config Config
	var launcher, dir string
	home, _ := os.UserHomeDir()

	if value := os.Getenv("STAGING"); value != "" {
		config.api_endpoint = globals.STAGING_NORISK_API_ENDPOINT
	} else {
		config.api_endpoint = globals.NORISK_API_ENDPOINT
	}

	if value := os.Getenv("LAUNCHER"); value != "" {
		if _, e := launchers.LAUNCHERS[value]; e {
			log.Printf("Set %s manually", value)
			launcher = value
		}
	} else {
		for i := len(os.Args) - 1; i >= 0; i-- {
			for id, l := range launchers.LAUNCHERS {
				if os.Args[i] == l.JavaClass {
					log.Printf("Detected %s\n", l.Name)
					launcher = id
					break
				}
			}
		}
	}

	if launcher == "" {
		utils.Notify("No valid launcher detected or set manually", true, config.notify)
	}

	if value := os.Getenv(fmt.Sprintf("%s_DIR", strings.ToUpper(launcher))); value != "" {
		dir = value
	}
	config.Launcher = launchers.LAUNCHERS[launcher].New(home, dir, os.Getenv("FLATPAK_ID") != "")

	switch os.Getenv("NOTIFY") {
	case "true", "True", "1":
		config.notify = true
	case "false", "False", "0":
		config.notify = false
	default:
		config.notify = config.Launcher.DefaultNotify()
	}

	if value := os.Getenv("NRC_PACK"); value != "" {
		config.pack = value
	} else {
		config.pack = globals.DEFAULT_PACK
	}

	minecraft, err := config.GetCurrentInstanceDetails()
	if err != nil {
		utils.Notify(
			fmt.Sprintf("Failed to get Minecraft details: %s", err.Error()),
			true,
			config.notify,
		)
	}
	config.Minecraft = minecraft

	if value := os.Getenv("NRC_MOD_DIR"); value != "" {
		config.mod_dir = value
	} else if config.Loader() == "fabric" {
		config.mod_dir = globals.DEFAULT_MOD_DIR
	} else {
		config.mod_dir = "mods"
	}

	config.eofd = os.Getenv("NO_ERROR_ON_FAILED_DOWNLOAD") == "" && os.Getenv("NEOFD") == ""

	return config
}
