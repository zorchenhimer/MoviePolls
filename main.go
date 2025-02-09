package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/logger"
	"github.com/zorchenhimer/MoviePolls/logic"
	"github.com/zorchenhimer/MoviePolls/web"
)

var ReleaseVersion string

func main() {
	var logFile string
	var logLevel string
	var addr string
	var debug bool
	var version bool

	flag.StringVar(&addr, "addr", ":8090", "Server address")
	flag.StringVar(&logFile, "logfile", "logs/server.log", "File to write logs")
	flag.StringVar(&logLevel, "loglevel", "debug", "Log verbosity")
	flag.BoolVar(&debug, "debug", false, "Enable debug code")
	flag.BoolVar(&version, "version", true, "Show the version of the binary file")
	flag.Parse()

	log, err := logger.NewLogger(logger.LogLevel(logLevel), logFile)
	if err != nil {
		fmt.Printf("Unable to load logger: %v\n", err)
		os.Exit(1)
	}

	config := web.Options{
		Debug:  debug,
		Listen: addr,
	}

	if version {
		log.Info("Running version: %s", ReleaseVersion)
	}
	if config.Debug {
		log.Info("Debug mode turned on")
	}

	// init database
	data, err := database.GetDatabase("json", "db/data.json", log)
	if err != nil {
		fmt.Printf("Unable to load json data: %v\n", err)
		os.Exit(1)
	}

	// init logic
	backend, err := logic.New(data, log)
	if err != nil {
		fmt.Printf("Unable to load backend: %v\n", err)
		os.Exit(1)
	}

	// init frontend
	frontend, err := web.New(config, backend, log)
	if err != nil {
		fmt.Printf("Unable to load frontend: %v\n", err)
		os.Exit(1)
	}

	// run frontend
	err = frontend.ListenAndServe()
	if err != http.ErrServerClosed {
		fmt.Printf("Error serving: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("goodbye")
}
