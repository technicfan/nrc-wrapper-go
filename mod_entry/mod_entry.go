package mod_entry

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"log"
	"main/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ModEntry struct {
	// SHA1 Hash
	hash string
	// Version number
	version string
	// id
	id       string
	filename string
	// old file if it was replaced
	old_file      string
	path          string
	url           string
	alt_url       string
	use_alt_url   bool
	check_hash    bool
}

func New(
	hash string,
	version string,
	id string,
	filename string,
	path string,
	url string,
	alt_url string,
	check_hash bool,
) ModEntry {
	return ModEntry{
		hash,
		version,
		id,
		filename,
		"",
		path,
		url,
		alt_url,
		false,
		check_hash,
	}
}

func (mod ModEntry) Url() string {
	if mod.use_alt_url && mod.alt_url != "" {
		mod.use_alt_url = false
		return mod.alt_url
	}
	mod.use_alt_url = true
	return mod.url
}

func (mod ModEntry) Path() string {
	return filepath.Join(mod.path, mod.filename)
}

func (mod ModEntry) Filename() string {
	return mod.filename
}

func (mod ModEntry) Enabled() bool {
	return strings.HasSuffix(mod.filename, ".jar")
}

func (mod ModEntry) ExpectedHash() string {
	if mod.check_hash {
		hash_response, err := http.Get(fmt.Sprintf("%s.sha1", mod.Url()))
		if err != nil {
			return ""
		}
		if hash_response.StatusCode != http.StatusOK {
			log.Printf("Maven does not provide a sha1 hash for %s", mod.Filename)
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

func (mod ModEntry) HashObj() hash.Hash {
	return sha1.New()
}

func (mod ModEntry) Download() error {
	err := utils.Download(mod)
	if err != nil && err.Error() == "HTTP 404" {
		err = utils.Download(mod)
	}
	if err == nil {
		log.Printf("Downloaded %s", mod.Filename())
		if mod.Filename() != mod.old_file && mod.Filename() != "" && mod.old_file != "" {
			os.Remove(filepath.Join(mod.path, mod.old_file))
			log.Printf("Removed old file %s", mod.old_file)
		}
	}
	return err
}

func (mod ModEntry) IndexPair() utils.Pair {
	hash, _ := utils.Calc_hash(mod.Path())
	return utils.Pair{mod.filename, map[string]string{"id": mod.id, "hash": hash, "version": mod.version}}
}

func (mod ModEntry) Type() int {
	return 0
}

func (mod *ModEntry) SetOldFile(name string) {
	mod.old_file = name
	if strings.HasSuffix(name, ".disabled") && mod.Enabled() {
		mod.filename += ".disabled"
	}
}

type ModEntries map[string]ModEntry

func (mods ModEntries) Get_missing_mods(
	installed_mods ModEntries,
	path string,
) (ModEntries, ModEntries) {
	result, removed := make(ModEntries), make(ModEntries)
	for _, mod := range mods {
		if installed_mod, exists := installed_mods[mod.id]; exists {
			if mod.version != installed_mod.version {
				mod.SetOldFile(installed_mod.Filename())
				result[mod.id] = mod
			} else {
				mod.hash = installed_mod.hash
				removed[mod.id] = mod
			}
			delete(installed_mods, mod.id)
		} else {
			result[mod.id] = mod
		}
	}

	for _, mod := range installed_mods {
		os.Remove(mod.Path())
		log.Printf("Removed left over file %s", mod.Filename())
	}

	return result, removed
}

func (mods ModEntries) Convert_to_index() utils.Index {
	results := make(utils.Index)
	for _, mod := range mods {
		info := make(map[string]string)
		info["hash"] = mod.hash
		info["version"] = mod.version
		results[mod.id] = info
	}

	return results
}
