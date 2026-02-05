package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func main() {
	launch := true
	if len(os.Args) == 2 && os.Args[1] == "--packs" {
		launch = false
		cli()
	} else if len(os.Args) < 3 {
		gui()
	}

	var token string
	var config Config
	var wg sync.WaitGroup
	if launch {
		log.Println("Loading NoRiskClient...")
		config = get_config()
	}

	if os.Getenv("STAGING") != "" {
		NORISK_API_URL = NORISK_API_STAGING_URL
	}

	versions, err := get_norisk_versions(NORISK_API_URL)
	if err == nil {
		if !launch {
			versions.Packs.print_packs()
			return
		}

		pack, exists := versions.Packs[config.NrcPack]
		if !exists {
			notify(fmt.Sprintf("%s is not a valid NRC pack", config.NrcPack), true, config.Notify)
		}
		mods, assets, loaders := pack.get_details(versions.Packs)

		if len(loaders) > 0 {
			if version, exists := loaders[config.Minecraft.Loader]; exists {
				if config.Minecraft.LoaderVersion < version {
					notify(
						fmt.Sprintf(
							"Please update %s to version %s",
							config.Minecraft.Loader,
							version,
						),
						true,
						config.Notify,
					)
				}
			} else {
				var loaders []string
				for loader, version := range pack.Loader["default"] {
					loaders = append(loaders, fmt.Sprintf("%s %s", loader, version))
				}
				notify(
					fmt.Sprintf(
						"%s requires one of the following modloaders: %s",
						config.NrcPack,
						strings.Join(loaders, ", "),
					),
					true,
					config.Notify,
				)
			}
		}

		wg.Add(2)

		go download_assets_async(assets, config, &wg)
		go download_mods_async(config, pack.Mods, mods, versions.Repositories, &wg)
		token, err = get_token(config, false)

		wg.Wait()
	} else {
		if !launch {
			log.Println("No connection to the API")
			return
		}
		notify("No connection to the API\nLaunching without doing anything", false, config.Notify)
		token, err = get_token(config, true)
	}

	if err != nil {
		notify(fmt.Sprintf("Failed to get nrc token: %s", err.Error()), true, config.Notify)
	}

	command := os.Args[1]
	args := append(
		[]string{
			command, fmt.Sprintf("-Dnorisk.token=%s", token),
			fmt.Sprintf("-Dnorisk.profile.name=%s", config.Minecraft.Profile),
			fmt.Sprintf("-Dfabric.addMods=%s", config.ModDir),
		}, os.Args[2:]...,
	)

	err = Exec(command, args)
	if err != nil {
		notify(fmt.Sprintf("Command failed with: %s", err.Error()), true, config.Notify)
	}
}
