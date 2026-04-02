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
	var print, refresh bool
	launch := true
	if len(os.Args) == 2 && os.Args[1] == "--packs" {
		launch = false
		print = true
		platform.Cli()
	} else if len(os.Args) == 3 && os.Args[1] == "--refresh" {
		launch = false
		refresh = true
	} else if len(os.Args) < 3 {
		gui.Gui()
		return
	}

	var err2 error
	var token string
	var cfg config.Config
	var wg sync.WaitGroup
	if !print {
		log.Println("Loading NoRiskClient...")
		cfg = config.GetConfig(refresh)
	}

	if os.Getenv("STAGING") != "" {
		globals.NORISK_API_URL = globals.NORISK_API_STAGING_URL
	}

	versions, err := api.GetVersions()
	if err == nil {
		if print {
			versions.Packs.Print()
			return
		}

		pack, exists := versions.Packs[cfg.Pack()]
		if !exists {
			utils.Notify(fmt.Sprintf("%s is not a valid NRC pack", cfg.Pack()), true, cfg.Notify())
		}
		pack_mods, assets, _, loaders := pack.Details(versions.Packs)

		if !refresh && len(loaders) > 0 {
			if version, exists := loaders[cfg.Loader()]; exists {
				if utils.CmpVersions(cfg.LoaderVersion(), version) < 0 {
					utils.Notify(
						fmt.Sprintf(
							"Please update %s to version %s",
							cfg.Loader(),
							version,
						),
						true,
						cfg.Notify(),
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
						cfg.Pack(),
						strings.Join(loaders_str, ", "),
					),
					true,
					cfg.Notify(),
				)
			}
		}

		mods := pack.Mods.CompatibleMods(cfg, versions.Repositories)
		if len(mods) == 0 {
			utils.Notify(
				fmt.Sprintf(
					"There are no NRC mods for %s in %s",
					cfg.Version(),
					cfg.Pack(),
				),
				true,
				cfg.Notify(),
			)
		}
		maps.Copy(mods, pack_mods.CompatibleMods(cfg, versions.Repositories))

		asset_resources, asset_index, update_assets := fetcher.GetAssets(assets)
		mod_resources, mod_index, update_mods := fetcher.GetMods(mods, cfg)
		resources := append(asset_resources, mod_resources...)

		token, err, err2 = fetcher.GetToken(cfg, false)

		if len(resources) > 0 {
			log.Println("Downloading missing/updated resources")
		}

		limiter := make(chan struct{}, 10)
		asset_index_chan := make(chan utils.Pair, len(asset_resources))
		mod_index_chan := make(chan utils.Pair, len(mod_resources))
		for i := range resources {
			wg.Add(1)
			go utils.DownloadAsync(
				resources[i],
				cfg.ErrorOnFailedDownload(),
				cfg.Notify(),
				mod_index_chan,
				asset_index_chan,
				&wg,
				limiter,
			)
		}

		wg.Wait()
		close(asset_index_chan)
		close(mod_index_chan)

		if update_assets || len(asset_index_chan) > 0 {
			asset_index.Merge(asset_index_chan).Write(globals.ASSET_INDEX)
		}
		if update_mods || len(mod_index_chan) > 0 {
			mod_index.Merge(mod_index_chan).Write(globals.MOD_INDEX)
		}
	} else {
		if !launch {
			log.Fatalln("No connection to the API")
			return
		}
		utils.Notify("No connection to the API\nLaunching without doing anything", false, cfg.Notify())
		token, err, err2 = fetcher.GetToken(cfg, true)
	}

	if err != nil {
		if err2 == nil {
			utils.Notify(fmt.Sprintf("Failed to get nrc token: %s", err.Error()), true, cfg.Notify())
		} else {
			utils.Notify(fmt.Sprintf("Failed to get nrc token: %s", err2.Error()), false, cfg.Notify())
			utils.Notify(fmt.Sprintf("Failed to get nrc token for flatpak: %s", err.Error()), true, cfg.Notify())
		}
	}
	if err2 != nil {
		utils.Notify(fmt.Sprintf("Failed to get nrc token: %s", err2.Error()), false, cfg.Notify())
	}

	if launch {
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
}
