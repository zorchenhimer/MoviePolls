package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zorchenhimer/MoviePolls"
	"github.com/zorchenhimer/MoviePolls/common"
)

func main() {
	var logFile string
	var logLevel string
	var debug bool
	flag.StringVar(&logFile, "logfile", "", "File to write logs")
	flag.StringVar(&logLevel, "loglevel", "debug", "Log verbosity")
	flag.BoolVar(&debug, "debug", false, "Enable debug code")
	flag.Parse()

	s, err := moviepoll.NewServer(moviepoll.Options{Debug: debug, LogLevel: common.LogLevel(logLevel), LogFile: logFile})
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
