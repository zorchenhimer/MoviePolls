package logic

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/zorchenhimer/MoviePolls/models"
)

var re_tagSearch = regexp.MustCompile(`t:"([a-zA-Z ]+)"`)

func (b *backend) SearchMovieTitles(query string) ([]*models.Movie, error) {
	// finding tags
	tags := re_tagSearch.FindAllString(query, -1)

	// clean up the tags from the "tagsyntax"
	tagsToFind := []string{}
	for _, tag := range tags {
		tagsToFind = append(tagsToFind, tag[3:len(tag)-1])
	}

	query = re_tagSearch.ReplaceAllString(query, "")
	query = strings.Trim(query, " ")

	// we first seach for matching titles (ignoring the tags for now)
	movieList, err := b.data.SearchMovieTitles(query)
	if err != nil {
		return nil, err
	}

	// NOW we filter the already found movies by the tags provided
	return models.FilterMoviesByTags(movieList, tagsToFind)
}

func (b *backend) GetActiveMovies() ([]*models.Movie, error) {
	return b.data.GetActiveMovies()
}

func (b *backend) GetMovie(id int) *models.Movie {
	m, err := b.data.GetMovie(id)
	if err != nil {
		b.l.Error("Error getting movie with ID %d: %v", id, err)
		return nil
	}
	return m
}

func (b *backend) validateForm(fields map[string]*InputField) (map[string]*InputField, bool, []*models.Link) {

	ret, links := b.parseLinks(fields["Links"])

	if ret.Error != nil {
		return fields, false, nil
	}

	fields["Links"] = ret

	// Check Remarks max length
	maxRemarksLength, err := b.GetMaxRemarksLength()
	if err != nil {
		b.l.Debug("%v", err.Error)
		fields["Remarks"].Error = fmt.Errorf("Something went wrong :C")
		return fields, false, nil
	}

	length := models.GetStringLength(fields["Remarks"].Value)
	if length > maxRemarksLength {
		b.l.Debug("Remarks too long: %d", length)
		fields["Remarks"].Error = fmt.Errorf("Remarks too long! Max Length: %d characters", maxRemarksLength)
		return fields, false, nil
	}

	autofill := false
	autofillField, ok := fields["AutofillBox"]
	if ok && autofillField.Value == "on" {
		autofill = true
	}

	if !autofill {
		maxTitleLength, err := b.GetMaxTitleLength()
		if err != nil {
			b.l.Debug("%v", err.Error)
			fields["Title"].Error = fmt.Errorf("Something went wrong :C")
			return fields, autofill, nil
		}

		title, ok := fields["Title"]
		length = models.GetStringLength(title.Value)
		if !ok || length == 0 {
			b.l.Debug("Title empty")
			fields["Title"].Error = fmt.Errorf("A title is required when not using autofill!")
		}

		if length > maxTitleLength {
			b.l.Debug("Title too long: %d", length)
			fields["Title"].Error = fmt.Errorf("Title too long! Max Length: %d characters", maxTitleLength)
		}

		movieExists, err := b.CheckMovieExists(title.Value)
		if err != nil {
			b.l.Debug("%v", err.Error)
			fields["Title"].Error = fmt.Errorf("Something went wrong :C")
			return fields, autofill, nil
		}

		if movieExists {
			b.l.Debug("Movie exists")
			fields["Title"].Error = fmt.Errorf("Movie already added to the poll or has been already watched")
		}

		maxDescriptionLength, err := b.GetMaxDescriptionLength()
		if err != nil {
			b.l.Debug("%v", err.Error)
			fields["Description"].Error = fmt.Errorf("Something went wrong :C")
			return fields, autofill, nil
		}

		description, ok := fields["Description"]
		length = models.GetStringLength(description.Value)

		if length > maxDescriptionLength {
			b.l.Debug("Description too long: %d", length)
			fields["Description"].Error = fmt.Errorf("Description too long! Max Length: %d characters", maxDescriptionLength)
		}
	}
	return fields, autofill, links
}

