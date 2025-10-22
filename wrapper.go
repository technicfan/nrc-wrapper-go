package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

func main(){
	launch := true
	if len(os.Args) == 2 && os.Args[1] == "--packs" {
		launch = false
	} else if len(os.Args) < 3 {
		log.Fatal("You need to use it as the wrapper command")
	}

	var token string
	var mods_dir string
	var config map[string]string
	var wg sync.WaitGroup
	token_out := make(chan string, 1)
	if launch {
		log.Println("Loading NoRiskClient...")

		config = get_config()
		mods_dir = config["mods-dir"]
	}

	versions, err := get_norisk_versions()
	if err == nil {
		if !launch {
			print_packs(versions.Packs)
			return
		}

		pack, exists := versions.Packs[config["nrc-pack"]]
		if !exists {
			log.Fatalf("%s is not a valid NRC pack", config["nrc-pack"])
		}
		mods, assets, loaders := get_pack_data(pack, versions.Packs)

		if len(loaders) > 0 {
			if version, exists := loaders[config["loader"]]; exists {
				if config["loader-version"] < version {
					log.Fatalf("Please update %s to version %s", config["loader"], version)
				}
			} else {
				var loaders []string
				for loader, version := range pack.Loader["default"] {
					loaders = append(loaders, fmt.Sprintf("%s %s", loader, version))
				}
				log.Fatalf(
					"%s requires one of the following modloaders: %s",
					config["nrc-pack"],
					strings.Join(loaders, ", "),
				)
			}
		}

		wg.Add(3)

		go get_token(config, false, &wg, token_out)
		go load_assets(assets, config["error-on-failed-download"] == "", &wg)
		go install(config, pack.Mods, mods, versions.Repositories, &wg)

		wg.Wait()

		token = <- token_out
	} else {
		log.Println("No connection to the API")
		if !launch { return }
		wg.Add(1)
		log.Println("Launching without doing anything")
		go get_token(config, true, &wg, token_out)
		wg.Wait()
		token = <- token_out
	}

    command := os.Args[1]
    args := append(
		[]string{
			command, fmt.Sprintf("-Dnorisk.token=%s", token),
			fmt.Sprintf("-Dfabric.addMods=%s", mods_dir),
		}, os.Args[2:]...
	)

	err = Exec(command, args)
	if err != nil {
		log.Fatalf("Command failed with: %s", err.Error())
	}
}
