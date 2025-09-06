package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

func main(){
	if len(os.Args) < 3 {
		log.Fatal("You need to use it as the wrapper command")
	}

	var token string
	log.Print("Loading NoRiskClient...")

	if check_connection() {
		os.Mkdir("mods", os.ModePerm)

		config := get_config()

		versions, err := get_norisk_versions()
		if err != nil {
			log.Fatal(err)
		}

		pack, exists := versions.Packs[config["nrc-pack"]]
		if !exists {
			log.Fatalf("%s is not a valid NRC pack", config["nrc-pack"])
		}

		var wg sync.WaitGroup
		token_out := make(chan string, 1)
		wg.Add(3)

		go get_token(config["prism_dir"], &wg, token_out)
		go load_assets(pack.Assets, &wg)
		go install(pack, versions.Repositories, &wg)

		wg.Wait()

		token = <- token_out
	} else {
		log.Print("No connection to the API")
		log.Print("Launching without doing anything")
		token = "offline"
	}

    command := os.Args[1]
    args := append([]string{command, fmt.Sprintf("-Dnorisk.token=%s", token)}, os.Args[2:]...)

	err := Exec(command, args)
	if err != nil {
		log.Fatal(err)
	}
}
