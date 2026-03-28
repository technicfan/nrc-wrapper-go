package fetcher

import (
	"fmt"
	"log"
	"main/config"
	"main/mod_entry"
	"main/utils"
	"os"
	"path/filepath"

	"sync"
)


func Get_installed_mods(
	root string,
	mod_dir string,
) (mod_entry.ModEntries, error) {
	files, err := os.ReadDir(filepath.Join(root, mod_dir))
	if err != nil {
		return nil, err
	}
	index := utils.Read_index(filepath.Join(root, ".nrc-mod-index.json"))

	hashes := make(map[string]string)
	for _, f := range files {
		if !f.IsDir() &&
			(filepath.Ext(f.Name()) == ".jar" || filepath.Ext(f.Name()) == ".disabled") {
			entry, e := index[f.Name()]
			var hash string
			var err error
			if e {
				hash = entry["hash"]
			} else {
				hash, err = utils.Calc_hash(filepath.Join(root, mod_dir, f.Name()))
			}
			if err == nil {
				hashes[hash] = f.Name()
			}
		}
	}

	results := make(mod_entry.ModEntries)
	for _, entry := range index {
		if _, exists := hashes[entry["hash"]]; exists {
			results[entry["id"]] = mod_entry.New(
				entry["hash"],
				entry["version"],
				entry["id"],
				hashes[entry["hash"]],
				mod_dir,
				"",
				"",
				false,
			)
		}
	}

	return results, nil
}

func Download_mods_async(
	mods mod_entry.ModEntries,
	config config.Config,
	limiter chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	os.Mkdir(config.ModDir, os.ModePerm)

	installed_mods, err := Get_installed_mods("./", config.ModDir)
	if err != nil {
		utils.Notify(fmt.Sprintf("Failed to get installed mods: %s", err.Error()), true, config.Notify)
	}
	mods_to_download, already_installed := mods.Get_missing_mods(
		installed_mods,
		config.ModDir,
	)

	if len(mods_to_download) == 0 {
		return
	}

	log.Println("Installing missing/updated mods")

	var wg1 sync.WaitGroup

	index := make(chan utils.Pair, len(mods_to_download))
	for _, mod := range mods_to_download {
		wg1.Add(1)
		go mod.Download_async(config, &wg1, index, limiter)
	}

	wg1.Wait()
	close(index)

	if len(index) > 0 {
		existing_index := already_installed.Convert_to_index()
		for k := range index {
			existing_index[k.Key] = k.Value
		}
		err = existing_index.Write(".nrc-mod-index.json")
		if err != nil {
			utils.Notify(
				fmt.Sprintf("Failed to write mod metadata: %s", err.Error()),
				true,
				config.Notify,
			)
		}
	}
}
