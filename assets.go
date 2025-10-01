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

func load_assets(
	packs []string,
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

	results := make(chan map[string]string, len(merged))
	for i, v := range merged {
		wg.Add(1)
		go verify_asset(i, v, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	if len(results) != 0 {
		log.Println("Downloading missing assets")
	}

	limiter := make(chan struct{}, 20)
	for result := range results {
		wg.Add(1)
		go download_single_asset(
			result["pack"],
			result["path"],
			result["hash"],
			&wg,
			limiter,
		)
	}

	wg.Wait()
}
