package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"path/filepath"
	"sync"
)

func calc_hash(
	path string,
) (string, error) {
	var file, err = os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	var hash = md5.New()
	_, err = hash.Write(data)
	if err != nil {
		return "", err
	}

	var bytesHash = hash.Sum(nil)
	return hex.EncodeToString(bytesHash[:]), nil
}

func verify_asset(
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

func download_asset(
	asset map[string]string,
	error_on_fail bool,
	wg *sync.WaitGroup,
	limiter chan struct{},
) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <- limiter }()

	err := download_single_asset(asset["pack"], asset["path"], asset["hash"])
	if err != nil {
		notify(
			fmt.Sprintf("Failed to download %s: %s", filepath.Base(asset["path"]), err.Error()),
			error_on_fail,
		)
	}

	log.Printf("Downloaded %s/%s", asset["pack"], filepath.Base(asset["path"]))
}

func load_assets(
	packs []string,
	error_on_fail bool,
	wg1 *sync.WaitGroup,
) {
	defer wg1.Done()
	var wg sync.WaitGroup
	data := make(chan map[int]map[string]map[string]string, len(packs))
	for i, pack := range packs {
		wg.Add(1)
		go get_asset_metadata(i, pack, &wg, data)
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
		go verify_asset(i, v, &wg, missing_assets)
	}

	go func() {
		wg.Wait()
		close(missing_assets)
	}()

	if len(missing_assets) != 0 {
		log.Println("Downloading missing assets")
	}

	limiter := make(chan struct{}, 20)
	for asset := range missing_assets {
		wg.Add(1)
		go download_asset(
			asset,
			error_on_fail,
			&wg,
			limiter,
		)
	}

	wg.Wait()
}
