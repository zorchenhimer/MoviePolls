package moviepoll

import "fmt"

type dataapi interface {
	getTitle() string
	getDesc() string
	getPoster() string //path to the file  (from root)
}

type tmdb struct {
	url string
}

type jikan struct {
	url string
}

func (t tmdb) getTitle() string {
	fmt.Printf("getting title for %s", t.url)
	// TODO FILL MEEE !!!!!
	return ""
}

func (t tmdb) getDesc() string {

	// TODO FILL MEEE !!!!!

	return ""
}

func (t tmdb) getPoster() string {

	// TODO FILL MEEE !!!!!

	return ""
}

func (j jikan) getTitle() string {

	// TODO FILL MEEE !!!!!

	return ""
}

func (j jikan) getDesc() string {

	// TODO FILL MEEE !!!!!
	return ""

}

func (j jikan) getPoster() string {

	// TODO FILL MEEE !!!!!

	return ""
}
