package main

import "slices"

func get_pack_data(
	pack Pack,
	packs map[string]Pack,
) ([]NoriskMod, []string, map[string]string) {
	loaders := make(map[string]string)
	for name, loader := range pack.Loader["default"] {
		loaders[name] = loader.Version
	}
	var mods []NoriskMod
	var assets []string
	for _, inherited_pack := range pack.Inherits {
		mods = append(mods, packs[inherited_pack].Mods...)
		for _, asset_pack := range packs[inherited_pack].Assets {
			if !slices.Contains(assets, asset_pack) &&
				!slices.Contains(pack.Assets, asset_pack) {
				assets = append(assets, asset_pack)
			}
		}
	}
	assets = append(assets, pack.Assets...)

	return mods, assets, loaders
}
