package logic

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/nfnt/resize"
	"github.com/zorchenhimer/MoviePolls/logger"
)

type dataapi interface {
	getTitle() (string, error)
	getDesc() (string, error)
	getPoster() (string, error) //path to the file  (from root)
	getDuration() (string, error)
	getRating() (string, error) //returns the rating as string i.e. 8.69
	getTags() (string, error)   //returns the tags as comma seperated string
	requestResults() error
}

type tmdb struct {
	l     *logger.Logger
	id    string
	token string
	resp  *map[string]interface{}
}

type jikan struct {
	l             *logger.Logger
	id            string
	excludedTypes []string
	resp          *map[string]interface{}
	maxEpisodes   int
	maxDuration   int
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

	val, err = api.getDuration()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	val, err = api.getRating()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	val, err = api.getTags()
	if err != nil {
		return nil, err
	}
	slice = append(slice, val)

	return slice, nil
}

func (t *tmdb) requestResults() error {
	url := fmt.Sprintf("https://api.themoviedb.org/3/find/%v?api_key=%v&language=en-US&external_source=imdb_id", t.id, t.token)
	resp, err := http.Get(url)

	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Tried to access API - Response Code: %v\nMaybe check your tmdb api token", resp.Status)
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
		return fmt.Errorf("Tried to access API - Response Code: %v\nMaybe check your tmdb api token", resp.Status)
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

	fileurl := "https://image.tmdb.org/t/p/original" + external_path

	path := "posters/" + t.id + ".jpg"

	err := DownloadFile(path, fileurl)

	if err != nil {
		return "unknown.jpg", errors.New("Error while downloading file, using unknown.jpg")
	}

	return path, nil
}

func (t *tmdb) getDuration() (string, error) {

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

func (t *tmdb) getTags() (string, error) {

	dat := t.resp

	tagMaps := (*dat)["genres"].([]interface{})

	tags := []string{}
	tags = append(tags, "IMDB")

	for _, tag := range tagMaps {
		tg := tag.(map[string]interface{})
		tags = append(tags, tg["name"].(string))
	}

	// Sadly the Decision was made to pass all results as a slice of strings
	// Therefore this has to be a string instead of a slice of string, otherwise it cannot be passed
	// to the caller
	return strings.Join(tags, ","), nil
}

var re_duration = regexp.MustCompile(`([0-9]{1,3}) min`)

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

		if dat["duration"] != nil && dat["duration"] != "Unknown" && j.maxDuration >= 0 {
			match := re_duration.FindStringSubmatch(dat["duration"].(string))
			if len(match) < 2 {
				j.l.Error("Could not detect episode duration.")
				return fmt.Errorf("The episode duration of this anime has not been published or has an unexpected format. Therefore this anime can not be added.")
			}
			duration, err := strconv.Atoi(match[1])

			if err != nil {
				j.l.Error("Could not convert duration %v to int", match[1])
				return fmt.Errorf("The episode duration of this anime has not been published or has an unexpected format. Therefore this anime can not be added.")
			}

			// this looks stupid but it works lul
			if (duration * int(dat["episodes"].(float64))) > j.maxDuration {
				j.l.Error("Duration of the anime %s is too long: %d", dat["title"].(string), (duration * int(dat["episodes"].(float64))))
				return fmt.Errorf("The duration of this series (episode duration * episodes) is longer than the maximum duration defined by the admin. Therefore this anime can not be added.")
			}
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
		return "posters/unknown.jpg", nil
	}

	path := "posters/" + j.id + ".jpg"
	err := DownloadFile(path, fileurl)

	if !(err == nil) {
		return "posters/unknown.jpg", errors.New("Error while downloading file, using unknown.jpg")
	}
	j.l.Debug("poster path: %s", path)
	return path, nil
}

func (j *jikan) getDuration() (string, error) {

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

func (j *jikan) getTags() (string, error) {

	dat := j.resp

	tagMaps := (*dat)["genres"].([]interface{})

	tags := []string{}
	tags = append(tags, "MAL")

	for _, tag := range tagMaps {
		tg := tag.(map[string]interface{})
		tags = append(tags, tg["name"].(string))
	}

	// Sadly the Decision was made to pass all results as a slice of strings
	// Therefore this has to be a string instead of a slice of string, otherwise it cannot be passed
	// to the caller
	return strings.Join(tags, ","), nil
}

func DownloadFile(filepath string, url string) error {

	// Download image data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode image data
	image, err := jpeg.Decode(resp.Body)

	// Resize the raw image
	resized := resize.Resize(200, 0, image, resize.NearestNeighbor)

	// Reencode the image data
	err = jpeg.Encode(file, resized, nil)
	if err != nil {
		return err
	}

	return nil
}