// Welcome to the ugliest error handling ever - I have to surround the actual errors with the `InputField`
// struct to send it back to the frontend. To check for a error from this function one has to check the
// first return parameters .Error field for not nil. If it is NOT nil, the error is there and the second
// return parameter is to be expected nil / only partially filled.
func (b *backend) parseLinks(linkField *InputField) (*InputField, []*models.Link) {

	linktext := strings.ReplaceAll(linkField.Value, "\r", "")

	// Check for links
	if len(linktext) == 0 {
		b.l.Error("no links given")
		linkField.Error = fmt.Errorf("No link found")
		return linkField, nil
	}

	links := strings.Split(linktext, "\n")

	linkList := []*models.Link{}

	maxLinkLength, err := b.GetMaxLinkLength()
	if err != nil {
		b.l.Error("Unable to get %q: %w", ConfigMaxLinkLength, err)
		linkField.Error = fmt.Errorf("something went wrong :C")
		return linkField, nil
	}

	for id, linkString := range links {
		// Check link maxlength
		length := models.GetStringLength(linkString)
		if length > maxLinkLength {
			b.l.Debug("Link too long: %d", length)
			linkField.Error = fmt.Errorf("A Link is too long! Max Length: %d characters, found a link with %d characters.", maxLinkLength, length)
			return linkField, nil
		}
		link, err := models.NewLink(linkString, id)
		if err != nil {
			linkField.Error = err
			return linkField, nil
		}
		linkList = append(linkList, link)
	}

	return linkField, linkList
}

// `fields` contains the key: {value,error} pairs from the input form on the 'addMovie' page. (i think?)
func (b *backend) AddMovie(fields map[string]*InputField, user *models.User) (int, map[string]*InputField) {

	validatedForm, autofill, links := b.validateForm(fields)

	// Exit early if any errors got reported
	for _, input := range validatedForm {
		if input.Error != nil {
			return 0, validatedForm
		}
	}

	remarks := validatedForm["Remarks"].Value

	id := -1
	err := fmt.Errorf("")
	if autofill {
		id, err = b.doAutofill(links, user, remarks)
		if err != nil {
			//handle
		}
	} else {
		// id = b.doFormfill(validatedForm, user)
	}

	return id, validatedForm

}

func (b *backend) doAutofill(links []*models.Link, user *models.User, remarks string) (int, error) {

	sourcelink := links[0]

	results := []string{}

	if sourcelink.Type == "MyAnimeList" {
		b.l.Debug("MAL link")

		results, err := b.autofillJikan(sourcelink.Url)

		if err != nil {
			b.l.Error(err.Error())
			return -1, err
		}

		var title string

		if len(results) != 6 {
			b.l.Error("Jikan API results have an unexpected length, expected 6 got %v", len(results))
			return -1, fmt.Errorf("API autofill did not return enough data, contact the server administrator")
		} else {
			title = results[0]
		}

		exists, err := b.CheckMovieExists(title)
		if err != nil {
			b.l.Error(err.Error())
			return -1, fmt.Errorf("Something went wrong :C")
		}

		if exists {
			b.l.Debug("Movie already exists")
			return -1, fmt.Errorf("Movie already exists in database")
		}

		results = append(results)
	} else if sourcelink.Type == "IMDb" {
		b.l.Debug("IMDB link")

		results, err := b.autofillTmdb(sourcelink.Url)

		if err != nil {
			b.l.Error(err.Error())
			return -1, err
		}

		var title string

		if len(results) != 6 {
			b.l.Error("Tmdb API results have an unexpected length, expected 6 got %v", len(results))
			return -1, fmt.Errorf("API autofill did not return enough data, did you input a link to a series?")
		} else {
			title = results[0]
		}

		exists, err := b.CheckMovieExists(title)
		if err != nil {
			b.l.Error(err.Error())
			return -1, fmt.Errorf("Something went wrong :C")
		}

		if exists {
			b.l.Debug("Movie already exists")
			return -1, fmt.Errorf("Movie already exists in database")
		}

		results = append(results)
	} else {
		b.l.Debug("no link")
		return -1, fmt.Errorf("To use autofill an imdb or myanimelist link as first link is required")
	}

	movie := models.Movie{}

	// Fill all the fields in the movie struct
	movie.Name = results[0]
	movie.Description = results[1]
	movie.Poster = filepath.Base(results[2])
	movie.Duration = results[3]

	rating, err := strconv.ParseFloat(results[4], 32)
	if err != nil {
		b.l.Error("[AddMovie] Error converting string to float")
		movie.Rating = 0.0
	} else {
		movie.Rating = float32(rating)
	}

	movie.Remarks = results[6]

	for _, link := range links {
		id, err := b.data.AddLink(link)
		if err != nil {
			b.l.Debug("[AddMovie] link error: %v", err)
		}
		link.Id = id
	}

	movie.Links = links
	movie.AddedBy = user

	tags := []*models.Tag{}
	for _, tagStr := range strings.Split(results[5], ",") {
		tag := &models.Tag{
			Name: tagStr,
		}

		id, err := b.data.AddTag(tag)
		if err != nil {
			b.l.Debug("[AddMovie] duplicate tag: %v", tagStr)
		}
		tag.Id = id

		tags = append(tags, tag)
	}

	movie.Tags = tags

	return b.AddMovieToDB(&movie)

}

