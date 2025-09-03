package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
)

func verify_asset(path string, data Asset, wg *sync.WaitGroup, results chan<- VerifiedAsset) {
	defer wg.Done()
	var file, err = os.Open(fmt.Sprintf("NoRiskClient/assets/%s", path))
	if err != nil {
		results <- VerifiedAsset{false, path, data}
	}
	defer file.Close()

	var hash = md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		results <- VerifiedAsset{false, path, data}
	}

	var bytesHash = hash.Sum(nil)
	if hex.EncodeToString(bytesHash[:]) == data.Hash {
		results <- VerifiedAsset{true, "", Asset{}}
	}

	results <- VerifiedAsset{false, path, data}
}

func load_assets(token string) error {
	metadata, err := get_asset_metadata("norisk-prod")
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	results := make(chan VerifiedAsset, len(metadata.Objects))

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

	return nil
}
