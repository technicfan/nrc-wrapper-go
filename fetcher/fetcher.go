package fetcher

import (
	"fmt"
	"log"
	"main/api"
	"main/config"
	"main/globals"
	"main/utils"
	"maps"
	"strings"
	"sync"
)

func Fetch(
	versions api.Versions,
	config config.Config,
) error {
	pack, exists := versions.Packs[config.Pack()]
	if !exists {
		return fmt.Errorf("%s is not a valid NRC pack", config.Pack())
	}
	pack_mods, assets, _, loaders := pack.Details(versions.Packs)

	if len(loaders) > 0 {
		if version, exists := loaders[config.Loader()]; exists {
			if utils.CmpVersions(config.LoaderVersion(), version) < 0 {
				return fmt.Errorf(
					"Please update %s to version %s",
					config.Loader(),
					version,
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
			return fmt.Errorf(
				"%s requires one of the following modloaders: %s",
				config.Pack(),
				strings.Join(loaders_str, ", "),
			)
		}
	}

	mods := pack.Mods.CompatibleMods(config, versions.Repositories)
	if len(mods) == 0 {
		return fmt.Errorf(
			"There are no NRC mods for %s in %s",
			config.Version(),
			config.Pack(),
		)
	}
	maps.Copy(mods, pack_mods.CompatibleMods(config, versions.Repositories))

	asset_resources, asset_index, update_assets := GetAssets(assets, config.ApiEndpoint())
	mod_resources, mod_index, update_mods := GetMods(mods, config)
	resources := append(asset_resources, mod_resources...)

	if len(resources) > 0 {
		log.Println("Downloading missing/updated resources")
	}

	var wg sync.WaitGroup
	limiter := make(chan struct{}, 10)
	asset_index_chan := make(chan utils.Pair, len(asset_resources))
	mod_index_chan := make(chan utils.Pair, len(mod_resources))
	for i := range resources {
		wg.Add(1)
		go utils.DownloadAsync(
			resources[i],
			config.ErrorOnFailedDownload(),
			config.Notify(),
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

	return nil
}
