package server

func (s *Server) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	if common.FileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handlerStatic(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimLeft(filepath.Clean("/"+r.URL.Path), "/\\")
	if s.debug {
		s.l.Info("Attempting to serve file %q", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerPoster(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimLeft(filepath.Clean("/"+r.URL.Path), "/\\")
	if s.debug {
		s.l.Info("Attempting to serve file %q", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerAddMovie(w http.ResponseWriter, r *http.Request) {

	// Get the user which adds a movie
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	// Get the current cycle to see if we can add a movie
	currentCycle, err := s.data.GetCurrentCycle()
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

	formfillEnabled, err := s.data.GetCfgBool(ConfigFormfillEnabled, DefaultFormfillEnabled)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", ConfigFormfillEnabled, err)
		return
	}

	maxRemLen, err := s.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", ConfigMaxRemarksLength, err)
		return
	}

	data := dataAddMovie{
		dataPageBase:     s.newPageBase("Add Movie", w, r),
		FormfillEnabled:  formfillEnabled,
		MaxRemarksLength: maxRemLen,
	}

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4096)
		if err != nil {
			s.l.Error("Error parsing movie form: %v", err)
		}

		movie := &common.Movie{}

		if r.FormValue("AutofillBox") == "on" {
			// do autofill
			s.l.Debug("autofill")
			results, links := s.handleAutofill(&data, w, r)

			if results == nil || links == nil {
				data.ErrorMessage = append(data.ErrorMessage, "Could not autofill all fields")
				data.ErrAutofill = true
			} else {
				// Fill all the fields in the movie struct
				movie.Name = results[0]
				movie.Description = results[1]
				movie.Poster = filepath.Base(results[2])
				movie.Duration = results[3]

				rating, err := strconv.ParseFloat(results[4], 32)
				if err != nil {
					s.l.Error("Error converting string to float for adding a movie")
					movie.Rating = 0.0
				} else {
					movie.Rating = float32(rating)
				}

				movie.Remarks = results[6]

				for _, link := range links {
					id, err := s.data.AddLink(link)
					if err != nil {
						s.l.Debug("link error: %v", err)
					}
					link.Id = id
				}

				movie.Links = links
				movie.AddedBy = user

				tags := []*common.Tag{}
				for _, tagStr := range strings.Split(results[5], ",") {
					tag := &common.Tag{
						Name: tagStr,
					}

					id, err := s.data.AddTag(tag)
					if err != nil {
						s.l.Debug("duplicate tag: %v", tagStr)
					}
					tag.Id = id

					tags = append(tags, tag)
				}

				movie.Tags = tags
				// Prepare a int for the id
				var movieId int

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

var re_tagSearch = `t:"([a-zA-Z ]+)"`

func (s *Server) handlerRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	movieList := []*common.Movie{}

	data := struct {
		dataPageBase
		Movies         []*common.Movie
		VotingEnabled  bool
		AvailableVotes int
		LastCycle      *common.Cycle
		Cycle          *common.Cycle
	}{
		dataPageBase: s.newPageBase("Current Cycle", w, r),
	}

	if r.Body != http.NoBody {
		err := r.ParseForm()
		if err != nil {
			s.l.Error(err.Error())
		}
		searchVal := r.FormValue("search")

		// finding tags
		re := regexp.MustCompile(re_tagSearch)
		tags := re.FindAllString(searchVal, -1)

		// clean up the tags from the "tagsyntax"
		tagsToFind := []string{}
		for _, tag := range tags {
			tagsToFind = append(tagsToFind, tag[3:len(tag)-1])
		}

		searchVal = re.ReplaceAllString(searchVal, "")
		searchVal = strings.Trim(searchVal, " ")

		// we first seach for matching titles (ignoring the tags for now)
		movieList, err = s.data.SearchMovieTitles(searchVal)

		// NOW we filter the already found movies by the tags provided
		movieList, err = common.FilterMoviesByTags(movieList, tagsToFind)

		if err != nil {
			s.l.Error(err.Error())
		}
	} else {
		var err error = nil
		movieList, err = s.data.GetActiveMovies()
		if err != nil {
			s.l.Error(err.Error())
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get active movies. Please contact the server admin."),
				w, r)
			return
		}
	}

	if data.User != nil {
		unlimitedVotes, err := s.data.GetCfgBool(ConfigUnlimitedVotes, DefaultUnlimitedVotes)
		if err != nil {
			s.l.Error("Error getting UnlimitedVotes config setting: %v", err)
		}

		data.AvailableVotes = 1
		if !unlimitedVotes {
			maxVotes, err := s.data.GetCfgInt(ConfigMaxUserVotes, DefaultMaxUserVotes)
			if err != nil {
				s.l.Error("Error getting MaxUserVotes config setting: %v", err)
				maxVotes = DefaultMaxUserVotes
			}

			active, _, err := s.getUserVotes(data.User)
			if err != nil {
				s.doError(
					http.StatusBadRequest,
					fmt.Sprintf("Cannot get user votes :C"),
					w, r)
				s.l.Error("Unable to get votes for user %d: %v", data.User.Id, err)
				return
			}
			data.AvailableVotes = maxVotes - len(active)
		}
	}

	data.Movies = common.SortMoviesByVotes(movieList)
	data.VotingEnabled, _ = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)

	cycles, err := s.data.GetPastCycles(0, 1)
	if err != nil {
		s.l.Error("Error getting PastCycle: %v", err)
	}
	if cycles != nil {
		if len(cycles) != 0 {
			data.LastCycle = cycles[0]
		}
	}

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.l.Error("Error getting Current Cycle: %v", err)
	}
	if cycle != nil {
		data.Cycle = cycle
	}

	if err := s.executeTemplate(w, "cyclevotes", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerMovie(w http.ResponseWriter, r *http.Request) {
	var movieId int
	var command string
	n, err := fmt.Sscanf(r.URL.String(), "/movie/%d/%s", &movieId, &command)
	if err != nil && n == 0 {
		dataError := dataMovieError{
			dataPageBase: s.newPageBase("Error", w, r),
			ErrorMessage: "Missing movie ID",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			s.l.Error(err.Error())
		}
		return
	}

	movie, err := s.data.GetMovie(movieId)
	if err != nil {
		dataError := dataMovieError{
			dataPageBase: s.newPageBase("Error", w, r),
			ErrorMessage: "Movie not found",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			s.l.Error("movie not found: " + err.Error())
		}
		return
	}

	data := struct {
		dataPageBase
		Movie          *common.Movie
		VotingEnabled  bool
		AvailableVotes int
	}{
		dataPageBase: s.newPageBase(movie.Name, w, r),
		Movie:        movie,
	}

	data.VotingEnabled, _ = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)
	// FIXME: This is copied from handleCycle.  Put this in a business layer instead.
	if data.User != nil {
		maxVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
		if err != nil {
			s.l.Error("Error getting MaxUserVotes config setting: %v", err)
			maxVotes = DefaultMaxUserVotes
		}

		active, _, err := s.getUserVotes(data.User)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get user votes :C"),
				w, r)
			s.l.Error("Unable to get votes for user %d: %v", data.User.Id, err)
			return
		}
		data.AvailableVotes = maxVotes - len(active)
	}

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}

// outsourced autofill logic
func (s *Server) handleAutofill(data *dataAddMovie, w http.ResponseWriter, r *http.Request) (results []string, links []*common.Link) {

	// Get all needed values from the form

	// Get all links from the corresponding input field
	linktext := strings.ReplaceAll(r.FormValue("Links"), "\r", "")
	data.ValLinks = linktext

	// Get the remarks from the corresponding input field
	remarkstext := strings.ReplaceAll(r.FormValue("Remarks"), "\r", "")
	data.ValRemarks = remarkstext

	// Check link maxlength
	maxLinkLength, err := s.data.GetCfgInt(ConfigMaxLinkLength, DefaultMaxLinkLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxLinkLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	if common.GetStringLength(linktext) > maxLinkLength {
		s.l.Debug("Links too long: %d", common.GetStringLength(linktext))
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Links too long! Max Length: %d characters", maxLinkLength))
		data.ErrLinks = true
	}

	// Check for links
	linkstrings := strings.Split(linktext, "\n")
	if len(linkstrings) == 0 {
		s.l.Error("no links given")
		data.ErrorMessage = append(data.ErrorMessage, "No link found.")
		data.ErrLinks = true
	}

	var sourcelink *common.Link

	// Convert links to structs
	for id, link := range linkstrings {

		ls, err := common.NewLink(link, id)

		if err != nil {
			s.l.Error("Cannot add link")
			data.ErrorMessage = append(data.ErrorMessage, "Could not add link: %v", err.Error())
			data.ErrLinks = true
			return
		}

		if ls.IsSource {
			sourcelink = ls
		}

		links = append(links, ls)
	}

	// Check Remarks max length
	maxRemarksLength, err := s.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxRemarksLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	if common.GetStringLength(remarkstext) > maxRemarksLength {
		s.l.Debug("Remarks too long: %d", common.GetStringLength(remarkstext))
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Remarks too long! Max Length: %d characters", maxRemarksLength))
		data.ErrRemarks = true
	}

	// Exit early if any errors got reported
	if data.isError() {
		return nil, nil
	}

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

// List of past cycles
func (s *Server) handlerHistory(w http.ResponseWriter, r *http.Request) {
	past, err := s.data.GetPastCycles(0, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("something went wrong :C"),
			w, r)
		s.l.Error("Unable to get past cycles: ", err)
		return
	}

	data := struct {
		dataPageBase
		Cycles []*common.Cycle
	}{
		dataPageBase: s.newPageBase("Cycle History", w, r),
		Cycles:       past,
	}

	if err := s.executeTemplate(w, "history", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

var re_jikanToken = regexp.MustCompile(`[^\/]*\/anime\/([0-9]+)`)

func (s *Server) handleJikan(data *dataAddMovie, w http.ResponseWriter, r *http.Request, sourcelink string) ([]string, error) {

	jikanEnabled, err := s.data.GetCfgBool("JikanEnabled", DefaultJikanEnabled)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return nil, fmt.Errorf("Error while retriving config value 'JikanEnabled':\n %v", err)
	}

	s.l.Debug("jikanEnabled: %v", jikanEnabled)

	if !jikanEnabled {
		data.ErrorMessage = append(data.ErrorMessage, "Jikan API usage was not enabled by the site administrator")
		return nil, fmt.Errorf("Jikan not enabled")
	}

	// Get Data from MAL (jikan api)
	match := re_jikanToken.FindStringSubmatch(sourcelink)
	var id string
	if len(match) < 2 {
		s.l.Debug("Regex match didn't find the anime id in %v", sourcelink)
		data.ErrorMessage = append(data.ErrorMessage, "Could not retrive anime id from provided link, did you input a manga link?")
		data.ErrLinks = true
		return nil, fmt.Errorf("Could not retrive anime id from link")
	}
	id = match[1]

	bannedTypesString, err := s.data.GetCfgString(ConfigJikanBannedTypes, DefaultJikanBannedTypes)

	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return nil, fmt.Errorf("Error while retriving config value 'JikanBannedTypes':\n %v", err)
	}

	bannedTypes := strings.Split(bannedTypesString, ",")

	maxEpisodes, err := s.data.GetCfgInt(ConfigJikanMaxEpisodes, DefaultJikanMaxEpisodes)

	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return nil, fmt.Errorf("Error while retriving config value 'JikanMaxEpisodes':\n %v", err)
	}

	maxDuration, err := s.data.GetCfgInt(ConfigMaxMultEpLength, DefaultMaxMultEpLength)

	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return nil, fmt.Errorf("Error while retriving config value 'MaxMultEpLength':\n %v", err)
	}

	sourceAPI := jikan{id: id, l: s.l, excludedTypes: bannedTypes, maxEpisodes: maxEpisodes, maxDuration: maxDuration}

	// Request data from API
	results, err := getMovieData(&sourceAPI)

	if err != nil {
		data.ErrorMessage = append(data.ErrorMessage, err.Error())
		return nil, fmt.Errorf("Error while accessing Jikan API: %v", err)
	}

	return results, nil
}

