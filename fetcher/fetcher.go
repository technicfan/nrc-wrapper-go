package fetcher

import (
	"fmt"
	"log"
	"main/api"
	"main/config"
	"main/globals"
	"main/mods"
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
	inherited_mods, assets, support := pack.Details(versions.Packs)

	if len(support) > 0 {
		if version, exists := support[config.Loader()]; exists {
			if utils.CmpVersions(config.LoaderVersion(), version.LoaderVersion) < 0 {
				return fmt.Errorf(
					"Please update %s to version %s",
					config.Loader(),
					version,
				)
			}
		} else {
			var loaders_str []string
			for loader, version := range support {
				if version.LoaderVersion != "0" {
					loaders_str = append(loaders_str, fmt.Sprintf("%s %s", loader, version.LoaderVersion))
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

	pack_mods := pack.Mods.CompatibleMods(config, versions.Repositories)
	if len(pack_mods) == 0 {
		return fmt.Errorf(
			"There are no NRC mods for %s in %s",
			config.Version(),
			config.Pack(),
		)
	}
	maps.Copy(pack_mods, inherited_mods.CompatibleMods(config, versions.Repositories))

	resources, asset_index, update_assets := get_assets(assets, config.ApiEndpoint())
	installed_mods, update_mods := mods.GetInstalledMods("./", config.ModDir())
	mods_to_download, already_installed := pack_mods.GetMissing(
		installed_mods,
		config.ModDir(),
	)
	asset_index_chan := make(chan utils.Pair, len(resources))
	mod_index_chan := make(chan utils.Pair, len(mods_to_download))
	for id := range mods_to_download {
		resources = append(resources, mods_to_download[id])
	}

	if len(resources) > 0 {
		log.Println("Downloading missing/updated resources")
	}

	var wg sync.WaitGroup
	limiter := make(chan struct{}, 10)
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
		already_installed.Index().Merge(mod_index_chan).Write(globals.MOD_INDEX)
	}

	return nil
}
