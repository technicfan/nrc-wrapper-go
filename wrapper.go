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
		os.Exit(1)
	}

	log.Print("Loading NoRiskClient...")

	os.Mkdir("mods", os.ModePerm)

	config := get_config()

	versions, err := get_norisk_versions()
	if err != nil {
		log.Fatal(err)
	}

	pack := versions.Packs[config["nrc-pack"]]

	var wg sync.WaitGroup
	token_out := make(chan string, 1)
	wg.Add(3)

	go get_token(config["prism_dir"], &wg, token_out)
	go load_assets(pack.Assets, &wg)
	go install(pack, versions.Repositories, &wg)

	wg.Wait()

	token := <- token_out
    command := os.Args[1]
    args := append([]string{command, fmt.Sprintf("-Dnorisk.token=%s", token)}, os.Args[2:]...)

	err = Exec(command, args)
	if err != nil {
		log.Fatal(err)
	}
}
