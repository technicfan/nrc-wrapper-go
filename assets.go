package main

import (
	"crypto/md5"
	"fmt"
	"hash"
	"log"
	"maps"
	"path/filepath"
	"sync"
)

type Asset struct {
	pack string
	path string
	hash string
}

func (asset Asset) Url() string {
	return fmt.Sprintf("%s/%s/assets/%s", NORISK_ASSETS_URL, asset.pack, asset.path)
}

func (asset Asset) Path() string {
	return filepath.Join("NoRiskClient/assets", asset.path)
}

func (asset Asset) Filename() string {
	return filepath.Base(asset.path)
}

func (asset Asset) ExpectedHash() string {
	return asset.hash
}

func (asset Asset) HashObj() hash.Hash {
	return md5.New()
}

func (asset Asset) Download() error {
	return download(asset)
}

func (asset Asset) IsMissing() bool {
	if hash, err := calc_hash(asset.Path()); err == nil && hash == asset.ExpectedHash() {
		return false
	}
	return true
}

func (asset Asset) download_async(
	config Config,
	wg *sync.WaitGroup,
	limiter chan struct{},
) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <-limiter }()

	err := asset.Download()
	if err != nil {
		notify(
			fmt.Sprintf("Failed to download %s: %s", asset.Filename(), err.Error()),
			config.ErrorOnFailedDownload,
			config.Notify,
		)
		return
	}

	log.Printf("Downloaded %s/%s", asset.pack, asset.Filename())
}

func download_assets_async(
	packs []string,
	config Config,
	limiter chan struct{},
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

	var missing_assets []Asset
	for _, asset := range merged {
		if asset.IsMissing() {
			missing_assets = append(missing_assets, asset)
		}
	}

	if len(missing_assets) != 0 {
		log.Println("Downloading missing/updated assets")
	}

	for i := range missing_assets {
		wg.Add(1)
		go missing_assets[i].download_async(config, &wg, limiter)
	}

	wg.Wait()
}
