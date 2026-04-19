package utils

import (
	"encoding/json"
	"io"
	"main/globals"
	"os"
)

type Index map[string]IndexItem

func NewIndex[T IndexItem](data map[string]T) Index {
	index := make(Index)
	for id := range data {
		index[id] = data[id]
	}
	return index
}

type Pair struct {
	Key   string
	Value IndexItem
}

type AssetIndex map[string]AssetIndexItem

type AssetIndexItem struct {
	Hash string `json:"hash"`
}

func NewAssetIndexItem(hash string) AssetIndexItem {
	return AssetIndexItem{hash}
}

func (asset AssetIndexItem) Path() (string, bool) {
	return globals.ASSETS_PATH, true
}

type ModIndex map[string]ModIndexItem

type ModIndexItem struct {
	Id string `json:"id"`
	Hash string `json:"hash"`
	Version string `json:"version"`
	ModPath any `json:"path"`
}

func NewModIndexItem(
	id string,
	hash string,
	version string,
	path string,
) ModIndexItem {
	return ModIndexItem{id, hash, version, path}
}

func (mod ModIndexItem) Path() (string, bool) {
	if (mod.ModPath == nil) {
		return "", false
	} else {
		return mod.ModPath.(string), true
	}
}

type IndexItem interface {
	Path() (string, bool)
}

func ReadIndex[T any](path string, data *T) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return
	}
	defer file.Close()

	err = json.Unmarshal(byte_data, &data)
}

func (data Index) Write(path string) error {
	var file *os.File
	file, err := os.OpenFile(path, os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(path)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	json_string, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(json_string))
	if err != nil {
		return err
	}

	return nil
}

func (data Index) Merge(index chan Pair) Index {
	for e := range index {
		data[e.Key] = e.Value
	}
	return data
}
