package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mitchellh/mapstructure"
	mpc "github.com/zorchenhimer/MoviePolls/common"
	mpd "github.com/zorchenhimer/MoviePolls/data"
)

const jsonFilename string = "db/data.json"

func main() {
	err := loadOldDB()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

type OldMovie struct {
	Id             int
	Name           string
	Links          []string
	Description    string
	Remarks        string
	Duration       string
	Rating         float32
	CycleAddedId   int
	CycleWatchedId int
	Removed        bool
	Approved       bool
	Poster         string
	AddedBy        int
	Tags           []int
}

func loadOldDB() error {

	data, err := ioutil.ReadFile(jsonFilename)
	if err != nil {
		fmt.Println(err)
	}

	var fullData map[string]interface{}

	json.Unmarshal(data, &fullData)

	// fullData now contains the whole db
	jMovies := fullData["Movies"].(map[string]interface{})
	oldmovies := []OldMovie{}
	for _, movie := range jMovies {
		om := &OldMovie{}
		mapstructure.Decode(movie, &om)
		oldmovies = append(oldmovies, *om)
	}

	delete(fullData, "Movies")

	data, err = json.MarshalIndent(fullData, "", " ")

	err = ioutil.WriteFile("db/temp.json", data, 666)
	if err != nil {
		return err
	}

	l, err := mpc.NewLogger(mpc.LLDebug, "")

	if err != nil {
		return err
	}

	dc, err := mpd.GetDataConnector("json", "db/temp.json", l)

	if err != nil {
		return err
	}

	for _, oldmovie := range oldmovies {
		// convert old movies to new ones
		newMovie := mpc.Movie{
			Id:          oldmovie.Id,
			Name:        oldmovie.Name,
			Description: oldmovie.Description,
			Remarks:     oldmovie.Remarks,
			Duration:    oldmovie.Duration,
			Rating:      oldmovie.Rating,
			Removed:     oldmovie.Removed,
			Approved:    oldmovie.Approved,
			Poster:      oldmovie.Poster,
		}

		added, err := dc.GetCycle(oldmovie.CycleAddedId)
		if err != nil {
			return err
		}
		newMovie.CycleAdded = added

		if oldmovie.CycleWatchedId != 0 {
			watched, err := dc.GetCycle(oldmovie.CycleWatchedId)
			if err != nil {
				return err
			}
			newMovie.CycleWatched = watched
		}

		user, err := dc.GetUser(oldmovie.AddedBy)
		if err != nil {
			return err
		}
		newMovie.AddedBy = user

		tags := []*mpc.Tag{}
		for _, tagID := range oldmovie.Tags {
			tag := dc.GetTag(tagID)
			tags = append(tags, tag)
		}
		newMovie.Tags = tags

		// convert strings to link structs and add them to the db
		links := []*mpc.Link{}
		for id, linkUrl := range oldmovie.Links {
			link := mpc.Link{
				Url:      linkUrl,
				IsSource: id == 0,
			}
			ltype, err := link.DetermineLinkType()
			if err != nil {
				return err
			}
			link.Type = ltype

			dc.AddLink(&link)
			links = append(links, &link)
		}
		newMovie.Links = links

		dc.AddMovie(&newMovie)
	}

	err = os.Rename("db/temp.json", jsonFilename)
	if err != nil {
		return err
	}

	return nil
}
