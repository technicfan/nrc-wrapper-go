package packs

import (
	"main/globals"
	"main/utils"
	"maps"
	"slices"
)

type LoaderSupport struct {
	LoaderVersion string
	Versions []string
}

type MetaPack struct {
	Name     string
	Desc     string
	Support map[string]LoaderSupport
	// Versions []string
	// Loaders  map[string]string
}

type MetaPacks struct {
	Packs    map[string]MetaPack
	Support map[string]LoaderSupport
	// Versions []string
	// Loaders  []string
	Names    []string
}

func (packs MetaPacks) CompatiblePacks(version string, loader string) ([]string, []string, bool) {
	has_main_pack := false
	var pack_ids, unique_pack_names []string
	for _, p := range globals.MAIN_PACKS {
		if s, e := packs.Packs[p].Support[loader]; e && slices.Contains(s.Versions, version) {
			has_main_pack = true
			pack_ids = append(pack_ids, p)
			unique_pack_names = append(
				unique_pack_names,
				utils.Unique(packs.Packs[p].Name, len(unique_pack_names)),
			)
		}
	}
	for _, i := range slices.Sorted(maps.Keys(packs.Packs)) {
		if s, e := packs.Packs[i].Support[loader]; e &&
			slices.Contains(s.Versions, version) && !slices.Contains(globals.MAIN_PACKS, i) {
			unique_pack_names = append(
				unique_pack_names,
				utils.Unique(packs.Packs[i].Name, len(unique_pack_names)),
			)
			pack_ids = append(pack_ids, i)
		}
	}
	return unique_pack_names, pack_ids, has_main_pack
}

func (packs MetaPacks) GenericSupport() map[string][]string {
	support := make(map[string][]string)
	for l := range packs.Support {
		support[l] = packs.Support[l].Versions
	}
	return support
}
