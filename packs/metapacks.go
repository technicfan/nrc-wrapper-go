package packs

import (
	"main/globals"
	"main/utils"
	"maps"
	"slices"
)

type MetaPack struct {
	Name     string
	Desc     string
	Versions []string
	Loaders  map[string]string
}

type MetaPacks struct {
	Packs    map[string]MetaPack
	Versions []string
	Loaders  []string
	Names    []string
}

func (packs MetaPacks) Get_compatible_packs(version string, loader string) ([]string, []string, bool) {
	has_main_pack := false
	var pack_ids, unique_pack_names []string
	for _, p := range globals.MAIN_PACKS {
		if _, e := packs.Packs[p].Loaders[loader]; e && slices.Contains(packs.Packs[p].Versions, version) {
			has_main_pack = true
			pack_ids = append(pack_ids, p)
			unique_pack_names = append(
				unique_pack_names,
				utils.Make_unique(packs.Packs[p].Name, len(unique_pack_names)),
			)
		}
	}
	for _, i := range slices.Sorted(maps.Keys(packs.Packs)) {
		if _, e := packs.Packs[i].Loaders[loader]; e &&
			slices.Contains(packs.Packs[i].Versions, version) && !slices.Contains(globals.MAIN_PACKS, i) {
			unique_pack_names = append(
				unique_pack_names,
				utils.Make_unique(packs.Packs[i].Name, len(unique_pack_names)),
			)
			pack_ids = append(pack_ids, i)
		}
	}
	return unique_pack_names, pack_ids, has_main_pack
}
