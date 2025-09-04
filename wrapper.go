package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

func main(){
	if len(os.Args) < 3 {
		log.Fatal("you need to use it as the wrapper command")
		os.Exit(1)
	}

	os.Mkdir("mods", os.ModePerm)

	config := get_config()

	token, err := get_token(config["prism_dir"])
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	err = load_assets(token, &wg)
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(1)
	err = install(&wg)
	if err != nil {
		log.Fatal(err)
	}
	wg.Wait()

    command := os.Args[1]
    args := append([]string{command, fmt.Sprintf("-Dnorisk.token=%s", token)}, os.Args[2:]...)

	log.Print("starting minecraft")

	err = Exec(command, args)
	if err != nil {
		log.Fatal(err)
	}
}
