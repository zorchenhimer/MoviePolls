package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/logic"
	"github.com/zorchenhimer/MoviePolls/models"
	"github.com/zorchenhimer/MoviePolls/web"
)

var ReleaseVersion string

func main() {
	log, err := models.NewLogger(models.LLInfo, "logs/server.log")
	if err != nil {
		fmt.Printf("Unable to load logger: %v\n", err)
		os.Exit(1)
	}

	config := web.Options{
		Debug:  true,
		Listen: ":8090",
	}

	log.Info("Running version: %s", ReleaseVersion)
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
