package common

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
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

var re_validLink = *regexp.MustCompile(`[a-zA-Z0-9:._\+]{1,256}\.[a-zA-Z0-9()]{1,6}[a-zA-Z0-9%_:\+.\/]*`)

// verifyLinks checks each link for valid syntax and trims spaces.
func VerifyLinks(links []string) ([]string, error) {
	l := []string{}
	for _, link := range links {
		fmt.Printf(">> %q\n", link)

		// verify url with regex (ripped and adapted from SO)
		// [a-zA-Z0-9:._\+]{1,256}\.[a-zA-Z0-9()]{1,6}[a-zA-Z0-9%_:\+.\/]*
		//   ^ foobar 				 .com 				:123

		if re_validLink.Match([]byte(link)) {
			l = append(l, link)
		}
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("No valid links provided")
	} else {
		return l, nil
	}
}

var re_cleanNameA = *regexp.MustCompile(`[^a-zA-Z0-9 ]`)
var re_cleanNameB = *regexp.MustCompile(`\s+`)

func CleanMovieName(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = re_cleanNameA.ReplaceAllString(input, "")
	return re_cleanNameB.ReplaceAllString(input, " ")
}

func IntSliceContains(needle int, haystack []int) bool {
	for _, i := range haystack {
		if i == needle {
			return true
		}
	}
	return false
}
