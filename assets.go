package main

import (
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"sync"
)

type Asset struct {
	Pack string
	Path string
	Hash string
}

func verify_asset_async(
	path string,
	data Asset,
	wg *sync.WaitGroup,
	results chan<- Asset,
) {
	defer wg.Done()

	if hash, err := calc_hash(fmt.Sprintf("NoRiskClient/assets/%s", path));
		err == nil && hash == data.Hash {
		return
	}

	data.Path = path
	results <- data
}

func (asset Asset) download_async(
	config Config,
	wg *sync.WaitGroup,
	limiter chan struct{},
) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <-limiter }()

	err := download_file(
		fmt.Sprintf("%s/%s/assets/%s", NORISK_ASSETS_URL, asset.Pack, asset.Path),
		asset.Path,
		"NoRiskClient/assets",
		false,
		asset.Hash,
	)
	if err != nil {
		notify(
			fmt.Sprintf("Failed to download %s: %s", filepath.Base(asset.Path), err.Error()),
			config.ErrorOnFailedDownload,
			config.Notify,
		)
		return
	}

	log.Printf("Downloaded %s/%s", asset.Pack, filepath.Base(asset.Path))
}

func download_assets_async(
	packs []string,
	config Config,
	wg1 *sync.WaitGroup,
) {
	defer wg1.Done()
	var wg sync.WaitGroup
	data := make(chan map[int]map[string]Asset, len(packs))
	for i, pack := range packs {
		wg.Add(1)
		go get_asset_metadata_async(i, pack, &wg, data)
	}

	go func() {
		wg.Wait()
		close(data)
	}()

	final_data := make(map[int]map[string]Asset)
	for obj := range data {
		maps.Copy(final_data, obj)
	}
	merged := make(map[string]Asset)
	for i := 0; i < len(final_data); i++ {
		maps.Copy(merged, final_data[i])
	}

	missing_assets := make(chan Asset, len(merged))
	for path, data := range merged {
		wg.Add(1)
		go verify_asset_async(path, data, &wg, missing_assets)
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
		go asset.download_async(config, &wg, limiter)
	}

	wg.Wait()
}