var re_jikanToken = regexp.MustCompile(`[^\/]*\/anime\/([0-9]+)`)

func (b *backend) autofillJikan(sourcelink string) ([]string, error) {

	jikanEnabled, err := b.GetJikanEnabled()
	if err != nil {
		b.l.Debug(err.Error())
		return nil, fmt.Errorf("Something went wrong :C")
	}

	if !jikanEnabled {
		return nil, fmt.Errorf("Jikan API usage was not enabled by the site administrator")
	}

	// Get Data from MAL (jikan api)
	match := re_jikanToken.FindStringSubmatch(sourcelink)
	var id string
	if len(match) < 2 {
		b.l.Debug("Regex match didn't find the anime id in %v", sourcelink)
		return nil, fmt.Errorf("Could not retrive anime id from provided link, did you input a manga link?")
	}
	id = match[1]

	bannedTypes, err := b.GetJikanBannedTypes()

	if err != nil {
		b.l.Debug("Error while retriving config value 'JikanBannedTypes':\n %v", err)
		return nil, fmt.Errorf("Something went wrong :C")
	}

	maxEpisodes, err := b.GetJikanMaxEpisodes()

	if err != nil {
		b.l.Debug("Error while retriving config value 'JikanMaxEpisodes':\n %v", err)
		return nil, fmt.Errorf("Something went wrong :C")
	}

	maxDuration, err := b.GetMaxDuration()

	if err != nil {
		b.l.Debug("Error while retriving config value 'MaxMultEpLength':\n %v", err)
		return nil, fmt.Errorf("Something went wrong :C")
	}

	sourceAPI := jikan{id: id, l: b.l, excludedTypes: bannedTypes, maxEpisodes: maxEpisodes, maxDuration: maxDuration}

	// Request data from API
	results, err := getMovieData(&sourceAPI)

	if err != nil {
		b.l.Debug("Error while accessing Jikan API: %v", err)
		return nil, fmt.Errorf("Could not complete autofill, contact your site administrator")
	}

	return results, nil
}

var re_tmdbToken = regexp.MustCompile(`[^\/]*\/title\/(tt[0-9]*)`)

func (b *backend) autofillTmdb(sourcelink string) ([]string, error) {

	tmdbEnabled, err := b.GetTmdbEnabled()
	if err != nil {
		b.l.Error(err.Error())
		return nil, fmt.Errorf("Something went wrong :C")
	}

	if !tmdbEnabled {
		b.l.Debug("Aborting Tmdb autofill since it is not enabled")
		return nil, fmt.Errorf("Tmdb API usage was not enabled by the site administrator")
	}

	// Retrieve token from database
	token, err := b.GetTmdbToken()
	if err != nil || token == "" {
		b.l.Debug("Aborting Tmdb autofill since no token was found, its either empty or was never set")
		return nil, fmt.Errorf("The Tmdb integration is not configured correctly, contact the site administrator")
	}
	// get the movie id
	match := re_tmdbToken.FindStringSubmatch(sourcelink)
	var id string
	if len(match) < 2 {
		b.l.Debug("Regex match didn't find the movie id in %v", sourcelink)
		return nil, fmt.Errorf("Could not retrive movie information from the first provided link")
	}

	id = match[1]

	sourceAPI := tmdb{id: id, token: token, l: b.l}

	// Request data from API
	results, err := getMovieData(&sourceAPI)

	if err != nil {
		b.l.Debug("Error while accessing Tmdb API: %v", err)
		return nil, fmt.Errorf("Could not complete autofill, contact your site administrator")
	}

	return results, nil
}
