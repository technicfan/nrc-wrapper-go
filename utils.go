package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

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

func cmp_mc_versions(
	a string,
	b string,
) int {
	if a == b {
		return 0
	}

	a_split := strings.Split(a, ".")
	b_split := strings.Split(b, ".")

	length := min(len(a_split), len(b_split))

	for i := range length {
		if a_split[i] == b_split[i] {
			continue
		}

		a_int, err := strconv.ParseInt(a_split[i], 10, 32)
		if err != nil {
			return 1
		}
		b_int, err := strconv.ParseInt(b_split[i], 10, 32)
		if err != nil {
			return -1
		}
		if a_int > b_int {
			return 1
		} else {
			return -1
		}
	}

	if len(a) > len(b) {
		return 1
	} else {
		return -1
	}
}

func print_packs(
	packs map[string]Pack,
) {
	fmt.Println("Available NRC packs:")
	for value, pack := range packs {
		var mc_versions []string
		mods, _, loaders := get_pack_data(pack, packs)
		for _, mod := range pack.Mods {
			for version := range mod.Compatibility {
				if !slices.Contains(mc_versions, version) && version != "1.8.9" {
					mc_versions = append(mc_versions, version)
				}
			}
		}
		slices.SortFunc(mc_versions, cmp_mc_versions)
		var loaders_string string
		var loaders_list []string
		for loader, version := range loaders {
			loaders_list = append(
				loaders_list, fmt.Sprintf("%s %s", loader, version),
			)
		}
		if len(loaders_list) > 0 {
			loaders_string = strings.Join(loaders_list, ", ")
		} else {
			loaders_string = "unknown"
		}
		fmt.Printf("- %s\n", pack.Name)
		fmt.Printf("  NRC_PACK: %s\n", value)
		fmt.Printf("  Description: %s\n", pack.Desc)
		fmt.Printf("  Compatible versions: %s\n", strings.Join(mc_versions, ", "))
		fmt.Printf("  Mod loaders: %s\n", loaders_string)
		fmt.Printf("  Mods: %v\n", len(append(pack.Mods, mods...)))
	}
}