var re_tmdbToken = regexp.MustCompile(`[^\/]*\/title\/(tt[0-9]*)`)

func (s *Server) handleTmdb(data *dataAddMovie, w http.ResponseWriter, r *http.Request, sourcelink string) ([]string, error) {

	tmdbEnabled, err := s.data.GetCfgBool("TmdbEnabled", DefaultTmdbEnabled)
	if err != nil {
		data.ErrorMessage = append(data.ErrorMessage, "Something went wrong :C")
		return nil, fmt.Errorf("Error while retriving config value 'TmdbEnabled':\n %v", err)
	}

	if !tmdbEnabled {
		s.l.Debug("Aborting Tmdb autofill since it is not enabled")
		data.ErrorMessage = append(data.ErrorMessage, "Tmdb API usage was not enabled by the site administrator")
		return nil, fmt.Errorf("Tmdb not enabled")
	}

	// Retrieve token from database
	token, err := s.data.GetCfgString("TmdbToken", "")
	if err != nil || token == "" {
		s.l.Debug("Aborting Tmdb autofill since no token was found")
		data.ErrorMessage = append(data.ErrorMessage, "TmdbToken is either empty or not set in the admin config")
		return nil, fmt.Errorf("TmdbToken is either empty or not set in the admin config")
	}
	// get the movie id
	match := re_tmdbToken.FindStringSubmatch(sourcelink)
	var id string
	if len(match) < 2 {
		s.l.Debug("Regex match didn't find the movie id in %v", sourcelink)
		data.ErrorMessage = append(data.ErrorMessage, "Could not retrive movie id from provided link")
		data.ErrLinks = true
		return nil, fmt.Errorf("Could not retrive movie id from link")
	}
	id = match[1]

	sourceAPI := tmdb{id: id, token: token, l: s.l}

	// Request data from API
	results, err := getMovieData(&sourceAPI)

	if err != nil {
		s.l.Error("Error while accessing Tmdb API: %v", err)
		data.ErrorMessage = append(data.ErrorMessage, err.Error())
		return nil, err
	}

	return results, nil
}

