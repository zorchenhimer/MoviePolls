package moviepoll

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/zorchenhimer/MoviePolls/common"
)

type dataapi interface {
	getTitle() (string, error)
	getDesc() (string, error)
	getPoster() (string, error) //path to the file  (from root)
	getRunningTime() (string, error)
	getRating() (string, error) //returns the rating as string i.e. 8.69
	requestResults() error
}

type tmdb struct {
	l     *common.Logger
	id    string
	token string
	resp  *map[string]interface{}
}

type jikan struct {
	l             *common.Logger
	id            string
	excludedTypes []string
	resp          *map[string]interface{}
	maxEpisodes   int
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

	val, err = api.getRunningTime()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	val, err = api.getRating()
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
	}
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var tmp map[string][]map[string]interface{}

	if err := json.Unmarshal(body, &tmp); err != nil {
		return errors.New("Error while unmarshalling json response")
	}

	if len(tmp["movie_results"]) == 0 {
		return errors.New("JSON Result did not return a movie, make sure the imdb link is for a movie")
	}

	movieId := tmp["movie_results"][0]["id"]

	url = fmt.Sprintf("https://api.themoviedb.org/3/movie/%v?api_key=%v", movieId, t.token)

	resp, err = http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		return errors.New("\n\nTried to access API - Response Code: " + resp.Status + "\nMaybe check your tmdb api token")
	}

	body, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var dat map[string]interface{}

	if err := json.Unmarshal(body, &dat); err != nil {
		return errors.New("Error while unmarshalling json response")
	}

	t.resp = &dat
	return nil
}

func (t *tmdb) getTitle() (string, error) {

	title := ""

	dat := t.resp

	title = (*dat)["title"].(string)
	release := (*dat)["release_date"].(string)

	release = release[0:4]

	title = title + " (" + release + ")"

	return title, nil
}

func (t *tmdb) getDesc() (string, error) {

	desc := ""

	dat := t.resp

	desc = (*dat)["overview"].(string)

	return desc, nil
}

func (t *tmdb) getPoster() (string, error) {

	external_path := ""

	dat := t.resp

	external_path = (*dat)["poster_path"].(string)

	if external_path == "" {
		return "unknown.jpg", nil
	}

	fileurl := "https://image.tmdb.org/t/p/w300" + external_path

	path := "posters/" + t.id + ".jpg"

	err := DownloadFile(path, fileurl)

	if !(err == nil) {
		return "unknown.jpg", errors.New("Error while downloading file, using unknown.jpg")
	}

	return path, nil
}

func (t *tmdb) getRunningTime() (string, error) {

	runtime := 0

	dat := t.resp

	runtime = int((*dat)["runtime"].(float64))

	return fmt.Sprintf("%v hr %v min", runtime/60, runtime%60), nil
}

func (t *tmdb) getRating() (string, error) {

	dat := t.resp

	rating := (*dat)["vote_average"].(float64)

	return fmt.Sprintf("%v", rating), nil
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

		for _, etype := range j.excludedTypes {
			thisType := dat["type"].(string)
			if strings.ToLower(thisType) == strings.ToLower(etype) {
				return fmt.Errorf("The anime type %s was banned by the sites administrator. Please choose a different type!", thisType)
			}
		}

		if dat["episodes"] != nil {
			episodes := dat["episodes"].(float64)

			if int(episodes) > int(j.maxEpisodes) && int(j.maxEpisodes) != 0 {
				return fmt.Errorf("The anime has too many (%d) episodes. The site administrator only allowed animes up to %d episodes.", int(episodes), int(j.maxEpisodes))
			}
		} else {
			return fmt.Errorf("The episode count of this anime has not been published yet. Therefore this anime can not be added.")
		}

		j.resp = &dat
		return nil
	}
}

func (j *jikan) getTitle() (string, error) {

	title := ""
	dat := j.resp

	if (*dat)["title"] != nil {
		title = (*dat)["title"].(string)
	} else {
		return "", errors.New("No title returned from API")
	}

	if (*dat)["title_english"] != nil && (*dat)["title_english"].(string) != (*dat)["title"].(string) {
		title += " (" + (*dat)["title_english"].(string) + ")"
	}

	return title, nil
}

func (j *jikan) getDesc() (string, error) {
	dat := j.resp

	if (*dat)["synopsis"] != nil {
		return (*dat)["synopsis"].(string), nil
	}

	return "", nil

}

func (j *jikan) getPoster() (string, error) {

	fileurl := ""
	dat := j.resp

	if (*dat)["image_url"] != nil {
		fileurl = (*dat)["image_url"].(string)
	} else {
		return "", nil
	}

	path := "posters/" + j.id + ".jpg"
	err := DownloadFile(path, fileurl)

	if !(err == nil) {
		return "unknown.jpg", errors.New("Error while downloading file, using unknown.jpg")
	}
	return path, nil
}

func (j *jikan) getRunningTime() (string, error) {

	dat := j.resp

	if (*dat)["duration"] != nil {
		return (*dat)["duration"].(string), nil
	}

	return "", nil
}

func (j *jikan) getRating() (string, error) {

	dat := j.resp

	if (*dat)["score"] != nil {
		return fmt.Sprintf("%v", (*dat)["score"].(float64)), nil
	}

	return "", nil
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
