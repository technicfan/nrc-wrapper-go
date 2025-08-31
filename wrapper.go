package main

import (
	"log"
	"os"
)

func main(){
	logger := log.New(os.Stdout, "owo: ", log.LstdFlags)
	logger.Print("test")
}
