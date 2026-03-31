package fetcher

import (
	"main/config"
	"main/globals"
	"main/mods"
	"main/utils"
	"os"
	"path/filepath"
	"strings"
)

func GetInstalledMods(
	root string,
	mod_dir string,
) (map[string]mods.Mod, bool) {
	files, _ := os.ReadDir(filepath.Join(root, mod_dir))
	index := utils.ReadIndex(filepath.Join(root, globals.MOD_INDEX))

	updated := false
	results := make(map[string]mods.Mod)
	for _, f := range files {
		if !f.IsDir() &&
			(filepath.Ext(f.Name()) == ".jar" || filepath.Ext(f.Name()) == ".disabled") {
			name := f.Name()
			entry, e := index[name]
			if !e {
				switch filepath.Ext(f.Name()) {
				case ".jar": name = f.Name() + ".disabled"
				case ".disabled": name = strings.TrimSuffix(f.Name(), ".disabled")
				}
				entry, e = index[name]
				if e {
					updated = true
				}
			}
			if e {
				results[entry["id"]] = mods.NewMod(
					entry["hash"],
					entry["version"],
					entry["id"],
					f.Name(),
					mod_dir,
				)
				delete(index, name)
			}
		}
	}
	
	if len(index) != 0 {
		updated = true
	}

	return results, updated
}

func GetMods(
	mods mods.ModResources,
	config config.Config,
) ([]utils.NrcResource, utils.Index, bool) {
	installed_mods, updated := GetInstalledMods("./", config.ModDir())
	mods_to_download, already_installed := mods.GetMissing(
		installed_mods,
		config.ModDir(),
	)

	var result []utils.NrcResource
	for id := range mods_to_download {
		result = append(result, mods_to_download[id])
	}

	return result, already_installed.Index(), updated
}
