package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
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

func verify_asset(path string, data Asset, wg *sync.WaitGroup, results chan <- VerifiedAsset) {
	defer wg.Done()

	if hash, err := calc_hash(fmt.Sprintf("NoRiskClient/assets/%s", path)); err == nil && hash == data.Hash {
		results <- VerifiedAsset{true, "", Asset{}}
		return
	}

	results <- VerifiedAsset{false, path, data}
}

func load_assets(token string, wg1 *sync.WaitGroup) error {
	defer wg1.Done()
	metadata, err := get_asset_metadata("norisk-prod")
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	results := make(chan VerifiedAsset, len(metadata.Objects))

	log.Print("verifying assets")

	for i, v := range metadata.Objects {
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
			go download_single_asset("norisk-prod", result.Path, result.Asset, token, &wg)
		}
	}

	wg.Wait()

	return nil
}
