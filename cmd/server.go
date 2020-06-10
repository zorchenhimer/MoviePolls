package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zorchenhimer/MoviePolls"
)

func main() {
	var logFile string
	var debug bool
	flag.StringVar(&logFile, "log", "", "File to write logs")
	flag.BoolVar(&debug, "debug", false, "Enable debug code")
	flag.Parse()

	s, err := moviepoll.NewServer(moviepoll.Options{Debug: debug, LogLevel: "debug", LogFile: logFile})
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
