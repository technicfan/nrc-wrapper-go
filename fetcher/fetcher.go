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
) ([]string, error) {
	pack, exists := versions.Packs[config.Pack()]
	if !exists {
		return nil, fmt.Errorf("%s is not a valid NRC pack", config.Pack())
	}
	inherited_mods, assets, support := pack.Details(versions.Packs)

	if len(support) > 0 {
		if version, exists := support[config.Loader()]; exists {
			if utils.CmpVersions(config.LoaderVersion(), version.LoaderVersion) < 0 {
				return nil, fmt.Errorf(
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
			return nil, fmt.Errorf(
				"%s requires one of the following modloaders: %s",
				config.Pack(),
				strings.Join(loaders_str, ", "),
			)
		}
	}

	pack_mods := pack.Mods.CompatibleMods(config, versions.Repositories)
	if len(pack_mods) == 0 {
		return nil, fmt.Errorf(
			"There are no NRC mods for %s in %s",
			config.Version(),
			config.Pack(),
		)
	}
	maps.Copy(pack_mods, inherited_mods.CompatibleMods(config, versions.Repositories))

	installed_mods, left_over, update_mods := mods.GetInstalledMods(config.Root(), config.ModDir())
	resources, already_installed, left_over1 := pack_mods.GetMissing(
		installed_mods,
		config.ModDir(),
	)
	maps.Copy(left_over, left_over1)
	for file, entry := range left_over {
		if path, e := entry.Path(); e {
			os.Remove(filepath.Join(path, file))
			log.Printf("Removed left over file %s", filepath.Base(file))
			if f, _ := os.ReadDir(path); path != "mods" && len(f) == 0 {
				os.Remove(path)
			}
		}
	}
	index := make(chan utils.Pair, len(resources))

	if len(resources) > 0 {
		log.Println("Downloading missing/updated mods")
	}

	var wg sync.WaitGroup
	limiter := make(chan struct{}, 10)
	for i := range resources {
		wg.Add(1)
		go utils.DownloadAsync(
			resources[i],
			config.ErrorOnFailedDownload(),
			config.Notify(),
			index,
			&wg,
			limiter,
		)
	}

	wg.Wait()
	close(index)

	if update_mods || len(index) > 0 {
		already_installed.Index().Merge(index).Write(filepath.Join(config.Root(), globals.MOD_INDEX))
	}

	return assets, nil
}
