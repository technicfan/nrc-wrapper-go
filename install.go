package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"

	"strings"
	"sync"
)

func download_jar_clean(
	url string,
	alt_url string,
	name string,
	version string,
	id string,
	old_file string,
	path string,
	error_on_fail bool,
	check_hash bool,
	wg *sync.WaitGroup,
	index chan<- map[string]string,
	limiter chan struct{},
) {
	defer wg.Done()

	limiter <- struct{}{}
	defer func() { <-limiter }()

	if strings.HasSuffix(old_file, ".disabled") {
		name = name + ".disabled"
	}
	a, err := download_jar(url, name, path, check_hash)
	if alt_url != "" && err != nil && err.Error() == "HTTP 404" {
		a, err = download_jar(alt_url, name, path, check_hash)
	}
	if err != nil {
		if error_on_fail {
			log.Fatalf("Failed to download %s: %s", name, err.Error())
		}
		log.Printf("Failed to download %s: %s", name, err.Error())
	}
	if a != old_file && a != "" && old_file != "" {
		os.Remove(filepath.Join(path, old_file))
		log.Printf("Removed old file %s", old_file)
	}

	result := make(map[string]string)
	result["id"] = id
	result["hash"], err = calc_hash(filepath.Join(path, name))
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

func write_index(
	data []map[string]string,
) error {
	var file *os.File
	file, err := os.OpenFile(".nrc-index.json", os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		file, err = os.Create(".nrc-index.json")
		if err != nil {
			return err
		}
	}
	defer file.Close()

	json_string, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(json_string))
	if err != nil {
		return err
	}

	return nil
}

func convert_to_index(
	mods []ModEntry,
) []map[string]string {
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

func get_installed_mods(
	path string,
) (map[string]map[string]string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	index := read_index()

	hashes := make(map[string]map[string]string)
	for _, f := range files {
		if !f.IsDir() &&
			(filepath.Ext(f.Name()) == ".jar" || filepath.Ext(f.Name()) == ".disabled") {
			hash, err := calc_hash(filepath.Join(path, f.Name()))
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

func get_compatible_nrc_mods(
	mc_version string,
	loader string,
	nrc_mods []NoriskMod,
) ([]ModEntry, error) {
	var mods []ModEntry
	for _, mod := range nrc_mods {
		if _, exists := mod.Compatibility[mc_version]; exists {
			var filename string
			if mod.Compatibility[mc_version][loader]["source"] != nil {
				source := mod.Compatibility[mc_version][loader]["source"].(map[string]any)
				for k, v := range source {
					mod.Source[k] = v.(string)
				}
			}
			if mod.Compatibility[mc_version][loader]["filename"] != nil {
				filename = mod.Compatibility[mc_version][loader]["filename"].(string)
			}
			mods = append(
				mods,
				ModEntry{
					"",
					mod.Compatibility[mc_version][loader]["identifier"].(string),
					mod.Id,
					filename,
					"",
					mod.Source["type"],
					mod.Source["repositoryRef"],
					mod.Source["groupId"],
					mod.Source["projectId"],
					mod.Source["projectSlug"],
					mod.Source["artifactId"],
				},
			)
		}
	}

	return mods, nil
}

func get_missing_mods_clean(
	mods []ModEntry,
	installed_mods map[string]map[string]string,
	path string,
) ([]ModEntry, []ModEntry) {
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
			delete(installed_mods, mod.Id)
		} else {
			result = append(result, mod)
		}
	}

	for _, file := range installed_mods {
		os.Remove(filepath.Join(path, file["filename"]))
		log.Printf("Removed left over file %s", file["filename"])
	}

	return result, removed
}

func build_maven_url(
	mod ModEntry,
	repos map[string]string,
) (string, string, string) {
	var url, alt_url string
	if mod.SourceType == "modrinth" {
		version := mod.Version
		if !strings.Contains(mod.Version, "-") {
			version = strings.Replace(mod.Version, ",", "-", 1)
		}
		filename := fmt.Sprintf("%s-%s.jar", mod.ModrinthId, version)
		url = fmt.Sprintf(
			"%smaven/modrinth/%s/%s/%s",
			repos[mod.SourceType], mod.ModrinthId, version, filename,
		)
		alt_url = strings.ReplaceAll(url, mod.ModrinthId, mod.ProjectSlug)
	} else {
		group_path := strings.ReplaceAll(mod.GroupId, ".", "/")
		filename := fmt.Sprintf("%s-%s.jar", mod.MavenId, mod.Version)
		url = fmt.Sprintf(
			"%s%s/%s/%s/%s",
			repos[mod.RepositoryRef], group_path, mod.MavenId, mod.Version, filename,
		)
	}

	return url, alt_url, fmt.Sprintf("%s-%s.jar", mod.Id, mod.Version)
}

func install(
	config map[string]string,
	nrc_mods_main []NoriskMod,
	nrc_mods_inherited []NoriskMod,
	repos map[string]string,
	wg1 *sync.WaitGroup,
) {
	defer wg1.Done()
	os.Mkdir(config["mods-dir"], os.ModePerm)

	mods, err := get_compatible_nrc_mods(config["mc-version"], config["loader"], nrc_mods_main)
	if err != nil {
		log.Fatalf("Failed to get nrc mods: %s", err.Error())
	}
	if len(mods) == 0 {
		log.Fatalf("There are no NRC mods for %s in %s", config["mc-version"], config["nrc-pack"])
	}
	inherited_mods, err := get_compatible_nrc_mods(
		config["mc-version"],
		config["loader"],
		nrc_mods_inherited,
	)
	if err != nil {
		log.Fatalf("Failed to get nrc mods: %s", err.Error())
	}
	var ids []string
	for _, mod := range mods {
		ids = append(ids, mod.Id)
	}
	for _, mod := range inherited_mods {
		if !slices.Contains(ids, mod.Id) {
			mods = append(mods, mod)
		}
	}
	installed_mods, err := get_installed_mods(config["mods-dir"])
	if err != nil {
		log.Fatalf("Failed to get installed mods: %s", err.Error())
	}
	mods_to_download, already_installed := get_missing_mods_clean(
		mods,
		installed_mods,
		config["mods-dir"],
	)

	if len(mods_to_download) == 0 {
		return
	}

	log.Println("Installing missing mods")

	limiter := make(chan struct{}, 10)
	var wg sync.WaitGroup

	index := make(chan map[string]string, len(mods_to_download))
	for _, mod := range mods_to_download {
		var url, alt_url, filename string
		if mod.SourceType == "url" {
			url = mod.Version
			filename = mod.Filename
		} else {
			url, alt_url, filename = build_maven_url(mod, repos)
		}
		wg.Add(1)
		go download_jar_clean(
			url,
			alt_url,
			filename,
			mod.Version,
			mod.Id,
			mod.OldFile,
			config["mods-dir"],
			config["error-on-failed-download"] == "",
			mod.SourceType != "url",
			&wg,
			index,
			limiter,
		)

	}

	wg.Wait()
	close(index)

	if len(index) > 0 {
		existing_index := convert_to_index(already_installed)
		for entry := range index {
			existing_index = append(existing_index, entry)
		}
		err = write_index(existing_index)
		if err != nil {
			log.Fatalf("Failed to write mod metadata: %s", err.Error())
		}
	}
}
