package mods

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

type Mod struct {
	hash string
	version string
	id string
	filename string
	path string
}

func NewMod(
	hash string,
	version string,
	id string,
	filename string,
	path string,
) Mod {
	return Mod{hash, version, id, filename, path}
}

func (mod Mod) Path() string {
	return filepath.Join(mod.path, mod.filename)
}

func (mod Mod) Filename() string {
	return mod.filename
}

func (mod Mod) Enabled() bool {
	return strings.HasSuffix(mod.filename, ".jar")
}

type ModResource struct /*implements NewModResource*/ {
	*Mod
	old_file      string
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
	path string,
	url string,
	alt_url string,
	check_hash bool,
) ModResource {
	return ModResource{
		&Mod{hash, version, id, filename, path},
		"",
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
		if mod.Filename() != mod.old_file && mod.Filename() != "" && mod.old_file != "" {
			os.Remove(filepath.Join(mod.path, mod.old_file))
			log.Printf("Removed old file %s", mod.old_file)
		}
	}
	return err
}

func (mod ModResource) IndexPair() utils.Pair {
	hash, _ := utils.Hash(mod.Path())
	return utils.Pair{
		Key: mod.filename,
		Value: map[string]string{"id": mod.id, "hash": hash, "version": mod.version},
	}
}

func (mod ModResource) Type() int {
	return 0
}

func (mod *ModResource) SetOldFile(name string) {
	mod.old_file = name
	if strings.HasSuffix(name, ".disabled") && mod.Enabled() {
		mod.filename += ".disabled"
	}
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
				mod.SetOldFile(installed_mod.Filename())
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
		info := make(map[string]string)
		info["hash"] = mod.hash
		info["version"] = mod.version
		results[mod.id] = info
	}

	return results
}