func (s *Server) handleFormfill(data *dataAddMovie, w http.ResponseWriter, r *http.Request) (results []string, links []*common.Link) {
	// Get all links from the corresponding input field
	linktext := strings.ReplaceAll(r.FormValue("Links"), "\r", "")
	data.ValLinks = linktext

	// Get the remarks from the corresponding input field
	remarkstext := strings.ReplaceAll(r.FormValue("Remarks"), "\r", "")
	data.ValRemarks = remarkstext

	// Check link maxlength
	maxLinkLength, err := s.data.GetCfgInt(ConfigMaxLinkLength, DefaultMaxLinkLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxLinkLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return nil, nil
	}

	if common.GetStringLength(linktext) > maxLinkLength {
		s.l.Debug("Links too long: %d", common.GetStringLength(linktext))
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Links too long! Max Length: %d characters", maxLinkLength))
		data.ErrLinks = true
	}

	// Check for links
	linkstrings := strings.Split(linktext, "\n")
	if len(linkstrings) == 0 {
		s.l.Error("no links given")
		data.ErrorMessage = append(data.ErrorMessage, "No link found.")
		data.ErrLinks = true
	}

	// Convert links to structs
	for id, link := range linkstrings {

		ls, err := common.NewLink(link, id)

		if err != nil {
			s.l.Error("Cannot add link")
			data.ErrorMessage = append(data.ErrorMessage, "Could not add link: %v", err.Error())
			data.ErrLinks = true
		}

		links = append(links, ls)
	}

	// Check Remarks max length
	maxRemarksLength, err := s.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxRemarksLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	if common.GetStringLength(remarkstext) > maxRemarksLength {
		s.l.Debug("Remarks too long: %d", common.GetStringLength(remarkstext))
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Remarks too long! Max Length: %d characters", maxRemarksLength))
		data.ErrRemarks = true
	}

	// Here we continue with the other input checks
	maxTitleLength, err := s.data.GetCfgInt(ConfigMaxTitleLength, DefaultMaxTitleLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxTitleLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	title := strings.TrimSpace(r.FormValue("MovieName"))
	data.ValTitle = title

	if data.ValTitle == "" {
		data.ErrorMessage = append(data.ErrorMessage, "Missing movie title")
		data.ErrTitle = true
	}

	if common.GetStringLength(data.ValTitle) > maxTitleLength {
		s.l.Debug("Title too long: %d", common.GetStringLength(data.ValTitle))
		data.ErrTitle = true
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Title too long! Max Length: %d characters", maxTitleLength))
	} else if common.GetStringLength(common.CleanMovieName(data.ValTitle)) == 0 {
		s.l.Debug("Title too short: %d", common.GetStringLength(common.CleanMovieName(data.ValTitle)))
		data.ErrTitle = true
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Title too short! Min Length: %d characters", 1))
	}

	movieExists, err := s.data.CheckMovieExists(title)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to check if movie exists: %v", err),
			w, r)
		return
	}

	if movieExists {
		data.ErrTitle = true
		s.l.Debug("Movie exists")
		data.ErrorMessage = append(data.ErrorMessage, "Movie already exists")
	}

	descr := strings.TrimSpace(r.FormValue("Description"))
	data.ValDescription = descr

	maxDescriptionLength, err := s.data.GetCfgInt(ConfigMaxDescriptionLength, DefaultMaxDescriptionLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxTitleLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	if common.GetStringLength(data.ValDescription) > maxDescriptionLength {
		s.l.Debug("Description too long: %d", common.GetStringLength(data.ValDescription))
		data.ErrDescription = true
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Description too long! Max Length: %d characters", maxDescriptionLength))
	}

	if common.GetStringLength(descr) == 0 {
		data.ErrDescription = true
		data.ErrorMessage = append(data.ErrorMessage, "Missing description")
	}

	var posterpath string

	posterFileName := strings.TrimSpace(r.FormValue("MovieName"))

	posterFile, _, err := r.FormFile("PosterFile")

	if posterFile != nil {
		if err != nil {
			s.l.Error("Parsing of the uploaded file resulted in the following error: %v", err.Error())
			data.ErrPoster = true
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		}

		file, err := s.uploadFile(r, posterFileName)

		if err != nil {
			s.l.Error("Upload of the file was not possible: %v", err.Error())
			data.ErrPoster = true
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		} else {
			posterpath = filepath.Base(file)
		}
	} else {
		posterpath = "unknown.jpg"
	}

	if data.isError() {
		return nil, nil
	}

	results = append(results, title, descr, posterpath, remarkstext)

	return results, links
}

