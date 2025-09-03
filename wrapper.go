package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func main(){
	os.Mkdir("mods", os.ModePerm)

	token, err := get_token()
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

	args := []string{strings.TrimSpace(os.Args[1]), fmt.Sprintf("-Dnorisk.token=%s", token)}
	args = append(args, os.Args[2:]...)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
