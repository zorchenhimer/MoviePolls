package main

import (
	"fmt"
	"os"
	"time"

	mp "github.com/zorchenhimer/MoviePolls"
)

func main() {
	err := os.MkdirAll("db", 0777)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := writeData(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := printData(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const jsonFilename string = "db/data.json"

func printData() error {
	jc, err := mp.LoadJson(jsonFilename)
	if err != nil {
		return err
	}

	movies := jc.GetActiveMovies()
	if len(movies) == 0 {
		return fmt.Errorf("No active movies")
	}

	for _, m := range movies {
		fmt.Println(m)
	}

	return nil
}

func writeData() error {
	dur, err := time.ParseDuration("-168h")
	if err != nil {
		return err
	}

	cycles := []*mp.Cycle{
		&mp.Cycle{
			Id: 1,
			Start: time.Now().Add(dur),
			End: time.Now(),
			EndingSet: false,
		},
	}

	movies := []*mp.Movie{
		&mp.Movie{
			Id: 1,
			Name: "Austin Powers",
			Links: []string{"http://localhost:8080/"},
			Description: "The first Austin Powers movie.  Idk.",
			CycleAdded: cycles[0],
			Removed: false,
			Approved: true,
		},
		&mp.Movie{
			Id: 2,
			Name: "Rubber",
			Links: []string{"http://localhost:8080/"},
			Description: "A movie about a tire.",
			CycleAdded: cycles[0],
			Removed: false,
			Approved: true,
		},
	}

	users := []*mp.User{
		&mp.User{
			Id: 1,
			Name: "Zorchenhimer",
			OAuthToken: "123abc",
			Email: "zorchenhimer@gmail.com",
			NotifyCycleEnd: true,
			NotifyVoteSelection: true,
			Privilege: mp.PRIV_ADMIN,
		},
		&mp.User{
			Id: 2,
			Name: "SleepyMia",
			OAuthToken: "abc123",
			Email: "SleepyMia@gmail.com",
			NotifyCycleEnd: true,
			NotifyVoteSelection: true,
			Privilege: mp.PRIV_MOD,
		},
		&mp.User{
			Id: 3,
			Name: "jojoa1997",
			OAuthToken: "",
			Email: "",
			NotifyCycleEnd: false,
			NotifyVoteSelection: false,
			Privilege: mp.PRIV_USER,
		},
	}

	type vdata struct {
		User int
		Movie int
		Cycle int
	}
	votes := []vdata{
		vdata{1, 1, 1},
		vdata{2, 1, 1},
		vdata{1, 2, 1},
	}

	jc := mp.NewJsonConnector()

	for _, c := range cycles {
		jc.AddCycle(c)
	}

	for _, u := range users {
		if err := jc.AddUser(u); err != nil {
			return fmt.Errorf("Unable to add user %s: %v\n", err)
		}
	}

	for _, m := range movies {
		if err := jc.AddMovie(m); err != nil {
			return fmt.Errorf("Unable to add user %s: %v\n", err)
		}
	}

	for _, v := range votes {
		if err := jc.AddVote(v.User, v.Movie, v.Cycle); err != nil {
			return fmt.Errorf("Unable to add vote %s: %v\n", err)
		}
	}

	err = jc.Save(jsonFilename)
	if err != nil {
		return err
	}

	//fmt.Printf("Current cycle: %d\n", jc.CurrentCycle)
	return nil
}
