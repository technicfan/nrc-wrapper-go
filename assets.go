package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"sync"
)

func calc_hash(path string) (string, error) {
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

func verify_asset(path string, data map[string]string, wg *sync.WaitGroup, results chan <- VerifiedAsset) {
	defer wg.Done()

	if hash, err := calc_hash(fmt.Sprintf("NoRiskClient/assets/%s", path)); err == nil && hash == data["hash"] {
		results <- VerifiedAsset{true, "", nil}
		return
	}

	results <- VerifiedAsset{false, path, data}
}

func load_assets(packs []string, wg1 *sync.WaitGroup) error {
	defer wg1.Done()
	var wg sync.WaitGroup
	data := make(chan map[string]map[string]string)
	for _, pack := range packs {
		wg.Add(1)
		go get_asset_metadata(pack, &wg, data)
	}

	go func() {
		wg.Wait()
		close(data)
	}()

	merged := make(map[string]map[string]string)
	for pack := range data {
		maps.Copy(merged, pack)
	}

	results := make(chan VerifiedAsset, len(merged))
	for i, v := range merged {
		wg.Add(1)
		go verify_asset(i, v, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	downloading := false
	limiter := make(chan struct{}, 20)
	for result := range results {
		if !result.Result {
			if !downloading { log.Print("Downloading missing assets"); downloading = true }
			wg.Add(1)
			go download_single_asset(result.Asset["pack"], result.Path, result.Asset["hash"], &wg, limiter)
		}
	}

	wg.Wait()

	return nil
}
