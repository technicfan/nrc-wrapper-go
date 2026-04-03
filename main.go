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
)

func main() {
	if len(os.Args) < 3 {
		gui.Gui()
		return
	}

	var token string
	var cfg config.Config
	log.Println("Loading NoRiskClient...")
	cfg = config.GetConfig()

	versions, err := api.GetVersions(cfg.ApiEndpoint())
	if err == nil {
		token, err = fetcher.GetToken(cfg, false)

		fetch_err := fetcher.Fetch(versions, cfg)
		if fetch_err != nil {
			utils.Notify(fetch_err.Error(), true, cfg.Notify())
		}
	} else {
		utils.Notify("No connection to the API\nLaunching without doing anything", false, cfg.Notify())
		token, err = fetcher.GetToken(cfg, true)
	}

	if err != nil {
		utils.Notify(fmt.Sprintf("Failed to get nrc token: %s", err.Error()), true, cfg.Notify())
	}

	command := os.Args[1]
	args := append(
		[]string{
			command, fmt.Sprintf("-Dnorisk.token=%s", token),
			fmt.Sprintf("-Dnorisk.profile.name=%s", cfg.Profile()),
			fmt.Sprintf("-Dfabric.addMods=%s", cfg.ModDir()),
		}, os.Args[2:]...,
	)

	err = platform.Exec(command, args)
	if err != nil {
		utils.Notify(fmt.Sprintf("Command failed with: %s", err.Error()), true, cfg.Notify())
	}
}
