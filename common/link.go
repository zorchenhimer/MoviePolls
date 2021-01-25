package common

import (
	"fmt"
	"regexp"
	"strings"
)

type Link struct {
	Id       int
	IsSource bool
	Type     string
	Url      string
}

func NewLink(link string, id int) (*Link, error) {
	var source bool
	if id == 0 {
		source = true
	} else {
		source = false
	}

	ls := Link{
		Url:      link,
		IsSource: source,
	}

	err := ls.validateLink()
	if err != nil {
		return nil, err
	}

	err = ls.determineLinkType()
	if err != nil {
		return nil, err
	}

	return &ls, nil
}

func (l Link) String() string {
	return fmt.Sprintf("Link{Id: %v Url: %s Type: %s IsSource: %v}", l.Id, l.Url, l.Type, l.IsSource)
}

var re_validLink = *regexp.MustCompile(`[a-zA-Z0-9:._\+]{1,256}\.[a-zA-Z0-9()]{1,6}[a-zA-Z0-9%_:\+.\/]*`)

func (l *Link) validateLink() error {
	url := l.Url
	if re_validLink.MatchString(url) {
		url = stripRefFromLink(url)
		url = stripMobileFromLink(url)

		if len(url) <= 8 {
			// lets be stupid when the link is too short
			if !strings.ContainsAny(url, "//") {
				url = "https://" + url
			}
		} else {
			// lets be smart when the link is long enough
			// url[:8] ensures that we only look at the 8 first characters of the url -> where the protocol part should be
			if !strings.Contains(url[:8], "//") {
				url = "https://" + url
			}
		}
		l.Url = url
		return nil
	}
	return fmt.Errorf("Invalid link: %v", l.Url)
}

func stripRefFromLink(link string) string {
	idx := strings.Index(link, "/?")
	if idx != -1 {
		return link[:idx]
	}
	return link
}

// for now we just replace it for imdb, if we need others we need to adapt that
// is it stupid, YES, does it work, i guess? do i have a better idea, no
func stripMobileFromLink(link string) string {
	return strings.Replace(link, "m.imdb.com", "imdb.com", 1)
}

func (l *Link) determineLinkType() error {
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
