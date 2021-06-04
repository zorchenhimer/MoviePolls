package web

import (
	"net/http"

	"github.com/zorchenhimer/MoviePolls/models"
	"github.com/zorchenhimer/MoviePolls/logic"
)

func (s *webServer) pageAddMovie(w http.ResponseWriter, r *http.Request) {

	// Get the user which adds a movie
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	// Get the current cycle to see if we can add a movie
	currentCycle, err := s.backend.GetCurrentCycle()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get current cycle: %v", err)
		return
	}

	if currentCycle == nil {
		s.doError(
			http.StatusInternalServerError,
			"No cycle active!",
			w, r)
		return
	}

	formfillEnabled, err := s.backend.GetFormFillEnabled()
	//formfillEnabled, err := s.data.GetCfgBool(ConfigFormfillEnabled, DefaultFormfillEnabled)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

			s.l.Error("Unable to get config value %s: %v", ConfigFormfillEnabled, err)
		return
	}

	maxRemLen, err := s.backend.GetMaxRemarksLength()
	//maxRemLen, err := s.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", ConfigMaxRemarksLength, err)
		return
	}

	data := struct{
		dataPageBase

		// eg, "Title": InputField{}
		Fields map[string]logic.InputField

		AutofillEnabled bool
		FormfillEnabled bool

		MaxRemarksLength int

		HasErrors bool
	}{
		dataPageBase:     s.newPageBase("Add Movie", w, r),
		FormfillEnabled:  formfillEnabled,
		MaxRemarksLength: maxRemLen,
		ErrorFields:      map[string]bool
	}

	// .ErrorFields.Title

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4096)
		if err != nil {
			s.l.Error("Error parsing movie form: %v", err)
		}

		input := make(map[string]&InputField{})
		for key, slice := range r.MultipartForm {
			if len(slice) == 0 {
				continue
			}
			input[key] = &InputField{Value: slice[0]}
		}

		// if err is not nil, fields is not nil
		movieId, fields, err = s.backend.AddMovie(input, )
		if fields == nil && err == nil {
			http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
			return
		}

		if r.FormValue("AutofillBox") == "on" {
			// do autofill

				movieId, err = s.data.AddMovie(movie)
				if err != nil {
					data.ErrTitle = true // For now we enable the title flag
					data.ErrorMessage = append(data.ErrorMessage, "Could not add movie, contact your server administrator")
					s.l.Error("Movie could not be added. Error: %v", err)
				} else {
					http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
				}

			}
		} else if formfillEnabled {
			s.l.Debug("formfill")
			// do formfill
			results, links := s.handleFormfill(&data, w, r)

			if results == nil || links == nil {
				data.ErrorMessage = append(data.ErrorMessage, "One or more fields reported an error.")
			} else {
				// Fill all the fields in the movie struct
				movie.Name = results[0]
				movie.Description = results[1]
				movie.Poster = filepath.Base(results[2])
				movie.Remarks = results[3]
				movie.Links = links
				movie.AddedBy = user

				// Prepare a int for the id
				var movieId int

				for _, link := range movie.Links {
					id, err := s.data.AddLink(link)
					if err != nil {
						s.l.Debug("link error: %v", err)
					}
					link.Id = id
				}

				movieId, err = s.data.AddMovie(movie)
				if err != nil {
					data.ErrTitle = true // For now we enable the title flag
					data.ErrorMessage = append(data.ErrorMessage, "Could not add movie, contact your server administrator")
					s.l.Error("Movie could not be added. Error: %v", err)
				} else {
					http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
				}
			}
		}
	}
	if err := s.executeTemplate(w, "addmovie", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
