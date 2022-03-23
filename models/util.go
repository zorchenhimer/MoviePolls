package models

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/rivo/uniseg"
)

var NotImplementedError error = errors.New("Not implemented")

// fileExists returns whether the given file or directory exists or not.
// Taken from https://stackoverflow.com/a/10510783
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

var re_cleanName = *regexp.MustCompile(`\s+`)

func CleanMovieName(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	return re_cleanName.ReplaceAllString(input, " ")
}

func IntSliceContains(needle int, haystack []int) bool {
	for _, i := range haystack {
		if i == needle {
			return true
		}
	}
	return false
}

// This function filters the given movies by the supplied tags
// To be returned a movie has to match ALL supplied tags
func FilterMoviesByTags(movies []*Movie, tags []string) ([]*Movie, error) {

	// converting the slice to a map to make removing movies easier
	movieMap := make(map[int]*Movie)
	for idx, movie := range movies {
		movieMap[idx] = movie
	}

	for idx, movie := range movieMap {
		ok := true
		for _, tag := range tags {
			if !movieContainsTag(movie, tag) {
				ok = false
			}
		}

		if !ok {
			delete(movieMap, idx)
		}
	}

	// converting the map back to a slice
	found := []*Movie{}
	for _, movie := range movieMap {
		found = append(found, movie)
	}

	return found, nil
}

// checks if a movie contains a certain tag - returns either true or false
func movieContainsTag(movie *Movie, tag string) bool {

	for _, mTag := range movie.Tags {
		if strings.ToLower(tag) == strings.ToLower(mTag.Name) {
			return true
		}
	}
	return false
}

// Returns the length of a string in regards of the "acutal" glypes (i.e. a emoji is counted as
// one character).
func GetStringLength(str string) int {
	return uniseg.GraphemeClusterCount(strings.TrimSpace(str))
}