func (s *Server) handlerUser(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	totalVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		s.l.Error("Error getting MaxUserVotes config setting: %v", err)
		totalVotes = DefaultMaxUserVotes
	}

	activeVotes, watchedVotes, err := s.getUserVotes(user)
	if err != nil {
		s.l.Error("Unable to get votes for user %d: %v", user.Id, err)
	}

	addedMovies, err := s.data.GetUserMovies(user.Id)
	if err != nil {
		s.l.Error("Unable to get movies added by user %d: %v", user.Id, err)
	}

	unlimited, err := s.data.GetCfgBool(ConfigUnlimitedVotes, DefaultUnlimitedVotes)
	if err != nil {
		s.l.Error("Error getting %s config setting: %v", ConfigUnlimitedVotes, err)
	}

	data := struct {
		dataPageBase

		User *common.User

		TotalVotes     int
		AvailableVotes int
		UnlimitedVotes bool

		OAuthEnabled        bool
		TwitchOAuthEnabled  bool
		DiscordOAuthEnabled bool
		PatreonOAuthEnabled bool

		HasLocal   bool
		HasTwitch  bool
		HasDiscord bool
		HasPatreon bool

		ActiveVotes    []*common.Movie
		WatchedVotes   []*common.Movie
		AddedMovies    []*common.Movie
		SuccessMessage string

		PassError   []string
		NotifyError []string
		EmailError  []string

		ErrCurrentPass bool
		ErrNewPass     bool
		ErrEmail       bool
	}{
		dataPageBase: s.newPageBase("Account", w, r),

		User: user,

		TotalVotes:     totalVotes,
		AvailableVotes: totalVotes - len(activeVotes),
		UnlimitedVotes: unlimited,

		ActiveVotes:  activeVotes,
		WatchedVotes: watchedVotes,
		AddedMovies:  addedMovies,
	}

	twitchAuth, err := s.data.GetCfgBool(ConfigTwitchOauthEnabled, DefaultTwitchOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthEnabled config value: %v", err)
		return
	}
	data.TwitchOAuthEnabled = twitchAuth

	discordAuth, err := s.data.GetCfgBool(ConfigDiscordOauthEnabled, DefaultDiscordOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthEnabled config value: %v", err)
		return
	}
	data.DiscordOAuthEnabled = discordAuth

	patreonAuth, err := s.data.GetCfgBool(ConfigPatreonOauthEnabled, DefaultPatreonOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthEnabled config value: %v", err)
		return
	}
	data.PatreonOAuthEnabled = patreonAuth

	data.OAuthEnabled = twitchAuth || discordAuth || patreonAuth

	_, err = user.GetAuthMethod(common.AUTH_LOCAL)
	data.HasLocal = err == nil
	_, err = user.GetAuthMethod(common.AUTH_TWITCH)
	data.HasTwitch = err == nil
	_, err = user.GetAuthMethod(common.AUTH_DISCORD)
	data.HasDiscord = err == nil
	_, err = user.GetAuthMethod(common.AUTH_PATREON)
	data.HasPatreon = err == nil

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			s.l.Error("ParseForm() error: %v", err)
			s.doError(http.StatusInternalServerError, "Form error", w, r)
			return
		}

		formVal := r.PostFormValue("Form")
		if formVal == "ChangePassword" {
			// Do password stuff
			currentPass := s.hashPassword(r.PostFormValue("PasswordCurrent"))
			newPass1_raw := r.PostFormValue("PasswordNew1")
			newPass2_raw := r.PostFormValue("PasswordNew2")

			localAuth, err := user.GetAuthMethod(common.AUTH_LOCAL)
			if err != nil {
				data.ErrCurrentPass = true
				data.PassError = append(data.PassError, "No Password detected.")
			} else {

				if currentPass != localAuth.Password {
					data.ErrCurrentPass = true
					data.PassError = append(data.PassError, "Invalid current password")
				}

				if newPass1_raw == "" {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "New password cannot be blank")
				}

				if newPass1_raw != newPass2_raw {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "Passwords do not match")
				}
				if !(data.ErrCurrentPass || data.ErrNewPass || data.ErrEmail) {
					// Change pass
					data.SuccessMessage = "Password successfully changed"
					localAuth.Password = s.hashPassword(newPass1_raw)
					localAuth.Date = time.Now()

					if err = s.data.UpdateAuthMethod(localAuth); err != nil {
						s.l.Error("Unable to save User with new password:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
						return
					}

					s.l.Info("new Date_Local: %s", localAuth.Date)
					err = s.login(user, common.AUTH_LOCAL, w, r)
					if err != nil {
						s.l.Error("Unable to login to session:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
						return
					}
				}
			}
		} else if formVal == "Notifications" {
			// Update notifications
		} else if formVal == "SetPassword" {
			pass1_raw := r.PostFormValue("Password1")
			pass2_raw := r.PostFormValue("Password2")

			_, err := user.GetAuthMethod(common.AUTH_LOCAL)
			if err == nil {
				data.ErrCurrentPass = true
				data.PassError = append(data.PassError, "Existing password detected. (how did you end up here anyways?)")
			} else {
				localAuth := &common.AuthMethod{
					Type: common.AUTH_LOCAL,
				}

				if pass1_raw == "" {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "New password cannot be blank")
				}

				if pass1_raw != pass2_raw {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "Passwords do not match")
				}
				if !(data.ErrCurrentPass || data.ErrNewPass || data.ErrEmail) {
					// Change pass
					data.SuccessMessage = "Password successfully set"
					localAuth.Password = s.hashPassword(pass1_raw)
					localAuth.Date = time.Now()
					s.l.Info("new Date_Local: %s", localAuth.Date)

					user, err = s.AddAuthMethodToUser(localAuth, user)

					if err != nil {
						s.l.Error("Unable to add AuthMethod %s to user %s", localAuth.Type, user.Name)
						s.doError(http.StatusInternalServerError, "Unable to link password to user", w, r)
					}

					s.data.UpdateUser(user)

					if err != nil {
						s.l.Error("Unable to update user %s", user.Name)
						s.doError(http.StatusInternalServerError, "Unable to update user", w, r)
					}

					err = s.login(user, common.AUTH_LOCAL, w, r)
					if err != nil {
						s.l.Error("Unable to login to session:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
					}

					http.Redirect(w, r, "/user", http.StatusFound)
				}
			}
		}
	}
	if err := s.executeTemplate(w, "account", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
func (s *Server) handlerUserLogin(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		s.l.Error("Error parsing login form: %v", err)
	}

	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/user", http.StatusFound)
		return
	}

	data := dataLoginForm{}
	doRedirect := false

	twitchAuth, err := s.data.GetCfgBool(ConfigTwitchOauthEnabled, DefaultTwitchOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthEnabled config value: %v", err)
		return
	}
	data.TwitchOAuth = twitchAuth

	discordAuth, err := s.data.GetCfgBool(ConfigDiscordOauthEnabled, DefaultDiscordOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthEnabled config value: %v", err)
		return
	}
	data.DiscordOAuth = discordAuth

	patreonAuth, err := s.data.GetCfgBool(ConfigPatreonOauthEnabled, DefaultPatreonOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthEnabled config value: %v", err)
		return
	}
	data.PatreonOAuth = patreonAuth

	data.OAuth = twitchAuth || discordAuth || patreonAuth

	if r.Method == "POST" {
		// do login

		un := r.PostFormValue("Username")
		pw := r.PostFormValue("Password")
		user, err = s.data.UserLocalLogin(un, s.hashPassword(pw))
		if err != nil {
			data.ErrorMessage = err.Error()
		} else {
			doRedirect = true
		}

	} else {
		s.l.Info("> no post: %s", r.Method)
	}

	if user != nil {
		err = s.login(user, common.AUTH_LOCAL, w, r)
		if err != nil {
			s.l.Error("Unable to login: %v", err)
			s.doError(http.StatusInternalServerError, "Unable to login", w, r)
			return
		}
	}

	// Redirect to base page on successful login
	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data.dataPageBase = s.newPageBase("Login", w, r) // set this last to get correct login status

	if err := s.executeTemplate(w, "simplelogin", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerUserLogout(w http.ResponseWriter, r *http.Request) {
	err := s.logout(w, r)
	if err != nil {
		s.l.Error("Error logging out: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handlerUserNew(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}

	data := struct {
		dataPageBase

		ErrorMessage []string
		ErrName      bool
		ErrPass      bool
		ErrEmail     bool

		OAuth         bool
		TwitchOAuth   bool
		TwitchSignup  bool
		DiscordOAuth  bool
		DiscordSignup bool
		PatreonOAuth  bool
		PatreonSignup bool
		LocalSignup   bool

		ValName           string
		ValEmail          string
		ValNotifyEnd      bool
		ValNotifySelected bool
	}{
		dataPageBase: s.newPageBase("Create Account", w, r),
	}

	doRedirect := false

	twitchAuth, err := s.data.GetCfgBool(ConfigTwitchOauthEnabled, DefaultTwitchOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthEnabled config value: %v", err)
		return
	}
	data.TwitchOAuth = twitchAuth

	twitchSignup, err := s.data.GetCfgBool(ConfigTwitchOauthSignupEnabled, DefaultTwitchOauthSignupEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthSignupEnabled config value: %v", err)
		return
	}
	data.TwitchSignup = twitchSignup

	discordAuth, err := s.data.GetCfgBool(ConfigDiscordOauthEnabled, DefaultDiscordOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthEnabled config value: %v", err)
		return
	}
	data.DiscordOAuth = discordAuth

	discordSignup, err := s.data.GetCfgBool(ConfigDiscordOauthSignupEnabled, DefaultDiscordOauthSignupEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthSignupEnabled config value: %v", err)
		return
	}
	data.DiscordSignup = discordSignup

	patreonAuth, err := s.data.GetCfgBool(ConfigPatreonOauthEnabled, DefaultPatreonOauthEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthEnabled config value: %v", err)
		return
	}
	data.PatreonOAuth = patreonAuth

	patreonSignup, err := s.data.GetCfgBool(ConfigPatreonOauthSignupEnabled, DefaultPatreonOauthSignupEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthSignupEnabled config value: %v", err)
		return
	}
	data.PatreonSignup = patreonSignup

	localSignup, err := s.data.GetCfgBool(ConfigLocalSignupEnabled, DefaultLocalSignupEnabled)
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigLocalSignupEnabled config value: %v", err)
		return
	}
	data.LocalSignup = localSignup

	data.OAuth = twitchAuth || discordAuth || patreonAuth

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			s.l.Error("Error parsing login form: %v", err)
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		}

		un := strings.TrimSpace(r.PostFormValue("Username"))
		data.ValName = un

		// TODO: password requirements
		pw1 := r.PostFormValue("Password1")
		pw2 := r.PostFormValue("Password2")

		data.ValName = un

		if un == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Username cannot be blank!")
			data.ErrName = true
		}

		maxlen, err := s.data.GetCfgInt(ConfigMaxNameLength, DefaultMaxNameLength)
		if err != nil {
			s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
			s.l.Error("Unable to get MaxNameLength config value: %v", err)
			return
		}

		minlen, err := s.data.GetCfgInt(ConfigMinNameLength, DefaultMinNameLength)
		if err != nil {
			s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
			s.l.Error("Unable to get MinNameLength config value: %v", err)
			return
		}

		s.l.Debug("New user: %s (%d) maxlen: %d", un, len(un), maxlen)

		if len(un) > maxlen {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Username cannot be longer than %d characters", maxlen))
			data.ErrName = true
		}

		if len(un) < minlen {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Username cannot be shorter than %d characters", minlen))
			data.ErrName = true
		}

		if pw1 != pw2 {
			data.ErrorMessage = append(data.ErrorMessage, "Passwords do not match!")
			data.ErrPass = true

		} else if pw1 == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Password cannot be blank!")
			data.ErrPass = true
		}

		notifyEnd := r.PostFormValue("NotifyEnd")
		notifySelected := r.PostFormValue("NotifySelected")
		email := r.PostFormValue("Email")

		data.ValEmail = email
		if notifyEnd != "" {
			data.ValNotifyEnd = true
		}

		if notifySelected != "" {
			data.ValNotifySelected = true
		}

		if (notifyEnd != "" || notifySelected != "") && email == "" {
			data.ErrEmail = true
			data.ErrorMessage = append(data.ErrorMessage, "Email required for notifications")
		}

		auth := &common.AuthMethod{
			Type:     common.AUTH_LOCAL,
			Password: s.hashPassword(pw1),
			Date:     time.Now(),
		}

		if err != nil {
			s.l.Error(err.Error())
			data.ErrorMessage = append(data.ErrorMessage, "Could not create new User, message the server admin")
		}

		if len(data.ErrorMessage) == 0 {
			newUser := &common.User{
				Name:                un,
				Email:               email,
				NotifyCycleEnd:      data.ValNotifyEnd,
				NotifyVoteSelection: data.ValNotifySelected,
			}

			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			newUser.Id, err = s.data.AddUser(newUser)
			if err != nil {
				data.ErrorMessage = append(data.ErrorMessage, err.Error())
			} else {
				err = s.login(newUser, common.AUTH_LOCAL, w, r)
				if err != nil {
					s.l.Error("Unable to login to session: %v", err)
					s.doError(http.StatusInternalServerError, "Login error", w, r)
					return
				}
				doRedirect = true
			}
		}
	}

	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := s.executeTemplate(w, "newaccount", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// Toggles votes
func (s *Server) handlerVote(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	enabled, err := s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)
	if err != nil {
		s.l.Error("Unable to get config value for VotingEnabled: %s", err)
	}

	// this should be false if an error was returned
	if !enabled {
		s.doError(
			http.StatusBadRequest,
			"Voting is not enabled",
			w, r)
		return
	}

	var movieId int
	if _, err := fmt.Sscanf(r.URL.Path, "/vote/%d", &movieId); err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		s.l.Info("invalid vote URL: %q", r.URL.Path)
		return
	}

	movie, err := s.data.GetMovie(movieId)
	if err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		s.l.Info("Movie with ID %d doesn't exist", movieId)
		return
	}
	if movie.CycleWatched != nil {
		s.doError(http.StatusBadRequest, "Movie already watched", w, r)
		s.l.Error("Attempted to vote on watched movie ID %d", movieId)
		return
	}

	userVoted, err := s.data.UserVotedForMovie(user.Id, movieId)
	if err != nil {
		s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
		s.l.Error("Cannot get user vote: %v", err)
		return
	}

	if userVoted {
		//s.doError(http.StatusBadRequest, "You already voted for that movie!", w, r)
		if err := s.data.DeleteVote(user.Id, movieId); err != nil {
			s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
			s.l.Error("Unable to remove vote: %v", err)
			return
		}
	} else {

		unlimited, err := s.data.GetCfgBool(ConfigUnlimitedVotes, DefaultUnlimitedVotes)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get unlimited vote setting: %v", err),
				w, r)
			return
		}

		if !unlimited {
			// TODO: implement this on the data layer
			votedMovies, err := s.data.GetUserVotes(user.Id)
			if err != nil {
				s.doError(
					http.StatusBadRequest,
					fmt.Sprintf("Cannot get user votes: %v", err),
					w, r)
				return
			}

			count := 0
			for _, movie := range votedMovies {
				// Only count active movies
				if movie.CycleWatched == nil && movie.Removed == false {
					count++
				}
			}

			maxVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
			if err != nil {
				s.l.Error("Error getting MaxUserVotes config setting: %v", err)
				maxVotes = DefaultMaxUserVotes
			}

			if count >= maxVotes {
				s.doError(http.StatusBadRequest,
					"You don't have any more available votes!",
					w, r)
				return
			}
		}

		if err := s.data.AddVote(user.Id, movieId); err != nil {
			s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
			s.l.Error("Unable to cast vote: %v", err)
			return
		}
	}

	ref := r.Header.Get("Referer")
	if ref == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	http.Redirect(w, r, ref, http.StatusFound)
}
