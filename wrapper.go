package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"os/exec"

	"golang.org/x/sys/unix"
)

func main(){
	if len(os.Args) < 3 {
		log.Fatal("you need to use it as the wrapper command")
		os.Exit(1)
	}

	os.Mkdir("mods", os.ModePerm)

	config := get_config()

	token, err := get_token(config["prism_data"])
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
    token_arg := fmt.Sprintf("-Dnorisk.token=%s", token)
    args := append([]string{command, token_arg}, os.Args[2:]...)

	if runtime.GOOS == "windows" {
		cmd := exec.Command(command, args[1:]...)
		cmd.Stdin, cmd.Stderr, cmd.Stdout = os.Stdin, os.Stderr, os.Stdout
		err = cmd.Run()
	} else {
		err = unix.Exec(command, args, os.Environ())
	}
	if err != nil {
		log.Fatal(err)
	}
}
