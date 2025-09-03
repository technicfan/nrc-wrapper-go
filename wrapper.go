package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

func main(){
	os.Mkdir("mods", 0600)

	token, err := get_token()
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	load_assets(token, &wg)
	wg.Add(1)
	install(&wg)
	wg.Wait()

	args := []string{fmt.Sprintf("-Dnorisk.token=%s", token)}
	args = append(args, os.Args[3:]...)

	cmd := exec.Command(os.Args[2], args...)
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}
