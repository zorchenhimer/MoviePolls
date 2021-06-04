package logic

import (
	"fmt"
	"net/http"
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

	id := -1
	if autofill {
		id = b.doAutofill(links, user)
	} else {
		id = b.doFormfill(validatedForm, user)
	}

	///////////////////////////////////////////////////

	if autofill {
		b.l.Debug("autofill")
		results, links, err := b.handleAutofill(input)

		if results == nil || links == nil {
			fields["Autofill"] = InputField{Error: fmt.Errorf("Could not autofill all fields")}
		} else {
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
			// Prepare a int for the id
			var movieId int
		}

	} else {
	}

	return 0, fields, fmt.Errorf("// TODO")
}

func (b *backend) doAutofill(links []*models.Link) (int, error) {

	sourcelink := links[0]

	if sourcelink.Type == "MyAnimeList" {
		b.l.Debug("MAL link")

		results, err = b.handleJikan(data, sourcelink.Url)

		if err != nil {
			s.l.Error(err.Error())
			return nil, err
		}

		var title string

		if len(results) != 6 {
			s.l.Error("Jikan API results have an unexpected length, expected 6 got %v", len(results))
			data.ErrorMessage = append(data.ErrorMessage, "API autofill did not return enough data, contact the server administrator")
			return nil, nil
		} else {
			title = results[0]
		}

		exists, err := s.data.CheckMovieExists(title)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(
				http.StatusInternalServerError,
				"something went wrong :C",
				w, r)
			return nil, nil
		}

		if exists {
			s.l.Debug("Movie already exists")
			data.ErrorMessage = append(data.ErrorMessage, "Movie already exists in database")
			data.ErrAutofill = true
			return nil, nil
		}

		results = append(results, remarkstext)
		return results, links

	}
	if sourcelink.Type == "IMDb" {
		s.l.Debug("IMDB link")

		results, err = s.handleTmdb(data, w, r, sourcelink.Url)

		if err != nil {
			s.l.Error(err.Error())
			return nil, nil
		}

		var title string

		if len(results) != 6 {
			s.l.Error("Tmdb API results have an unexpected length, expected 6 got %v", len(results))
			data.ErrorMessage = append(data.ErrorMessage, "API autofill did not return enough data, did you input a link to a series?")
			return nil, nil
		} else {
			title = results[0]
		}

		exists, err := s.data.CheckMovieExists(title)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(
				http.StatusInternalServerError,
				"something went wrong :C",
				w, r)
			return nil, nil

		}

		if exists {
			s.l.Debug("Movie already exists")
			data.ErrorMessage = append(data.ErrorMessage, "Movie already exists in database")
			data.ErrAutofill = true
			return nil, nil
		}

		results = append(results, remarkstext)
		return results, links

	}

	s.l.Debug("no link")
	data.ErrorMessage = append(data.ErrorMessage, "To use autofill an imdb or myanimelist link as first link is required")
	data.ErrLinks = true
	return nil, nil

}
