package moviepoll

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"math/big"
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

func intSliceContains(needle int, haystack []int) bool {
	for _, i := range haystack {
		if i == needle {
			return true
		}
	}
	return false
}

func getCryptRandKey(size int) string {
	out := ""
	large := big.NewInt(int64(1 << 60))
	large = large.Add(large, large)
	for len(out) < size {
		num, err := rand.Int(rand.Reader, large)
		if err != nil {
			panic("Error generating session key: " + err.Error())
		}
		out = fmt.Sprintf("%s%X", out, num)
	}

	if len(out) > size {
		out = out[:size]
	}
	return out
}

func (s *Server) hashPassword(pass string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(s.passwordSalt+pass)))
}
