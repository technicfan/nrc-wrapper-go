package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"sync"
)

type Index []map[string]string

func read_index(path string) Index {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return nil
	}
	defer file.Close()

	var data Index
	err = json.Unmarshal(byte_data, &data)
	if err != nil {
		return nil
	}

	return data
}

func (data Index) write() error {
	var file *os.File
	file, err := os.OpenFile(".nrc-index.json", os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(".nrc-index.json")
		if err != nil {
			return err
		}
	}
	defer file.Close()

	json_string, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(json_string))
	if err != nil {
		return err
	}

	return nil
}

func get_installed_mods(
	root string,
	mod_dir string,
) (ModEntries, error) {
	files, err := os.ReadDir(filepath.Join(root, mod_dir))
	if err != nil {
		return nil, err
	}
	index := read_index(filepath.Join(root, ".nrc-index.json"))

	hashes := make(map[string]map[string]string)
	for _, f := range files {
		if !f.IsDir() &&
			(filepath.Ext(f.Name()) == ".jar" || filepath.Ext(f.Name()) == ".disabled") {
			hash, err := calc_hash(filepath.Join(root, mod_dir, f.Name()))
			if err == nil {
				info := make(map[string]string)
				info["filename"] = f.Name()
				hashes[hash] = info
			}
		}
	}

	results := make(ModEntries)
	for _, entry := range index {
		if _, exists := hashes[entry["hash"]]; exists {
			results[entry["id"]] = ModEntry{
				entry["hash"],
				entry["version"],
				entry["id"],
				hashes[entry["hash"]]["filename"],
				"",
				mod_dir,
				"",
				"",
				false,
				false,
			}
		}
	}

	return results, nil
}

func download_mods_async(
	mods ModEntries,
	config Config,
	limiter chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	os.Mkdir(config.ModDir, os.ModePerm)

	installed_mods, err := get_installed_mods("./", config.ModDir)
	if err != nil {
		notify(fmt.Sprintf("Failed to get installed mods: %s", err.Error()), true, config.Notify)
	}
	mods_to_download, already_installed := mods.get_missing_mods(
		installed_mods,
		config.ModDir,
	)

	if len(mods_to_download) == 0 {
		return
	}

	log.Println("Installing missing/updated mods")

	var wg1 sync.WaitGroup

	index := make(chan map[string]string, len(mods_to_download))
	for _, mod := range mods_to_download {
		wg1.Add(1)
		go mod.download_async(config, &wg1, index, limiter)
	}

	wg1.Wait()
	close(index)

	if len(index) > 0 {
		existing_index := already_installed.convert_to_index()
		for entry := range index {
			existing_index = append(existing_index, entry)
		}
		err = existing_index.write()
		if err != nil {
			notify(
				fmt.Sprintf("Failed to write mod metadata: %s", err.Error()),
				true,
				config.Notify,
			)
		}
	}
}
