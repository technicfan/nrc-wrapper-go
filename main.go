package main

import (
	"fmt"
	"log"
	"main/api"
	"main/config"
	"main/fetcher"
	"main/gui"
	"main/platform"
	"main/utils"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		gui.Gui()
		return
	}

	var cfg config.Config
	var assets []string
	log.Println("Loading NoRiskClient...")
	cfg = config.GetConfig()

	versions, err := api.GetVersions(cfg.ApiEndpoint())
	if err == nil {
		assets, err = fetcher.Fetch(versions, cfg)
		if err != nil {
			utils.Notify(err.Error(), true, cfg.Notify())
		}
	} else {
		utils.Notify("No connection to the API\nLaunching without doing anything", false, cfg.Notify())
	}

	command := os.Args[1]
	args := append(
		[]string{
			command,
			fmt.Sprintf("-Dnrc.assets.bucket=%s", strings.Join(assets, ",")),
			fmt.Sprintf("-Dnorisk.experimental=%t", cfg.Staging()),
			fmt.Sprintf("-Dnorisk.pack=%s", cfg.Pack()),
			fmt.Sprintf("-Dnorisk.profile.name=%s", cfg.Profile()),
			fmt.Sprintf("-Dfabric.addMods=%s", cfg.ModDir()),
		}, os.Args[2:]...,
	)

	err = platform.Exec(command, args)
	if err != nil {
		utils.Notify(fmt.Sprintf("Command failed with: %s", err.Error()), true, cfg.Notify())
	}
}
