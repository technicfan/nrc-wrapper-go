package fetcher

import (
	"encoding/json"
	"fmt"
	"main/assets"
	"main/globals"
	"main/utils"
	"maps"
	"net/http"
	"sync"
)

func get_asset_metadata_async(
	index int,
	pack string,
	wg *sync.WaitGroup,
	data chan<- map[int]map[string]assets.Asset,
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

	var pack_data assets.Assets
	if err := json.NewDecoder(response.Body).Decode(&pack_data); err != nil {
		return
	}

	data <- map[int]map[string]assets.Asset{index: pack_data.Assets(pack)}
}

func GetAssets(packs []string) ([]utils.NrcResource, utils.Index, bool) {
	var wg sync.WaitGroup
	data := make(chan map[int]map[string]assets.Asset, len(packs))
	for i, pack := range packs {
		wg.Add(1)
		go get_asset_metadata_async(i, pack, &wg, data)
	}

	existing_index := utils.ReadIndex(globals.ASSET_INDEX)

	go func() {
		wg.Wait()
		close(data)
	}()

	final_data := make(map[int]map[string]assets.Asset)
	for obj := range data {
		maps.Copy(final_data, obj)
	}
	merged := make(map[string]assets.Asset)
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
			existing_index[asset.Path()] = map[string]string{"hash": asset.ExpectedHash()}
		}
	}

	return missing_assets, existing_index, index_updated
}
