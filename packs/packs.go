package packs

import (
	"fmt"
	"main/config"
	"main/mods"
	"main/utils"
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
) (NrcMods, []string, map[string]LoaderSupport) {
	supported_versions := make(map[string]LoaderSupport)
	// loaders := make(map[string]string)
	for name, loader := range pack.Loader["default"] {
		// loaders[name] = loader.Version
		supported_versions[name] = LoaderSupport{loader.Version, []string{}}
	}
	var exclude []string
	if pack.Exclude != nil {
		for _, id := range pack.Exclude {
			exclude = append(exclude, id.(string))
		}
	}
	var mods []NrcMod
	var assets []string
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
			// if !slices.Contains(versions, version) {
			// 	versions = append(versions, version)
			// }
			for loader := range mod.Compatibility[version] {
				if v, e := supported_versions[loader]; e {
					if !slices.Contains(v.Versions, version) {
						v.Versions = append(v.Versions, version)
						supported_versions[loader] = v
					}
				} else {
					// loaders[loader] = "0"
					supported_versions[loader] = LoaderSupport{"0", []string{version}}
				}
			}
		}
	}

	return mods, assets, supported_versions
}

type Packs map[string]Pack

func (packs Packs) MetaPacks() MetaPacks {
	var pack_names []string
	global_support := make(map[string]LoaderSupport)
	metapacks := make(map[string]MetaPack)
	for i := range packs {
		var mc_versions []string
		pack := packs[i]
		pack_names = append(pack_names, i)
		_, _, support := pack.Details(packs)
		slices.SortFunc(mc_versions, utils.CmpVersions)
		for l := range support {
			slices.SortFunc(support[l].Versions, utils.CmpVersions)
			if v, e := global_support[l]; e {
				for _, version := range support[l].Versions {
					if !slices.Contains(v.Versions, version) {
						v.Versions = append(v.Versions, version)
						global_support[l] = v
					}
				}
			} else {
				global_support[l] = LoaderSupport{"0", support[l].Versions}
			}
		}
		metapacks[i] = MetaPack{pack.Name, pack.Desc, support}
	}

	for l := range global_support {
		slices.SortFunc(global_support[l].Versions, utils.CmpVersions)
	}

	return MetaPacks{metapacks, global_support, pack_names}
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
		if _, e := mod.Compatibility[config.Version()]; e {
			if compatibility, e := mod.Compatibility[config.Version()][config.Loader()]; e {
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
