package mods

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"log"
	"main/globals"
	"main/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Mod struct {
	hash string
	version string
	id string
	filename string
	mod_dir string
	root string
}

func NewMod(
	hash string,
	version string,
	id string,
	filename string,
	mod_dir string,
	root string,
) Mod {
	return Mod{hash, version, id, filename, mod_dir, root}
}

func (mod Mod) Path() string {
	return filepath.Join(mod.root, mod.mod_dir, mod.filename)
}

func (mod Mod) Filename() string {
	return mod.filename
}

func (mod Mod) Enabled() bool {
	return strings.HasSuffix(mod.filename, ".jar")
}

func (mod Mod) IndexPair() utils.Pair {
	hash, _ := utils.Hash(mod.Path())
	return utils.Pair{
		Key: mod.filename,
		Value: utils.NewModIndexItem(
			mod.id,
			hash,
			mod.version,
			mod.mod_dir,
		),
	}
}

type ModResource struct /*implements NrcResource*/ {
	*Mod
	url           string
	alt_url       string
	use_alt_url   bool
	check_hash    bool
}

func NewModResource(
	hash string,
	version string,
	id string,
	filename string,
	mod_dir string,
	root string,
	url string,
	alt_url string,
	check_hash bool,
) ModResource {
	return ModResource{
		&Mod{hash, version, id, filename, mod_dir, root},
		url,
		alt_url,
		false,
		check_hash,
	}
}

func (mod ModResource) Url() string {
	if mod.use_alt_url {
		return mod.alt_url
	}
	return mod.url
}

func (mod ModResource) ExpectedHash() string {
	if mod.check_hash {
		hash_response, err := http.Get(fmt.Sprintf("%s.sha1", mod.Url()))
		if err != nil {
			return ""
		}
		if hash_response.StatusCode != http.StatusOK {
			log.Printf("Maven does not provide a sha1 hash for %s", mod.filename)
		} else {
			defer hash_response.Body.Close()

			hash_body, err := io.ReadAll(hash_response.Body)
			if err != nil {
				return ""
			}
			return string(hash_body)
		}
	}
	return ""
}

func (mod ModResource) HashObj() hash.Hash {
	return sha1.New()
}

func (mod ModResource) Download() error {
	err := utils.Download(mod)
	if err != nil && err.Error() == "HTTP 404" && mod.alt_url != "" {
		mod.use_alt_url = true
		err = utils.Download(mod)
	}
	if err == nil {
		log.Printf("Downloaded %s", mod.Filename())
	}
	return err
}

func (mods ModResources) Index() utils.Index {
	results := make(utils.Index)
	for _, mod := range mods {
		results[mod.filename] = mod.IndexPair().Value
	}
	return results
}

func (mod ModResource) Type() int {
	return 1
}

type ModResources map[string]ModResource

func (mods ModResources) GetMissing(
	installed_mods map[string]Mod,
	path string,
) (ModResources, ModResources, utils.Index) {
	missing, installed := make(ModResources), make(ModResources)
	for _, mod := range mods {
		if installed_mod, exists := installed_mods[mod.id]; exists {
			if mod.version != installed_mod.version {
				if !installed_mod.Enabled() {
					if mod.Enabled() {
						mod.filename += ".disabled"
					}
				}
				os.Remove(installed_mod.Path())
				log.Printf("Removed old file %s", installed_mod.Filename())
				missing[mod.id] = mod
			} else {
				mod.hash = installed_mod.hash
				installed[mod.id] = mod
			}
			delete(installed_mods, mod.id)
		} else {
			missing[mod.id] = mod
		}
	}

	index := make(utils.Index)
	for id, _ := range installed_mods {
		index[installed_mods[id].filename] = installed_mods[id].IndexPair().Value
	}

	return missing, installed, index
}

func GetInstalledMods(
	root string,
	mod_dir string,
) (map[string]Mod, utils.Index, bool) {
	files, _ := os.ReadDir(filepath.Join(root, mod_dir))
	index := make(utils.ModIndex)
	utils.ReadIndex(filepath.Join(root, globals.MOD_INDEX), &index)

	updated := false
	result := make(map[string]Mod)
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
				path, e := entry.Path()
				if !e {
					path = mod_dir
				}
				result[entry.Id] = Mod{
					entry.Hash,
					entry.Version,
					entry.Id,
					f.Name(),
					path,
					root,
				}
				delete(index, name)
			}
		}
	}

	if len(index) != 0 {
		updated = true
	}

	return result, utils.NewIndex(index), updated
}
