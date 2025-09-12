package main

import (
	"fmt"
	"log"
	"os"
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
	if launch {
		log.Println("Loading NoRiskClient...")

		config = get_config()
		mods_dir = config["mods-dir"]
	}

	if check_connection() {
		versions, err := get_norisk_versions()
		if err != nil {
			log.Fatal(err)
		}

		if !launch {
			fmt.Println("Available values for \"NRC_PACK\":")
			for value, pack := range versions.Packs {
				fmt.Printf("- %s (%s)\n", value, pack.Desc)
			}
			return
		}

		pack, exists := versions.Packs[config["nrc-pack"]]
		if !exists {
			log.Fatalf("%s is not a valid NRC pack", config["nrc-pack"])
		}
		var mods []NoriskMod
		var assets []string
		for _, inherited_pack := range pack.Inherits {
			mods = append(mods, versions.Packs[inherited_pack].Mods...)
			for _, asset_pack := range versions.Packs[inherited_pack].Assets {
				assets = append(assets, asset_pack)
			}
		}
		assets = append(assets, pack.Assets...)

		var wg sync.WaitGroup
		token_out := make(chan string, 1)
		wg.Add(3)

		go get_token(config, &wg, token_out)
		go load_assets(assets, &wg)
		go install(config, pack.Mods, mods, versions.Repositories, &wg)

		wg.Wait()

		token = <- token_out
	} else {
		log.Println("No connection to the API")
		if !launch { return }
		log.Println("Launching without doing anything")
		token = "offline"
	}

    command := os.Args[1]
    args := append(
		[]string{
			command, fmt.Sprintf("-Dnorisk.token=%s", token),
			fmt.Sprintf("-Dfabric.addMods=%s", mods_dir),
		}, os.Args[2:]...
	)

	err := Exec(command, args)
	if err != nil {
		log.Fatal(err)
	}
}
