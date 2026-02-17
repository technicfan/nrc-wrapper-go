package main

import (
	"errors"
	"io/fs"
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

func (packs *MetaPacks) get_compatible_packs(version string, loader string) ([]string, []string) {
	var index int
	var unique_pack_names []string
	pack_ids := MAIN_PACKS
	for i, p := range MAIN_PACKS {
		if _, e := packs.Packs[p].Loaders[loader]; e && slices.Contains(packs.Packs[p].Versions, version) {
			unique_pack_names = append(unique_pack_names, make_unique(packs.Packs[p].Name, index))
			index++
		} else {
			pack_ids = slices.Delete(pack_ids, i, i+1)
		}
	}
	for i := range packs.Packs {
		if _, e := packs.Packs[i].Loaders[loader]; e &&
			slices.Contains(packs.Packs[i].Versions, version) && !slices.Contains(MAIN_PACKS, i) {
			unique_pack_names = append(unique_pack_names, make_unique(packs.Packs[i].Name, index))
			pack_ids = append(pack_ids, i)
			index++
		}
	}
	return unique_pack_names, pack_ids
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
