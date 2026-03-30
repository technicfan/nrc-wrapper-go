package fetcher

import (
	"main/config"
	"main/globals"
	"main/mod_entry"
	"main/utils"
	"os"
	"path/filepath"
)

func Get_installed_mods(
	root string,
	mod_dir string,
) (mod_entry.ModEntries, bool) {
	files, _ := os.ReadDir(filepath.Join(root, mod_dir))
	index := utils.Read_index(filepath.Join(root, globals.MOD_INDEX))

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
				hash, err = utils.Calc_hash(filepath.Join(root, mod_dir, f.Name()))
			}
			if err == nil {
				hashes[hash] = f.Name()
			}
		}
	}

	updated := false
	results := make(mod_entry.ModEntries)
	for entry_name, entry := range index {
		if name, exists := hashes[entry["hash"]]; exists {
			results[entry["id"]] = mod_entry.New(
				entry["hash"],
				entry["version"],
				entry["id"],
				name,
				mod_dir,
				"",
				"",
				false,
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

func Get_Mods(
	mods mod_entry.ModEntries,
	config config.Config,
) ([]utils.NrcResource, utils.Index, bool) {
	installed_mods, updated := Get_installed_mods("./", config.ModDir())
	mods_to_download, already_installed := mods.Get_missing_mods(
		installed_mods,
		config.ModDir(),
	)

	if len(mods_to_download) == 0 {
		return []utils.NrcResource{}, already_installed.Convert_to_index(), updated
	}

	var result []utils.NrcResource
	for id := range mods_to_download {
		result = append(result, mods_to_download[id])
	}

	return result, already_installed.Convert_to_index(), updated
}
