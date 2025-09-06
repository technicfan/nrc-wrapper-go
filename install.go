package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
)

func get_minecraft_version() (string, error) {
	file, err := os.OpenFile("../mmc-pack.json", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var data PrismInstance
	err = json.Unmarshal(content, &data)
	if err != nil {
		return "", err
	}

	for _, entry := range data.Components {
		if entry.CName == "Minecraft" {
			return entry.Version, nil
		}
	}

	return "", errors.New("minecraft not found")
}

func download_jar_clean(url string, name string, version string, id string, old_file string, wg *sync.WaitGroup, index chan <- map[string]string, limiter chan struct{}) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <- limiter }()

	a, err := download_jar(url, name)
	if err != nil {
		log.Fatal(err)
	}
	if a != old_file && a != "" && old_file != "" {
		os.Remove(fmt.Sprintf("mods/%s", old_file))
	}

	result := make(map[string]string)
	result["id"] = id
	result["hash"], err = calc_hash(fmt.Sprintf("mods/%s", name))
	if err != nil {
		result["hash"] = ""
	}
	result["version"] = version

	index <- result
}

func read_index() []map[string]string {
	file, err := os.Open(".nrc-index.json")
	if err != nil {
		return nil
	}

	byte_data, err := io.ReadAll(file)
	if err != nil {
		return nil
	}
	defer file.Close()

	var data []map[string]string
	err = json.Unmarshal(byte_data, &data)
	if err != nil {
		return nil
	}

	return data
}

func write_index(data []map[string]string) error {
	var file *os.File
	file, err := os.OpenFile(".nrc-index.json", os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(".nrc-index.json")
		if err != nil {
			return err
		}
	}
	defer file.Close()

	json_string, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(json_string))
	if err != nil {
		return err
	}

	return nil
}

func convert_to_index(mods []ModEntry) []map[string]string {
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

func get_installed_versions() (map[string]map[string]string, error) {
    files, err := os.ReadDir("mods")
    if err != nil {
        return nil, err
    }
	index := read_index()

	hashes := make(map[string]map[string]string)
    for _, f := range files {
        if !f.IsDir() && (filepath.Ext(f.Name()) == ".jar" || filepath.Ext(f.Name()) == ".disabled") {
			hash, err := calc_hash(fmt.Sprintf("mods/%s", f.Name()))
			if err == nil {
				info := make(map[string]string)
				info["filename"] = f.Name()
				hashes[hash] = info
			}
        }
    }

	results := make(map[string]map[string]string)
	for _, entry := range index {
		if _, exists := hashes[entry["hash"]]; exists {
			info := make(map[string]string)
			info["version"] = entry["version"]
			info["filename"] = hashes[entry["hash"]]["filename"]
			info["hash"] = entry["hash"]
			results[entry["id"]] = info
		}
	}

	return results, nil
}

func get_compatible_nrc_mods(mc_version string, nrc_mods []NoriskMod) ([]ModEntry, error) {
	var mods []ModEntry
	for _, mod := range nrc_mods {
		if _, exists := mod.Compatibility[mc_version]; exists {
			mods = append(
				mods,
				ModEntry{
					"",
					mod.Compatibility[mc_version]["fabric"]["identifier"],
					mod.Id,
					"",
					"",
					mod.Source["type"],
					mod.Source["repositoryRef"],
					mod.Source["groupId"],
					mod.Source["projectId"],
					mod.Source["artifactId"],
				},
			)
		}
	}

	return mods, nil
}

func remove_installed_mods(mods []ModEntry, installed_mods map[string]map[string]string) ([]ModEntry, []ModEntry) {
	var result []ModEntry
	var removed []ModEntry
	for _, mod := range mods {
		if _, exists := installed_mods[mod.Id]; exists {
			if mod.Version != installed_mods[mod.Id]["version"] {
				mod.OldFile = installed_mods[mod.Id]["filename"]
				result = append(result, mod)
			} else {
				mod.Hash = installed_mods[mod.Id]["hash"]
				removed = append(removed, mod)
			}
		} else {
			result = append(result, mod)
		}
	}

	return result, removed
}

func build_maven_url(mod ModEntry, repos map[string]string) (string, string) {
	group_path := strings.ReplaceAll(mod.GroupId, ".", "/")
	filename := fmt.Sprintf("%s-%s.jar", mod.MavenId, mod.Version)
	mod_path := fmt.Sprintf("%s/%s/%s/%s", group_path, mod.MavenId, mod.Version, filename)

	return repos[mod.RepositoryRef] + mod_path, filename
}

func install(pack string, nrc_mods []NoriskMod, repos map[string]string, wg1 *sync.WaitGroup) error {
	defer wg1.Done()

	mc_version, err := get_minecraft_version()
	if err != nil {
		return err
	}
	mods, err := get_compatible_nrc_mods(mc_version, nrc_mods)
	if err != nil {
		return err
	}
	if len(mods) == 0 {
		log.Fatalf("There are no NRC mods for %s in %s", mc_version, pack)
	}
	installed_mods, err := get_installed_versions()
	if err != nil {
		return err
	}
	mods_to_download, already_installed := remove_installed_mods(mods, installed_mods)

	if len(mods_to_download) == 0 {
		return nil
	}

	log.Println("Installing missing mods")

	modrinth_lookup := make(map[string]ModEntry)
	limiter := make(chan struct{}, 10)
	var modrinth_mods []ModEntry
	var wg sync.WaitGroup

	index := make(chan map[string]string, len(mods_to_download))
	for _, mod := range mods_to_download {
		if mod.SourceType == "modrinth" {
			modrinth_lookup[mod.Id] = mod
			modrinth_lookup[mod.ModrinthId] = mod
			modrinth_mods = append(modrinth_mods, mod)
		} else {
			url, filename := build_maven_url(mod, repos)
			wg.Add(1)
			go download_jar_clean(url, filename, mod.Version, mod.Id, mod.OldFile, &wg, index, limiter)
		}
	}

	results := make(chan []ModrinthMod, len(modrinth_mods))
	for _, mod := range modrinth_mods {
		wg.Add(1)
		go get_modrinth_versions(mod.Id, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	reg := regexp.MustCompile(`(-|,).*$`)

	for modrinth_versions := range results {
		for _, modrinth_mod := range modrinth_versions {
			mod := modrinth_lookup[modrinth_mod.Project_id]
			if slices.Contains(modrinth_mod.Loaders, "fabric") && 
				(slices.Contains(modrinth_mod.Versions, mc_version) || mod.Version == modrinth_mod.Id || mod.Id == "silk") && 
				(reg.ReplaceAllString(mod.Version, "") == modrinth_mod.Version ||
				mod.Version == modrinth_mod.Version || mod.Version == modrinth_mod.Id) {
				for _, file := range modrinth_mod.Files {
					if file.Primary {
						wg.Add(1)
						go download_jar_clean(file.Url, file.Filename, mod.Version, mod.Id, mod.OldFile, &wg, index, limiter)
					}
				}
			}
		}
	}

	go func() {
		wg.Wait()
		close(index)
	}()

	if len(index) > 0 {
		existing_index := convert_to_index(already_installed)
		for entry := range index {
			existing_index = append(existing_index, entry)
		}
		err = write_index(existing_index)
		if err != nil {
			return err
		}
	}

	return nil
}
