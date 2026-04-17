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
	"os"
	"path/filepath"
	"slices"
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
					version.LoaderVersion,
				)
			}
		} else {
			var loaders_str []string
			for _, loader := range slices.Sorted(maps.Keys(support)) {
				if support[loader].LoaderVersion != "0" {
					loaders_str = append(loaders_str,
						fmt.Sprintf("%s %s", loader, support[loader].LoaderVersion),
					)
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

	resources, asset_index, left_over, update_assets := get_assets(config.Root(), assets, config.ApiEndpoint())
	installed_mods, left_over1, update_mods := mods.GetInstalledMods(config.Root(), config.ModDir())
	mods_to_download, already_installed, left_over2 := pack_mods.GetMissing(
		installed_mods,
		config.ModDir(),
	)
	maps.Copy(left_over, left_over1)
	maps.Copy(left_over, left_over2)
	for file, entry := range left_over {
		if path, e := entry["path"]; e {
			os.Remove(filepath.Join(path, file))
			log.Printf("Removed left over file %s", filepath.Base(file))
			if f, _ := os.ReadDir(path); path != "mods" && len(f) == 0 {
				os.Remove(path)
			}
		}
	}
	indexes := []chan utils.Pair{
		make(chan utils.Pair, len(resources)),
		make(chan utils.Pair, len(mods_to_download)),
	}
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
			indexes,
			&wg,
			limiter,
		)
	}

	wg.Wait()
	for i := range indexes {
		close(indexes[i])
	}

	if update_assets || len(indexes[0]) > 0 {
		asset_index.Merge(indexes[0]).Write(filepath.Join(config.Root(), globals.ASSET_INDEX))
	}
	if update_mods || len(indexes[1]) > 0 {
		already_installed.Index().Merge(indexes[1]).Write(filepath.Join(config.Root(), globals.MOD_INDEX))
	}

	return nil
}
