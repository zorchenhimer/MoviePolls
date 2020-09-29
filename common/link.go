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

func (l *Link) ValidateLink() error {
	url := l.Url
	if !strings.Contains(url, "//") {
		l.Url = "https://" + url
	}
	return nil
}

func (l *Link) DetermineLinkType() error {
	if strings.Contains(l.Url, "imdb") {
		l.Type = "IMDb"
		return nil
	}
	if strings.Contains(l.Url, "myanimelist") {
		l.Type = "MyAnimeList"
		return nil
	}

	l.Type = "Misc"
	return nil
}
