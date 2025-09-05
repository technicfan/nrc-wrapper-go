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

func load_assets(token string, packs []string, wg1 *sync.WaitGroup) error {
	defer wg1.Done()
	var wg sync.WaitGroup
	data := make(map[string]map[string]map[string]string)
	results := make(chan VerifiedAsset, 10)
	for _, pack := range packs {
		metadata, err := get_asset_metadata(pack)
		if err != nil {
			return err
		}
		data[pack] = metadata
	}

	merged := make(map[string]map[string]string)
	for _, pack := range data {
		maps.Copy(merged, pack)
	}

	log.Print("Verifying assets")

	for i, v := range merged {
		wg.Add(1)
		go verify_asset(i, v, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if !result.Result {
			wg.Add(1)
			go download_single_asset(result.Asset["pack"], result.Path, result.Asset, token, &wg)
		}
	}

	wg.Wait()

	return nil
}
