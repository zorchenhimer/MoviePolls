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

func (b *backend) parseFields(fields map[string]*InputField) ([]*models.Link, error) {
	return nil, nil
}

func (b *backend) AddMovie(fields map[string]*InputField, user *models.User) (int, map[string]InputField, error) {

	links, err := b.parseFields(fields)

	if parsed.Autofill {
		b.doAutofill(links)
	} else {
		b.doFormfill()
	}

	// TODO: make sure this actually works, lmao
	input := &inputForm{*form}

	autofill := false
	val, err := input.GetValue("AutofillBox")
	//val, ok := form.Value["AutofillBox"]
	if val == "on" || err != nil {
		autofill = true
	}

	// Get all needed values from the form
	output := map[string]InputField{}
	inputErr := false

	// Get all links from the corresponding input field
	linktext, err := input.GetValue("Links")
	if err != nil {
		b.l.Error("[handleAutofill] Links: %w", err)
	}
	linktext = strings.ReplaceAll(linktext, "\r", "")
	links := InputField{Value: linktext}

	// Get the remarks from the corresponding input field
	remarkstext, err := input.GetValue("Remarks")
	if err != nil {
		b.l.Error("[handleAutofill] Remarks: %w", err)
	}
	remarkstext = strings.ReplaceAll(remarkstext, "\r", "")
	output["Remarks"] = InputField{Value: remarkstext}

	// Check link maxlength
	maxLinkLength, err := b.GetMaxLinkLength()
	if err != nil {
		b.l.Error("Unable to get %q: %w", ConfigMaxLinkLength, err)
		return nil, nil, fmt.Errorf("something went wrong :C")
	}

	if models.GetStringLength(linktext) > maxLinkLength {
		b.l.Debug("Links too long: %d", models.GetStringLength(linktext))
		links.Error = fmt.Errorf("Links too long! Max Length: %d characters", maxLinkLength)
		inputErr = true
	}

	// Check for links
	if len(linktext) == 0 {
		b.l.Error("no links given")
		links.Error = fmt.Errorf("No link found")
		inputErr = true
	}

	linkstrings := strings.Split(linktext, "\n")
	var sourcelink *models.Link

	linkErrors := []string{}
	// Convert links to structs
	linkList := []*models.Link{}
	for id, link := range linkstrings {

		ls, err := models.NewLink(link, id)
		if err != nil {
			b.l.Error("Cannot add link: %w", err)
			linkErrors = append(linkErrors, fmt.Sprintf("Could not add link: %v.", err.Error()))
			inputErr = true
		}

		if ls.IsSource {
			sourcelink = ls
		}

		linkList = append(linkList, ls)
	}
	if len(linkErrors) != 0 {
		// FIXME: format this better
		links.Error = fmt.Errorf("%s", strings.Join(linkErrors, " "))
	}

	output["Links"] = links
	// FIXME: left off here

	// Check Remarks max length
	maxRemarksLength, err := b.GetMaxRemarksLength()
	if err != nil {
		return nil, output, err
	}

	if models.GetStringLength(remarkstext) > maxRemarksLength {
		s.l.Debug("Remarks too long: %d", models.GetStringLength(remarkstext))
		output["Remarks"] = InputField{Value: remarkstext, Error: fmt.Errorf("Remarks too long! Max Length: %d characters", maxRemarksLength)}
		inputErr = true
	}

	// Exit early if any errors got reported
	if data.isError() {
		return nil, nil
	}

	///////////////////////////////////////////////////

	jovie := &models.Movie{}
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

// outsourced autofill logic
func (b *backend) handleAutofill(input *inputForm) (map[string]InputField, []*models.Link, error) {

	if sourcelink.Type == "MyAnimeList" {
		s.l.Debug("MAL link")

		results, err = s.handleJikan(data, w, r, sourcelink.Url)

		if err != nil {
			s.l.Error(err.Error())
			return nil, nil
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
