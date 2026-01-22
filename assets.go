package main

import (
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"sync"
)

func verify_asset_async(
	path string,
	data map[string]string,
	wg *sync.WaitGroup,
	results chan<- map[string]string,
) {
	defer wg.Done()

	if hash, err := calc_hash(fmt.Sprintf("NoRiskClient/assets/%s", path));
		err == nil && hash == data["hash"] {
		return
	}

	data["path"] = path
	results <- data
}

func download_asset_async(
	asset map[string]string,
	error_on_fail bool,
	do_notify bool,
	wg *sync.WaitGroup,
	limiter chan struct{},
) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <-limiter }()

	err := download_asset(asset["pack"], asset["path"], asset["hash"])
	if err != nil {
		notify(
			fmt.Sprintf("Failed to download %s: %s", filepath.Base(asset["path"]), err.Error()),
			error_on_fail,
			do_notify,
		)
		return
	}

	log.Printf("Downloaded %s/%s", asset["pack"], filepath.Base(asset["path"]))
}

func download_assets_async(
	packs []string,
	config Config,
	wg1 *sync.WaitGroup,
) {
	defer wg1.Done()
	var wg sync.WaitGroup
	data := make(chan map[int]map[string]map[string]string, len(packs))
	for i, pack := range packs {
		wg.Add(1)
		go get_asset_metadata_async(i, pack, &wg, data)
	}

	go func() {
		wg.Wait()
		close(data)
	}()

	final_data := make(map[int]map[string]map[string]string)
	for obj := range data {
		maps.Copy(final_data, obj)
	}
	merged := make(map[string]map[string]string)
	for i := 0; i < len(final_data); i++ {
		maps.Copy(merged, final_data[i])
	}

	missing_assets := make(chan map[string]string, len(merged))
	for i, v := range merged {
		wg.Add(1)
		go verify_asset_async(i, v, &wg, missing_assets)
	}

	go func() {
		wg.Wait()
		close(missing_assets)
	}()

	if len(missing_assets) != 0 {
		log.Println("Downloading missing/updated assets")
	}

	limiter := make(chan struct{}, 20)
	for asset := range missing_assets {
		wg.Add(1)
		go download_asset_async(
			asset,
			config.ErrorOnFailedDownload,
			config.Notify,
			&wg,
			limiter,
		)
	}

	wg.Wait()
}
