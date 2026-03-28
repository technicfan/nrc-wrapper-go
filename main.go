package main

import (
	"fmt"
	"log"
	"main/api"
	"main/config"
	"main/fetcher"
	"main/globals"
	"main/gui"
	"main/platform"
	"main/utils"
	"maps"
	"os"
	"strings"
	"sync"
)

func main() {
	launch := true
	if len(os.Args) == 2 && os.Args[1] == "--packs" {
		launch = false
		platform.Cli()
	} else if len(os.Args) == 2 && os.Args[1] == "--refresh" {
		globals.REFRESH = true
	} else if len(os.Args) < 3 {
		gui.Gui()
		os.Exit(0)
	}

	var token string
	var cfg config.Config
	var wg sync.WaitGroup
	if launch {
		log.Println("Loading NoRiskClient...")
		cfg = config.Get_config()
	}

	if os.Getenv("STAGING") != "" {
		globals.NORISK_API_URL = globals.NORISK_API_STAGING_URL
	}

	versions, err := api.Get_norisk_versions()
	if err == nil {
		if !launch {
			versions.Packs.Print()
			return
		}

		pack, exists := versions.Packs[cfg.NrcPack]
		if !exists {
			utils.Notify(fmt.Sprintf("%s is not a valid NRC pack", cfg.NrcPack), true, cfg.Notify)
		}
		pack_mods, assets, _, loaders := pack.Get_details(versions.Packs)

		if !globals.REFRESH && len(loaders) > 0 {
			if version, exists := loaders[cfg.Minecraft.Loader]; exists {
				if utils.Cmp_versions(cfg.Minecraft.LoaderVersion, version) < 0 {
					utils.Notify(
						fmt.Sprintf(
							"Please update %s to version %s",
							cfg.Minecraft.Loader,
							version,
						),
						true,
						cfg.Notify,
					)
				}
			} else {
				var loaders_str []string
				for loader, version := range loaders {
					if version != "0" {
						loaders_str = append(loaders_str, fmt.Sprintf("%s %s", loader, version))
					} else {
						loaders_str = append(loaders_str, loader)
					}
				}
				utils.Notify(
					fmt.Sprintf(
						"%s requires one of the following modloaders: %s",
						cfg.NrcPack,
						strings.Join(loaders_str, ", "),
					),
					true,
					cfg.Notify,
				)
			}
		}

		mods := pack.Mods.Get_compatible_mods(cfg, versions.Repositories)
		if len(mods) == 0 {
			utils.Notify(
				fmt.Sprintf(
					"There are no NRC mods for %s in %s",
					cfg.Minecraft.Version,
					cfg.NrcPack,
				),
				true,
				cfg.Notify,
			)
		}
		maps.Copy(mods, pack_mods.Get_compatible_mods(cfg, versions.Repositories))

		wg.Add(2)
		limiter := make(chan struct{}, 10)

		go fetcher.Download_assets_async(assets, cfg, limiter, &wg)
		go fetcher.Download_mods_async(mods, cfg, limiter, &wg)
		token, err = fetcher.Get_token(cfg, false)

		wg.Wait()
	} else {
		if !launch {
			log.Println("No connection to the API")
			return
		}
		utils.Notify("No connection to the API\nLaunching without doing anything", false, cfg.Notify)
		token, err = fetcher.Get_token(cfg, true)
	}

	if err != nil {
		utils.Notify(fmt.Sprintf("Failed to get nrc token: %s", err.Error()), true, cfg.Notify)
	}

	if !globals.REFRESH {
		command := os.Args[1]
		args := append(
			[]string{
				command, fmt.Sprintf("-Dnorisk.token=%s", token),
				fmt.Sprintf("-Dnorisk.profile.name=%s", cfg.Minecraft.Profile),
				fmt.Sprintf("-Dfabric.addMods=%s", cfg.ModDir),
			}, os.Args[2:]...,
		)

		err = platform.Exec(command, args)
		if err != nil {
			utils.Notify(fmt.Sprintf("Command failed with: %s", err.Error()), true, cfg.Notify)
		}
	}
}
