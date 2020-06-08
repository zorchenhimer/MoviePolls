package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zorchenhimer/MoviePolls"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Turn on debug mode")
	flag.Parse()

	s, err := moviepoll.NewServer(moviepoll.Options{Debug: debug, LogLevel: "debug"})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = s.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
