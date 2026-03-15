package main

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
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

func (packs *MetaPacks) get_compatible_packs(version string, loader string) ([]string, []string, bool) {
	has_main_pack := false
	var pack_ids, unique_pack_names []string
	for _, p := range MAIN_PACKS {
		if _, e := packs.Packs[p].Loaders[loader]; e && slices.Contains(packs.Packs[p].Versions, version) {
			has_main_pack = true
			pack_ids = append(pack_ids, p)
			unique_pack_names = append(
				unique_pack_names,
				make_unique(packs.Packs[p].Name, len(unique_pack_names)),
			)
		}
	}
	for _, i := range slices.Sorted(maps.Keys(packs.Packs)) {
		if _, e := packs.Packs[i].Loaders[loader]; e &&
			slices.Contains(packs.Packs[i].Versions, version) && !slices.Contains(MAIN_PACKS, i) {
			unique_pack_names = append(
				unique_pack_names,
				make_unique(packs.Packs[i].Name, len(unique_pack_names)),
			)
			pack_ids = append(pack_ids, i)
		}
	}
	return unique_pack_names, pack_ids, has_main_pack
}

func make_unique(str string, index int) string {
	var builder strings.Builder
	builder.WriteString(str)
	for range index {
		builder.WriteRune('\u200d')
	}
	return builder.String()
}

func get_launcher_dirs() (map[string][]string, []string) {
	dirs, order := get_const_dirs()
	for i, l := range order {
		if l == "" {
			continue
		}
		_, err := os.Stat(dirs[l][0])
		if err != nil && errors.Is(err, fs.ErrNotExist) {
			order = slices.Delete(order, i, i+1)
		} else if err == nil && strings.HasPrefix(l, "Prism") {
			dir, err := get_prism_instance_dir(dirs[l][0])
			if dirs["Prism Launcher"][0] == dir {
				i := slices.Index(order, "Prism Launcher")
				order = slices.Delete(order, i, i+1)
			}
			if err != nil {
				order = slices.Delete(order, i, i+1)
			}
			dirs[l][0] = dir
		}
	}

	return dirs, order
}
