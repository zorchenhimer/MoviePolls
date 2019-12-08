package moviepoll

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var NotImplementedError error = errors.New("Not implemented")

// fileExists returns whether the given file or directory exists or not.
// Taken from https://stackoverflow.com/a/10510783
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// verifyLinks checks each link for valid syntax and trims spaces.
func verifyLinks(links []string) ([]string, error) {
	l := []string{}
	for _, link := range links {
		fmt.Printf(">> %q\n", link)
		u, err := url.ParseRequestURI(strings.TrimSpace(link))
		if err != nil {
			return nil, err
		}

		l = append(l, u.String())
	}
	return l, nil
}

var re_cleanNameA = *regexp.MustCompile(`[^a-zA-Z0-9 ]`)
var re_cleanNameB = *regexp.MustCompile(`\s+`)

func cleanMovieName(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = re_cleanNameA.ReplaceAllString(input, "")
	return re_cleanNameB.ReplaceAllString(input, " ")
}
