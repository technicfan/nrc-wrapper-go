package packs

import (
	"fmt"
	"main/config"
	"main/mods"
	"main/utils"
	"maps"
	"slices"
	"strings"
)

// Pack/Packs
// The modpacks that come directly from the nrc api

type Pack struct {
	// display
	Name string `json:"displayName"`
	Desc string `json:"description"`
	// the id of the packs parent (e.g. norisk-prod)
	Inherits []string `json:"inheritsFrom"`
	// not used right know but here for good measure
	Exclude []any `json:"excludeMods"`
	// mods of the pack
	Mods NrcMods `json:"mods"`
	// asset packs needed for this pack
	Assets []string `json:"assets"`
	// the loader needed
	// currently only fabric
	Loader map[string]map[string]struct {
		Version string `json:"version"`
	} `json:"loaderPolicy"`
}

func (pack Pack) Details(
	packs map[string]Pack,
) (NrcMods, []string, []string, map[string]string) {
	loaders := make(map[string]string)
	for name, loader := range pack.Loader["default"] {
		loaders[name] = loader.Version
	}
	var exclude []string
	if pack.Exclude != nil {
		for _, id := range pack.Exclude {
			exclude = append(exclude, id.(string))
		}
	}
	var mods []NrcMod
	var assets, versions []string
	for _, inherited_pack := range pack.Inherits {
		for i := range packs[inherited_pack].Mods {
			mod := &packs[inherited_pack].Mods[i]
			if !(slices.Contains(exclude, mod.Id) ||
				slices.ContainsFunc(
					pack.Mods, func(entry NrcMod) bool { return entry.Id == mod.Id },
				)) {
				mods = append(mods, *mod)
			}
		}
		for _, asset_pack := range packs[inherited_pack].Assets {
			if !slices.Contains(assets, asset_pack) &&
				!slices.Contains(pack.Assets, asset_pack) {
				assets = append(assets, asset_pack)
			}
		}
	}
	assets = append(assets, pack.Assets...)

	for _, mod := range pack.Mods {
		for version := range mod.Compatibility {
			if !slices.Contains(versions, version) {
				versions = append(versions, version)
			}
			for loader := range mod.Compatibility[version] {
				if _, e := loaders[loader]; !e {
					loaders[loader] = "0"
				}
			}
		}
	}

	return mods, assets, versions, loaders
}

type Packs map[string]Pack

func (packs Packs) MetaPacks() MetaPacks {
	var global_versions []string
	var global_loaders []string
	var pack_names []string
	metapacks := make(map[string]MetaPack)
	for i := range packs {
		var mc_versions []string
		pack := packs[i]
		pack_names = append(pack_names, i)
		_, _, mc_versions, loaders := pack.Details(packs)
		slices.SortFunc(mc_versions, utils.CmpVersions)
		for l := range loaders {
			if !slices.Contains(global_loaders, l) {
				global_loaders = append(global_loaders, l)
			}
		}
		for i := range mc_versions {
			if !slices.Contains(global_versions, mc_versions[i]) {
				global_versions = append(global_versions, mc_versions[i])
			}
		}
		metapacks[i] = MetaPack{pack.Name, pack.Desc, mc_versions, loaders}
	}

	return MetaPacks{metapacks, global_versions, global_loaders, pack_names}
}

func (packs Packs) Print() {
	fmt.Println("Available NRC packs:")
	meta := packs.MetaPacks().Packs
	for _, key := range slices.Sorted(maps.Keys(meta)) {
		var loaders_string string
		var loaders_list []string
		for loader, version := range meta[key].Loaders {
			if version != "0" {
				loaders_list = append(
					loaders_list, fmt.Sprintf("%s %s", loader, version),
				)
			} else {
				loaders_list = append(loaders_list, loader)
			}
		}
		if len(loaders_list) > 0 {
			loaders_string = strings.Join(loaders_list, ", ")
		} else {
			loaders_string = "unknown"
		}
		fmt.Printf("- %s\n", meta[key].Name)
		fmt.Printf("  NRC_PACK: %s\n", key)
		fmt.Printf("  Description: %s\n", meta[key].Desc)
		fmt.Printf("  Compatible versions: %s\n", strings.Join(meta[key].Versions, ", "))
		fmt.Printf("  Mod loaders: %s\n", loaders_string)
	}
}

// NoriskMod(s)
// The mods that come from the api

type NrcMod struct {
	// Mod identifier
	Id string `json:"id"`
	// Pretty name
	Name string `json:"displayName"`
	// Source (modrinth/maven/url)
	Source map[string]string `json:"source"`
	// Different versions for different Minecraft versions
	// also supports another source field to override the one for the whole mod
	Compatibility map[string]map[string]map[string]any `json:"compatibility"`
}

func (mod NrcMod) build_url(
	mod_version string,
	repos map[string]string,
) (string, string, string) {
	var url, alt_url string
	switch mod.Source["type"] {
	case "url":
		url = mod_version
	case "modrinth":
		version := mod_version
		if !strings.Contains(mod_version, "-") {
			version = strings.Replace(mod_version, ",", "-", 1)
		}
		filename := fmt.Sprintf("%s-%s.jar", mod.Source["projectId"], version)
		url = fmt.Sprintf(
			"%smaven/modrinth/%s/%s/%s",
			repos[mod.Source["type"]], mod.Source["projectId"], version, filename,
		)
		alt_url = strings.ReplaceAll(url, mod.Source["projectId"], mod.Source["projectSlug"])
	default:
		group_path := strings.ReplaceAll(mod.Source["groupId"], ".", "/")
		filename := fmt.Sprintf("%s-%s.jar", mod.Source["artifactId"], mod_version)
		url = fmt.Sprintf(
			"%s%s/%s/%s/%s",
			repos[mod.Source["repositoryRef"]], group_path,
			mod.Source["artifactId"], mod_version, filename,
		)
	}

	return url, alt_url, fmt.Sprintf("%s-%s.jar", mod.Id, mod_version)
}

type NrcMods []NrcMod

func (nrc_mods NrcMods) CompatibleMods(
	config config.Config,
	repos map[string]string,
) mods.ModResources {
	result := make(mods.ModResources)
	for _, mod := range nrc_mods {
		if _, exists := mod.Compatibility[config.Version]; exists {
			if compatibility, exists := mod.Compatibility[config.Version][config.Loader]; exists {
				if compatibility["source"] != nil {
					source := compatibility["source"].(map[string]any)
					for k, v := range source {
						mod.Source[k] = v.(string)
					}
				}
				url, alt_url, filename := mod.build_url(
					compatibility["identifier"].(string),
					repos,
				)
				if compatibility["filename"] != nil {
					filename = compatibility["filename"].(string)
				}
				result[mod.Id] = mods.NewModResource(
					"",
					compatibility["identifier"].(string),
					mod.Id,
					filename,
					config.ModDir(),
					url,
					alt_url,
					mod.Source["type"] != "url",
				)
			}
		}
	}

	return result
}

func (nrc_mods NrcMods) DisplayNames(mods map[string]mods.Mod) map[string]string {
	result := make(map[string]string)
	for i := range nrc_mods {
		if _, e := mods[nrc_mods[i].Id]; e {
			result[nrc_mods[i].Id] = nrc_mods[i].Name
		}
	}
	return result
}
