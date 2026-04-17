package fetcher

import (
	"log"
	"main/api"
	"main/assets"
	"main/globals"
	"main/utils"
	"maps"
	"os"
	"path/filepath"
	"sync"
)

func get_assets(
	root string,
	packs []string,
	api_endpoint string,
) ([]utils.NrcResource, utils.Index, bool) {
	var wg sync.WaitGroup
	data := make(chan map[int]map[string]assets.Asset, len(packs))
	for i, pack := range packs {
		wg.Add(1)
		go api.GetAssets(i, pack, root, api_endpoint, &wg, data)
	}

	existing_index := utils.ReadIndex(filepath.Join(root, globals.ASSET_INDEX))

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
	for path := range existing_index {
		if _, e := merged[path]; !e {
			index_updated = true
			delete(existing_index, path)
			os.Remove(filepath.Join(root, globals.ASSETS_PATH, path))
			log.Printf("Removed left over file %s", filepath.Base(path))
		}
	}
	for _, asset := range merged {
		missing, untracked := asset.IsMissing(existing_index)
		if missing {
			missing_assets = append(missing_assets, asset)
			if (!untracked) {
				index_updated = true
				delete(existing_index, asset.AssetPath())
			}
		} else if untracked {
			index_updated = true
			existing_index[asset.AssetPath()] = map[string]string{"hash": asset.ExpectedHash()}
		}
	}

	return missing_assets, existing_index, index_updated
}
