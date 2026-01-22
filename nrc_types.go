package main

import (
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Pack/Packs
// The modpacks that come directly from the nrc api

type Pack struct {
	// display
	Name     string     `json:"displayName"`
	Desc     string     `json:"description"`
	// the id of the packs parent (e.g. norisk-prod)
	Inherits []string   `json:"inheritsFrom"`
	// not used right know but here for good measure
	Exclude  []any      `json:"excludeMods"`
	// mods of the pack
	Mods     NoriskMods `json:"mods"`
	// asset packs needed for this pack
	Assets   []string   `json:"assets"`
	// the loader needed
	// currently only fabric
	Loader   map[string]map[string]struct {
		Version string `json:"version"`
	} `json:"loaderPolicy"`
}

func (pack *Pack) get_details(
	packs map[string]Pack,
) (NoriskMods, []string, map[string]string) {
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

type Packs map[string]Pack

func (packs Packs) print_packs() {
	fmt.Println("Available NRC packs:")
	for _, key := range slices.Sorted(maps.Keys(packs)) {
		var mc_versions []string
		var mod_count int
		pack := packs[key]
		mods, _, loaders := pack.get_details(packs)
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

type Versions struct {
	Packs        Packs             `json:"packs"`
	Repositories map[string]string `json:"repositories"`
}

// NoriskMod(s)
// The mods that come from the api

type NoriskMod struct {
	// Mod identifier
	Id            string                               `json:"id"`
	// Pretty name
	Name          string                               `json:"displayName"`
	// Source (modrinth/maven/url)
	Source        map[string]string                    `json:"source"`
	// Different versions for different Minecraft versions
	// also supports another source field to override the one for the whole mod
	Compatibility map[string]map[string]map[string]any `json:"compatibility"`
}

type NoriskMods []NoriskMod

func (nrc_mods NoriskMods) get_compatible_mods(
	mc_version string,
	loader string,
) (ModEntries, error) {
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

// ModEntry/ModEntries

type ModEntry struct {
	// MD5 Hash
	Hash          string
	// Version number
	Version       string
	// id
	Id            string
	Filename      string
	// old file if it was replaced
	OldFile       string
	// modrinth/maven/url
	SourceType    string
	// maven repo reference
	RepositoryRef string
	// maven group
	GroupId       string
	// modrinth id
	ModrinthId    string
	// modrinth mod name
	ProjectSlug   string
	// maven id
	MavenId       string
}

func (mod *ModEntry) build_maven_url(
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

type ModEntries []ModEntry

func (mods ModEntries) get_missing_mods(
	installed_mods map[string]map[string]string,
	path string,
) (ModEntries, ModEntries) {
	var result ModEntries
	var removed ModEntries
	for _, mod := range mods {
		if _, exists := installed_mods[mod.Id]; exists {
			if mod.Version != installed_mods[mod.Id]["version"] {
				mod.OldFile = installed_mods[mod.Id]["filename"]
				result = append(result, mod)
			} else {
				mod.Hash = installed_mods[mod.Id]["hash"]
				removed = append(removed, mod)
			}
			delete(installed_mods, mod.Id)
		} else {
			result = append(result, mod)
		}
	}

	for _, file := range installed_mods {
		os.Remove(filepath.Join(path, file["filename"]))
		log.Printf("Removed left over file %s", file["filename"])
	}

	return result, removed
}

func (mods ModEntries) convert_to_index() Index {
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
