package moviepoll

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

type dataapi interface {
	getTitle() string
	getDesc() string
	getPoster() string //path to the file  (from root)
}

type tmdb struct {
	id    string
	token string
}

type jikan struct {
	id string
}

func getMovieData(api dataapi) []string {
	var slice []string
	slice = append(slice, api.getTitle())
	slice = append(slice, api.getDesc())
	slice = append(slice, api.getPoster())

	fmt.Printf("DEBUG: %v\n", slice)

	return slice
}

func (t tmdb) getTitle() string {

	title := ""

	resp, err := http.Get("https://api.themoviedb.org/3/find/" + t.id +
		"?api_key=" + t.token + "&language=en-US&external_source=imdb_id")
	if err != nil || resp.StatusCode != 200 {
		panic("\n\nTried to access API - Response Code: " + resp.Status + "\nRequest Url: " + "https://api.themoviedb.org/3/find/" + t.id + "?api_key=" + t.token + "&language=en-US&external_source=imdb_id\n\n")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		var dat map[string][]map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			panic("unmarshal tmdb")
		}

		if len(dat["movie_results"]) == 0 {
			return ""
		}
		title = dat["movie_results"][0]["title"].(string)
		release := dat["movie_results"][0]["release_date"].(string)

		release = release[0:4]

		title = title + " (" + release + ")"
	}

	return title
}

func (t tmdb) getDesc() string {

	desc := ""

	resp, err := http.Get("https://api.themoviedb.org/3/find/" + t.id +
		"?api_key=" + t.token + "&language=en-US&external_source=imdb_id")
	if err != nil || resp.StatusCode != 200 {
		panic("\n\nTried to access API - Response Code: " + resp.Status + "\nRequest Url: " + "https://api.themoviedb.org/3/find/" + t.id + "?api_key=" + t.token + "&language=en-US&external_source=imdb_id\n\n")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		var dat map[string][]map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			panic("unmarshal tmdb")
		}
		if len(dat["movie_results"]) == 0 {
			return ""
		}
		desc = dat["movie_results"][0]["overview"].(string)
	}

	return desc
}

func (t tmdb) getPoster() string {

	external_path := ""

	resp, err := http.Get("https://api.themoviedb.org/3/find/" + t.id +
		"?api_key=" + t.token + "&language=en-US&external_source=imdb_id")
	if err != nil || resp.StatusCode != 200 {
		panic("\n\nTried to access API - Response Code: " + resp.Status + "\nRequest Url: " + "https://api.themoviedb.org/3/find/" + t.id + "?api_key=" + t.token + "&language=en-US&external_source=imdb_id\n\n")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		var dat map[string][]map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			panic("unmarshal tmdb")
		}
		if len(dat["movie_results"]) == 0 {
			return ""
		}
		external_path = dat["movie_results"][0]["poster_path"].(string)
	}

	if external_path == "" {
		return "unknown.jpg"
	} else {
		fileurl := "https://image.tmdb.org/t/p/w300" + external_path

		path := "posters/" + t.id + ".jpg"

		err = DownloadFile(path, fileurl)

		if !(err == nil) {
			panic("AAAAAAAAAA")
		}
		return path
	}
}

func (j jikan) getTitle() string {

	title := ""

	resp, err := http.Get("https://api.jikan.moe/v3/anime/" + j.id)
	if err != nil || resp.StatusCode != 200 {
		panic("\n\nTried to access API - Response Code: " + resp.Status + "\n Request URL: " + "https://api.jikan.moe/v3/anime/" + j.id + "\n\n")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		var dat map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			panic("unmarshal jikan")
		}

		title = dat["title"].(string)

		if dat["title_english"] != nil {
			title += dat["title_english"].(string)
		}
	}

	return title
}

func (j jikan) getDesc() string {

	desc := ""

	resp, err := http.Get("https://api.jikan.moe/v3/anime/" + j.id)
	if err != nil || resp.StatusCode != 200 {
		panic("\n\nTried to access API - Response Code: " + resp.Status + "\n Request URL: " + "https://api.jikan.moe/v3/anime/" + j.id + "\n\n")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		var dat map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			panic("unmarshal jikan")
		}

		desc = dat["synopsis"].(string)
	}

	return desc

}

func (j jikan) getPoster() string {

	fileurl := ""

	// get the file url form the api
	resp, err := http.Get("https://api.jikan.moe/v3/anime/" + j.id)
	if err != nil || resp.StatusCode != 200 {
		panic("\n\nTried to access API - Response Code: " + resp.Status + "\n Request URL: " + "https://api.jikan.moe/v3/anime/" + j.id + "\n\n")
	} else {
		body, _ := ioutil.ReadAll(resp.Body)

		var dat map[string]interface{}

		if err := json.Unmarshal(body, &dat); err != nil {
			panic(err)
		}

		fileurl = dat["image_url"].(string)
	}

	path := "posters/" + j.id + ".jpg"

	err = DownloadFile(path, fileurl)

	if !(err == nil) {
		panic("AAAAAAAAAA")
	}
	return path
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
