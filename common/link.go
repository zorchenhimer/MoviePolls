package common

import (
	"fmt"
	"strings"
)

type Link struct {
	Id       int
	IsSource bool
	Type     string
	Url      string
}

func (l Link) String() string {
	return fmt.Sprintf("Link{Id: %v Url: %s Type: %s IsSource: %v}", l.Id, l.Url, l.Type, l.IsSource)
}

func (l Link) DetermineLinkType() (string, error) {
	if strings.Contains(l.Url, "imdb") {
		return "IMDb", nil
	}
	if strings.Contains(l.Url, "myanimelist") {
		return "MyAnimeList", nil
	}

	return "Misc", nil
}
