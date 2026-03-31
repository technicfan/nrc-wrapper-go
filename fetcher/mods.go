package fetcher

import (
	"main/config"
	"main/globals"
	"main/mods"
	"main/utils"
	"os"
	"path/filepath"
)

func GetInstalledMods(
	root string,
	mod_dir string,
) (map[string]mods.Mod, bool) {
	files, _ := os.ReadDir(filepath.Join(root, mod_dir))
	index := utils.ReadIndex(filepath.Join(root, globals.MOD_INDEX))

	hashes := make(map[string]string)
	for _, f := range files {
		if !f.IsDir() &&
			(filepath.Ext(f.Name()) == ".jar" || filepath.Ext(f.Name()) == ".disabled") {
			entry, e := index[f.Name()]
			var hash string
			var err error
			if e {
				hash = entry["hash"]
			} else {
				hash, err = utils.Hash(filepath.Join(root, mod_dir, f.Name()))
			}
			if err == nil {
				hashes[hash] = f.Name()
			}
		}
	}

	updated := false
	results := make(map[string]mods.Mod)
	for entry_name, entry := range index {
		if name, exists := hashes[entry["hash"]]; exists {
			results[entry["id"]] = mods.NewMod(
				entry["hash"],
				entry["version"],
				entry["id"],
				name,
				mod_dir,
			)
			if entry_name != name {
				updated = true
			}
		} else {
			updated = true
		}
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
