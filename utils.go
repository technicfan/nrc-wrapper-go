package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gen2brain/beeep"
)

func calc_hash(
	path string,
) (string, error) {
	var file, err = os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	var hash = md5.New()
	_, err = hash.Write(data)
	if err != nil {
		return "", err
	}

	var bytesHash = hash.Sum(nil)
	return hex.EncodeToString(bytesHash[:]), nil
}

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

func build_maven_url(
	mod ModEntry,
	repos map[string]string,
) (string, string, string) {
	var url, alt_url string
	if mod.SourceType == "modrinth" {
		version := mod.Version
		if !strings.Contains(mod.Version, "-") {
			version = strings.Replace(mod.Version, ",", "-", 1)
		}
		filename := fmt.Sprintf("%s-%s.jar", mod.ModrinthId, version)
		url = fmt.Sprintf(
			"%smaven/modrinth/%s/%s/%s",
			repos[mod.SourceType], mod.ModrinthId, version, filename,
		)
		alt_url = strings.ReplaceAll(url, mod.ModrinthId, mod.ProjectSlug)
	} else {
		group_path := strings.ReplaceAll(mod.GroupId, ".", "/")
		filename := fmt.Sprintf("%s-%s.jar", mod.MavenId, mod.Version)
		url = fmt.Sprintf(
			"%s%s/%s/%s/%s",
			repos[mod.RepositoryRef], group_path, mod.MavenId, mod.Version, filename,
		)
	}

	return url, alt_url, fmt.Sprintf("%s-%s.jar", mod.Id, mod.Version)
}

func get_compatible_nrc_mods(
	mc_version string,
	loader string,
	nrc_mods []NoriskMod,
) ([]ModEntry, error) {
	var mods []ModEntry
	for _, mod := range nrc_mods {
		if _, exists := mod.Compatibility[mc_version]; exists {
			var filename string
			if mod.Compatibility[mc_version][loader]["source"] != nil {
				source := mod.Compatibility[mc_version][loader]["source"].(map[string]any)
				for k, v := range source {
					mod.Source[k] = v.(string)
				}
			}
			if mod.Compatibility[mc_version][loader]["filename"] != nil {
				filename = mod.Compatibility[mc_version][loader]["filename"].(string)
			}
			mods = append(
				mods,
				ModEntry{
					"",
					mod.Compatibility[mc_version][loader]["identifier"].(string),
					mod.Id,
					filename,
					"",
					mod.Source["type"],
					mod.Source["repositoryRef"],
					mod.Source["groupId"],
					mod.Source["projectId"],
					mod.Source["projectSlug"],
					mod.Source["artifactId"],
				},
			)
		}
	}

	return mods, nil
}

func convert_to_index(
	mods []ModEntry,
) []map[string]string {
	var results []map[string]string
	for _, mod := range mods {
		info := make(map[string]string)
		info["id"] = mod.Id
		info["hash"] = mod.Hash
		info["version"] = mod.Version
		results = append(results, info)
	}

	return results
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
	for _, key := range slices.Sorted(maps.Keys(packs)) {
		var mc_versions []string
		var mod_count int
		pack := packs[key]
		mods, _, loaders := get_pack_data(pack, packs)
		for _, mod := range pack.Mods {
			for version := range mod.Compatibility {
				if !slices.Contains(mc_versions, version) && cmp_mc_versions("1.21", version) < 1 {
					mc_versions = append(mc_versions, version)
				}
			}
		}
		for _, mod := range append(pack.Mods, mods...) {
			count := false
			for version := range mod.Compatibility {
				if cmp_mc_versions("1.21", version) < 1 {
					count = true
				}
			}
			if count {
				mod_count++
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
		fmt.Printf("  NRC_PACK: %s\n", key)
		fmt.Printf("  Description: %s\n", pack.Desc)
		fmt.Printf("  Compatible versions: %s\n", strings.Join(mc_versions, ", "))
		fmt.Printf("  Mod loaders: %s\n", loaders_string)
		fmt.Printf("  Mods: %v\n", mod_count)
	}
}

func notify(
	msg string,
	error bool,
	notify bool,
) {
	beeep.AppName = "nrc-wrapper-go"
	if error {
		if notify {
			err := beeep.Notify("Error", msg, "")
			if err != nil {
				log.Fatalf("Notify failed: %s", err.Error())
			}
		}
		log.Fatal(msg)
	} else {
		if notify {
			err := beeep.Notify("Info", msg, "")
			if err != nil {
				log.Fatalf("Notify failed: %s", err.Error())
			}
		}
		log.Println(msg)
	}
}
