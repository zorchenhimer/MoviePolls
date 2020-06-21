package moviepoll

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/zorchenhimer/MoviePolls/common"
)

type dataapi interface {
	getTitle() (string, error)
	getDesc() (string, error)
	getPoster() (string, error) //path to the file  (from root)
	requestResults() error
}

type tmdb struct {
	l     *common.Logger
	id    string
	token string
	resp  *map[string][]map[string]interface{}
}

type jikan struct {
	l            *common.Logger
	id           string
	excludeTypes []string
	resp         *map[string]interface{}
}

func getMovieData(api dataapi) ([]string, error) {
	var slice []string

	err := api.requestResults()

	if err != nil {
		return nil, err
	}

	val, err := api.getTitle()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	val, err = api.getDesc()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	val, err = api.getPoster()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	return slice, nil
}

func (t *tmdb) requestResults() error {
	url := "https://api.themoviedb.org/3/find/" + t.id + "?api_key=" + t.token + "&language=en-US&external_source=imdb_id"
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return errors.New("\n\nTried to access API - Response Code: " + resp.Status + "\nMaybe check your tmdb api token")
	} else {
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return err
		}

		var dat map[string][]map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			return errors.New("Error while unmarshalling json response")
		}

		t.resp = &dat
		return nil
	}
}

func (t *tmdb) getTitle() (string, error) {

	title := ""

	dat := t.resp

	if len((*dat)["movie_results"]) == 0 {
		return "", errors.New("JSON Result did not return a movie, make sure the imdb link is for a movie")
	}
	title = (*dat)["movie_results"][0]["title"].(string)
	release := (*dat)["movie_results"][0]["release_date"].(string)

	release = release[0:4]

	title = title + " (" + release + ")"

	return title, nil
}

func (t *tmdb) getDesc() (string, error) {

	desc := ""

	dat := t.resp

	if len((*dat)["movie_results"]) == 0 {
		return "", errors.New("JSON Result did not return a movie, make sure the imdb link is for a movie")
	}
	desc = (*dat)["movie_results"][0]["overview"].(string)

	return desc, nil
}

func (t *tmdb) getPoster() (string, error) {

	external_path := ""

	dat := t.resp

	if len((*dat)["movie_results"]) == 0 {
		return "", errors.New("JSON Result did not return a movie, make sure the imdb link is for a movie")
	}

	external_path = (*dat)["movie_results"][0]["poster_path"].(string)

	if external_path == "" {
		return "unknown.jpg", nil
	} else {
		fileurl := "https://image.tmdb.org/t/p/w300" + external_path

		path := "posters/" + t.id + ".jpg"

		err := DownloadFile(path, fileurl)

		if !(err == nil) {
			return "unknown.jpg", errors.New("Error while downloading file, using unknown.jpg")
		}
		return path, nil
	}
}

func (j *jikan) requestResults() error {
	resp, err := http.Get("https://api.jikan.moe/v3/anime/" + j.id)
	if err != nil || resp.StatusCode != 200 {

		return errors.New("\n\nTried to access API - Response Code: " + resp.Status + "\n Request URL: " + "https://api.jikan.moe/v3/anime/" + j.id + "\n")
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var dat map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			return errors.New("Error while unmarshalling json response")
		}

		j.resp = &dat
		return nil
	}
}

func (j *jikan) getTitle() (string, error) {

	title := ""

	dat := j.resp

	title = (*dat)["title"].(string)

	if (*dat)["title_english"] != nil && (*dat)["title_english"].(string) != (*dat)["title"].(string) {
		title += " (" + (*dat)["title_english"].(string) + ")"
	}

	return title, nil
}

func (j *jikan) getDesc() (string, error) {

	desc := ""

	dat := j.resp

	desc = (*dat)["synopsis"].(string)

	return desc, nil

}

func (j *jikan) getPoster() (string, error) {

	fileurl := ""

	dat := j.resp

	fileurl = (*dat)["image_url"].(string)

	path := "posters/" + j.id + ".jpg"

	err := DownloadFile(path, fileurl)

	if !(err == nil) {
		return "unknown.jpg", errors.New("Error while downloading file, using unknown.jpg")
	}
	return path, nil
}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
