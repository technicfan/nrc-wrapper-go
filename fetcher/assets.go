package fetcher

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"main/globals"
	"main/utils"
	"maps"
	"net/http"
	"path/filepath"
	"sync"
)

type Asset struct {
	pack string
	path string
	hash string
}

type Assets struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int    `json:"size"`
	} `json:"objects"`
}

func (asset Asset) Url() string {
	return fmt.Sprintf("%s/%s/assets/%s", globals.NORISK_ASSETS_URL, asset.pack, asset.path)
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
	err := utils.Download(asset)
	if err == nil {
		log.Printf("Downloaded %s/%s", asset.pack, asset.Filename())
	}
	return err
}

func (asset Asset) IndexPair() utils.Pair {
	return utils.Pair{Key: asset.path, Value: map[string]string{"hash": asset.hash}}
}

func (asset Asset) Type() int {
	return 1
}

func (asset Asset) IsMissing(index utils.Index) (bool, bool) {
	if entry, e := index[asset.path]; e && entry["hash"] == asset.hash {
		return false, false
	}
	if hash, err := utils.Calc_hash(asset.Path()); err == nil && hash == asset.hash {
		return false, true
	}
	return true, true
}

func get_asset_metadata_async(
	index int,
	pack string,
	wg *sync.WaitGroup,
	data chan<- map[int]map[string]Asset,
) {
	defer wg.Done()

	response, err := http.Get(fmt.Sprintf("%s/launcher/pack/%s", globals.NORISK_API_URL, pack))
	if err != nil {
		return
	}
	if response.StatusCode != http.StatusOK {
		return
	}
	defer response.Body.Close()

	var metadata Assets
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return
	}

	results := make(map[string]Asset)
	for path, obj := range metadata.Objects {
		asset := Asset{pack, path, obj.Hash}
		results[path] = asset
	}

	data <- map[int]map[string]Asset{index: results}
}

func Get_assets(packs []string) ([]utils.NrcResource, utils.Index, bool) {
	var wg sync.WaitGroup
	data := make(chan map[int]map[string]Asset, len(packs))
	for i, pack := range packs {
		wg.Add(1)
		go get_asset_metadata_async(i, pack, &wg, data)
	}

	existing_index := utils.Read_index(globals.ASSET_INDEX)

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

	index_updated := false
	var missing_assets []utils.NrcResource
	for _, asset := range merged {
		missing, untracked := asset.IsMissing(existing_index)
		if missing {
			missing_assets = append(missing_assets, asset)
		} else if untracked {
			index_updated = true
			existing_index[asset.path] = map[string]string{"hash": asset.hash}
		}
	}

	return missing_assets, existing_index, index_updated
}
