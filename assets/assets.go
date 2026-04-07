package assets

import (
	"crypto/md5"
	"fmt"
	"hash"
	"log"
	"main/globals"
	"main/utils"
	"path/filepath"
)

type Asset struct /*implements NrcResource*/ {
	pack string
	path string
	hash string
	root string
}

func (asset Asset) Url() string {
	return fmt.Sprintf("%s/%s/assets/%s", globals.NORISK_ASSETS_ENDPOINT, asset.pack, asset.path)
}

func (asset Asset) Path() string {
	return filepath.Join(asset.root, globals.ASSETS_PATH, asset.path)
}

func (asset Asset) AssetPath() string {
	return asset.path
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
	return 0
}

func (asset Asset) IsMissing(index utils.Index) (bool, bool) {
	if entry, e := index[asset.path]; e && entry["hash"] == asset.hash {
		return false, false
	}
	if hash, err := utils.Hash(asset.Path()); err == nil && hash == asset.hash {
		return false, true
	}
	return true, true
}

type Assets struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int    `json:"size"`
	} `json:"objects"`
}

func (assets Assets) Assets(pack string, root string) map[string]Asset {
	result := make(map[string]Asset)
	for path, asset := range assets.Objects {
		result[path] = Asset{pack, path, asset.Hash, root}
	}
	return result
}
