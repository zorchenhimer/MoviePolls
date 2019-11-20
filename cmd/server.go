package main

import (
	"fmt"
	"os"

	"github.com/zorchenhimer/MoviePolls"
)

func main() {
	s, err := moviepoll.NewServer(moviepoll.Options{Debug: true})
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
