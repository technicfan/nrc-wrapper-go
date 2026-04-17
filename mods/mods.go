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

func (mod ModResource) IndexPair() utils.Pair {
	hash, _ := utils.Hash(mod.Path())
	return utils.Pair{
		Key: mod.filename,
		Value: map[string]string{
			"id": mod.id,
			"hash": hash,
			"version": mod.version,
			"path": mod.mod_dir,
		},
	}
}

func (mod ModResource) Type() int {
	return 1
}

type ModResources map[string]ModResource

func (mods ModResources) GetMissing(
	installed_mods map[string]Mod,
	path string,
) (ModResources, ModResources) {
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

	for _, mod := range installed_mods {
		os.Remove(mod.Path())
		log.Printf("Removed left over file %s", mod.Filename())
	}

	return missing, installed
}

func (mods ModResources) Index() utils.Index {
	results := make(utils.Index)
	for _, mod := range mods {
		results[mod.filename] = map[string]string{
			"hash": mod.hash,
			"version": mod.version,
			"id": mod.id,
			"path": mod.mod_dir,
		}
	}
	return results
}

func GetInstalledMods(
	root string,
	mod_dir string,
) (map[string]Mod, bool) {
	files, _ := os.ReadDir(filepath.Join(root, mod_dir))
	index := utils.ReadIndex(filepath.Join(root, globals.MOD_INDEX))

	updated := false
	results := make(map[string]Mod)
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
				path, e := entry["path"]
				if !e {
					path = mod_dir
				}
				results[entry["id"]] = Mod{
					entry["hash"],
					entry["version"],
					entry["id"],
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
		for file, entry := range index {
			if path, e := entry["path"]; e && path != mod_dir {
				os.Remove(filepath.Join(path, file))
				log.Printf("Removed left over file %s", file)
				if f, _ := os.ReadDir(path); path != "mods" && len(f) == 0 {
					os.Remove(path)
				}
			}
		}
	}

	return results, updated
}
