package server

func (s *Server) handlerAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	data := dataAdminHome{
		dataPageBase: s.newPageBase("Admin", w, r),

		Cycle: cycle,
	}

	if err := s.executeTemplate(w, "adminHome", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	ulist, err := s.data.GetUsers(-1, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Error getting users: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase

		Users []*common.User
	}{
		dataPageBase: s.newPageBase("Admin - Users", w, r),
		Users:        ulist,
	}

	if err := s.executeTemplate(w, "adminUsers", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var uid int
	_, err := fmt.Sscanf(r.URL.Path, "/admin/user/%d", &uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Unable to parse user ID: %v", err),
			w, r)
		return
	}

	user, err := s.data.GetUser(uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get user: %v", err),
			w, r)
		return
	}

	action := r.URL.Query().Get("action")
	var urlKey *common.UrlKey
	switch action {
	//case "edit":
	//	// current function
	case "delete":
		s.adminDeleteUser(w, r, user)
		return
	case "ban":
		s.adminBanUser(w, r, user)
		return
	case "purge":
		s.adminPurgeUser(w, r, user)
		return
	case "password":
		urlKey, err = common.NewPasswordResetKey(user.Id)
		if err != nil {
			s.l.Error("Unable to generate UrlKey pair for user password reset: %v", err)
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to generate UrlKey pair for user password reset: %v", err),
				w, r)
			return
		}

		s.l.Debug("Saving new urlKey with URL %s", urlKey.Url)
		s.urlKeys[urlKey.Url] = urlKey
	}

	totalVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		s.l.Error("Error getting MaxUserVotes config setting: %v", err)
	}

	activeVotes, _, err := s.getUserVotes(user)
	if err != nil {
		s.l.Error("Unable to get votes for user %d: %v", user.Id, err)
	}

	host, err := s.data.GetCfgString(ConfigHostAddress, "http://<host>")
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get server host address: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase

		User         *common.User
		CurrentVotes []*common.Movie
		//PastVotes      []*common.Movie
		AvailableVotes int

		PassError   []string
		NotifyError []string
		UrlKey      *common.UrlKey
		Host        string
	}{
		dataPageBase: s.newPageBase("Admin - User Edit", w, r),

		User:         user,
		CurrentVotes: activeVotes,
		//PastVotes:      watchedVotes,
		AvailableVotes: totalVotes - len(activeVotes),
		UrlKey:         urlKey,
		Host:           host,
	}

	// FIXME: implement this
	if r.Method == "POST" {
	}

	if err := s.executeTemplate(w, "adminUserEdit", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminConfig(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	data := struct {
		dataPageBase

		ErrorMessage []string
		Values       []configValue

		TypeString ConfigValueType
		TypeBool   ConfigValueType
		TypeInt    ConfigValueType
		TypeKey    ConfigValueType
	}{
		ErrorMessage: []string{},
		Values: []configValue{
			configValue{Key: ConfigHostAddress, Default: "", Type: ConfigString},
			configValue{Key: ConfigNoticeBanner, Default: "", Type: ConfigString},

			configValue{Key: ConfigVotingEnabled, Default: DefaultVotingEnabled, Type: ConfigBool},
			configValue{Key: ConfigMaxUserVotes, Default: DefaultMaxUserVotes, Type: ConfigInt},
			configValue{Key: ConfigEntriesRequireApproval, Default: DefaultEntriesRequireApproval, Type: ConfigBool},

			configValue{Key: ConfigJikanEnabled, Default: DefaultJikanEnabled, Type: ConfigBool},
			configValue{Key: ConfigJikanBannedTypes, Default: DefaultJikanBannedTypes, Type: ConfigString},
			configValue{Key: ConfigJikanMaxEpisodes, Default: DefaultJikanMaxEpisodes, Type: ConfigInt},
			configValue{Key: ConfigTmdbEnabled, Default: DefaultTmdbEnabled, Type: ConfigBool},
			configValue{Key: ConfigTmdbToken, Default: DefaultTmdbToken, Type: ConfigKey},

			configValue{Key: ConfigFormfillEnabled, Default: DefaultFormfillEnabled, Type: ConfigBool},

			configValue{Key: ConfigMaxNameLength, Default: DefaultMaxNameLength, Type: ConfigInt},
			configValue{Key: ConfigMinNameLength, Default: DefaultMinNameLength, Type: ConfigInt},

			configValue{Key: ConfigMaxTitleLength, Default: DefaultMaxTitleLength, Type: ConfigInt},
			configValue{Key: ConfigMaxDescriptionLength, Default: DefaultMaxDescriptionLength, Type: ConfigInt},
			configValue{Key: ConfigMaxLinkLength, Default: DefaultMaxLinkLength, Type: ConfigInt},
			configValue{Key: ConfigMaxRemarksLength, Default: DefaultMaxRemarksLength, Type: ConfigInt},
			configValue{Key: ConfigMaxMultEpLength, Default: DefaultMaxMultEpLength, Type: ConfigInt},

			configValue{Key: ConfigUnlimitedVotes, Default: DefaultUnlimitedVotes, Type: ConfigBool},

			configValue{Key: ConfigLocalSignupEnabled, Default: DefaultLocalSignupEnabled, Type: ConfigBool},
			configValue{Key: ConfigTwitchOauthEnabled, Default: DefaultTwitchOauthEnabled, Type: ConfigBool},
			configValue{Key: ConfigTwitchOauthSignupEnabled, Default: DefaultTwitchOauthSignupEnabled, Type: ConfigBool},
			configValue{Key: ConfigTwitchOauthClientID, Default: DefaultTwitchOauthClientID, Type: ConfigString},
			configValue{Key: ConfigTwitchOauthClientSecret, Default: DefaultTwitchOauthClientSecret, Type: ConfigString},
			configValue{Key: ConfigDiscordOauthEnabled, Default: DefaultDiscordOauthEnabled, Type: ConfigBool},
			configValue{Key: ConfigDiscordOauthSignupEnabled, Default: DefaultDiscordOauthSignupEnabled, Type: ConfigBool},
			configValue{Key: ConfigDiscordOauthClientID, Default: DefaultDiscordOauthClientID, Type: ConfigString},
			configValue{Key: ConfigDiscordOauthClientSecret, Default: DefaultDiscordOauthClientSecret, Type: ConfigString},
			configValue{Key: ConfigPatreonOauthEnabled, Default: DefaultPatreonOauthEnabled, Type: ConfigBool},
			configValue{Key: ConfigPatreonOauthSignupEnabled, Default: DefaultPatreonOauthSignupEnabled, Type: ConfigBool},
			configValue{Key: ConfigPatreonOauthClientID, Default: DefaultPatreonOauthClientID, Type: ConfigString},
			configValue{Key: ConfigPatreonOauthClientSecret, Default: DefaultPatreonOauthClientSecret, Type: ConfigString},
		},

		TypeString: ConfigString,
		TypeBool:   ConfigBool,
		TypeInt:    ConfigInt,
		TypeKey:    ConfigKey,
	}

	var err error

	if r.Method == "POST" {
		if err = r.ParseForm(); err != nil {
			s.l.Error("Unable to parse form: %v", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		for _, val := range data.Values {
			str := r.PostFormValue(val.Key)
			switch val.Type {
			case ConfigString, ConfigKey:
				err = s.data.SetCfgString(val.Key, str)
				if err != nil {
					data.ErrorMessage = append(
						data.ErrorMessage,
						fmt.Sprintf("Unable to save string value for %s: %v", val.Key, err))
				}

			case ConfigInt:
				intVal, err := strconv.ParseInt(str, 9, 32)
				if err != nil {
					data.ErrorMessage = append(
						data.ErrorMessage,
						fmt.Sprintf("Value for %q is invalid: %v", val.Key, err))
				} else {
					err = s.data.SetCfgInt(val.Key, int(intVal))
					if err != nil {
						data.ErrorMessage = append(
							data.ErrorMessage,
							fmt.Sprintf("Unable to save int value for %s: %v", val.Key, err))
					}
				}

			case ConfigBool:
				boolVal := false
				if str != "" {
					boolVal = true
				}
				err = s.data.SetCfgBool(val.Key, boolVal)
				if err != nil {
					data.ErrorMessage = append(
						data.ErrorMessage,
						fmt.Sprintf("Unable to save bool value for %s: %v", val.Key, err))
				}

			default:
				data.ErrorMessage = append(
					data.ErrorMessage,
					fmt.Sprintf("Unknown config value type for %s: %v", val.Key, val.Type))
			}
		}

		// Don't enable this stuff for now
		//if clearPassSalt := r.PostFormValue("ClearPassSalt"); clearPassSalt != "" {
		//	s.data.DeleteCfgKey("PassSalt")
		//}

		//if clearCookies := r.PostFormValue("ClearCookies"); clearCookies != "" {
		//	s.data.DeleteCfgKey("SessionAuth")
		//	s.data.DeleteCfgKey("SessionEncrypt")
		//}
	}

	newValues := []configValue{}
	for _, val := range data.Values {
		var v interface{}

		switch val.Type {
		case ConfigString, ConfigKey:
			v, err = s.data.GetCfgString(val.Key, val.Default.(string))
		case ConfigBool:
			v, err = s.data.GetCfgBool(val.Key, val.Default.(bool))
		case ConfigInt:
			v, err = s.data.GetCfgInt(val.Key, val.Default.(int))
		}

		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("Unable to get config value for %s: %v", val.Key, err))
		} else {
			val.Value = v
		}

		newValues = append(newValues, val)
	}
	data.Values = newValues

	// Set this down here so the Notice Banner is updated
	data.dataPageBase = s.newPageBase("Admin - Config", w, r)

	// Reload Oauth
	err = s.initOauth()

	if err != nil {
		data.ErrorMessage = append(data.ErrorMessage, err.Error())
	}

	// getting ALL the booleans
	var localSignup, twitchSignup, patreonSignup, discordSignup, twitchOauth, patreonOauth, discordOauth bool

	for _, val := range data.Values {
		bval, ok := val.Value.(bool)
		if !ok && val.Type == ConfigBool {
			data.ErrorMessage = append(data.ErrorMessage, "Could not parse field %s as boolean", val.Key)
			break
		}
		switch val.Key {
		case ConfigLocalSignupEnabled:
			localSignup = bval
		case ConfigTwitchOauthSignupEnabled:
			twitchSignup = bval
		case ConfigTwitchOauthEnabled:
			twitchOauth = bval
		case ConfigDiscordOauthSignupEnabled:
			discordSignup = bval
		case ConfigDiscordOauthEnabled:
			discordOauth = bval
		case ConfigPatreonOauthSignupEnabled:
			patreonSignup = bval
		case ConfigPatreonOauthEnabled:
			patreonOauth = bval
		}
	}

	// Check that we have atleast ONE signup method enabled
	if !(localSignup || twitchSignup || discordSignup || patreonSignup) {
		data.ErrorMessage = append(data.ErrorMessage, "No Signup method is currently enabled, please ensure to enable atleast one method")
	}

	// Check that the corresponding oauth for the signup is enabled
	if twitchSignup && !twitchOauth {
		data.ErrorMessage = append(data.ErrorMessage, "To enable twitch signup you need to also enable twitch Oauth (and fill the token/secret)")
	}

	if discordSignup && !discordOauth {
		data.ErrorMessage = append(data.ErrorMessage, "To enable discord signup you need to also enable discord Oauth (and fill the token/secret)")
	}

	if patreonSignup && !patreonOauth {
		data.ErrorMessage = append(data.ErrorMessage, "To enable patreon signup you need to also enable patreon Oauth (and fill the token/secret)")
	}

	users, err := s.data.GetUsersWithAuth(common.AUTH_TWITCH, true)
	if err, ok := err.(*common.ErrNoUsersFound); !ok || err == nil {
		if (len(users) != -1) && !twitchOauth {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Disabling Twitch Oauth would cause %d users to be unable to login since they only have this auth method associated.", len(users)))
		}
	}

	users, err = s.data.GetUsersWithAuth(common.AUTH_PATREON, true)
	if err, ok := err.(*common.ErrNoUsersFound); !ok || err == nil {
		if (len(users) != -1) && !patreonOauth {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Disabling Patreon Oauth would cause %d users to be unable to login since they only have this auth method associated.", len(users)))
		}
	}

	users, err = s.data.GetUsersWithAuth(common.AUTH_DISCORD, true)
	if err, ok := err.(*common.ErrNoUsersFound); !ok || err == nil {
		if (len(users) != -1) && !discordOauth {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Disabling Discord Oauth would cause %d users to be unable to login since they only have this auth method associated.", len(users)))
		}
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminMovieEdit(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var mid int
	_, err := fmt.Sscanf(r.URL.Path, "/admin/movie/%d", &mid)
	if err != nil {
		s.l.Error("Unable to parse movie ID: %v", err)
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Unable to parse movie ID: %v", err),
			w, r)
		return
	}

	// TODO: Approve and Deny actions
	action := r.URL.Query().Get("action")
	switch action {
	case "remove":
		// TODO: Confirmation before removing
		err = s.data.RemoveMovie(mid)
		if err != nil {
			s.l.Error("Unable to remove movie with ID %d: %v", mid, err)
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to remove movie with ID %d: %v", mid, err),
				w, r)
			return
		}

		http.Redirect(w, r, "/admin/movies", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4095)
		if err != nil {
			s.l.Error("Unable to parse form: %v", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		movie, err := s.data.GetMovie(mid)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get movie: %v", err),
				w, r)
			return
		}

		movie.Name = r.PostFormValue("MovieName")
		movie.Description = r.PostFormValue("MovieDescr")

		linktext := strings.ReplaceAll(r.FormValue("MovieLinks"), "\r", "")

		links := strings.Split(linktext, "\n")
		// links is a slice of strings now
		s.l.Debug("Links: %s", links)

		linkstructs := []*common.Link{}

		// Convert links to structs
		for id, link := range links {

			ls, err := common.NewLink(link, id)

			if err != nil || ls == nil {
				s.l.Error("Cannot add link: %v", link)
				continue
			}

			id, err := s.data.AddLink(ls)
			if err != nil {
				s.l.Error("Could not add link struct to db: %v", err.Error())
				continue
			}
			ls.Id = id

			linkstructs = append(linkstructs, ls)
		}
		movie.Links = linkstructs

		posterFileName := strings.TrimSpace(r.FormValue("MovieName"))
		posterFile, _, _ := r.FormFile("PosterFile")

		if posterFile != nil {
			file, err := s.uploadFile(r, posterFileName)

			if err != nil {
				//data.ErrPoster = true
				//errText = append(errText, err.Error())
				s.l.Error("Unable to upload file: %v", err)
			} else {
				movie.Poster = filepath.Base(file)
			}
		}

		err = s.data.UpdateMovie(movie)
		if err != nil {
			s.l.Error("Unable to update movie: %v", err)
		}
	}

	movie, err := s.data.GetMovie(mid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get movie: %v", err),
			w, r)
		return
	}

	linktext := ""
	for _, link := range movie.Links {
		linktext = linktext + link.Url + "\n"
	}

	data := struct {
		dataPageBase
		Movie    *common.Movie
		LinkText string
	}{
		dataPageBase: s.newPageBase("Admin - Movies", w, r),
		Movie:        movie,
		LinkText:     linktext,
	}

	if err := s.executeTemplate(w, "adminMovieEdit", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminMovies(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	active, err := s.data.GetActiveMovies()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get active movies: %v", err),
			w, r)
		return
	}

	approval, err := s.data.GetCfgBool(ConfigEntriesRequireApproval, DefaultEntriesRequireApproval)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get entries require approval setting: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase
		Active  []*common.Movie
		Past    []*common.Movie
		Pending []*common.Movie

		RequireApproval bool
	}{
		dataPageBase: s.newPageBase("Admin - Movies", w, r),
		Active:       common.SortMoviesByName(active),
		//Pending:      common.SortMoviesByName(active),

		RequireApproval: approval,
	}

	if err := s.executeTemplate(w, "adminMovies", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminCycles_Post(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
		return
	}

	s.l.Debug("Cycle post")

	err := r.ParseForm()
	if err != nil {
		s.l.Error("Unable to parse form: %v", err)
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to parse form: %v", err), w, r)
		return
	}

	var plannedEnd *time.Time
	action := r.PostFormValue("actionType")
	s.l.Debug("Form action: %s", action)

	switch action {
	case "update":
		dateStr := strings.TrimSpace(r.PostFormValue("modEndDate"))
		if dateStr == "" {
			s.l.Debug("No date given in update")
			http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
			return
		}

		cycle, err := s.data.GetCurrentCycle()
		if err != nil {
			s.l.Error(err.Error())
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
			return
		}

		end, err := time.Parse("2005-01-02", dateStr)
		if err != nil {
			s.l.Error(err.Error())
		} else {
			t := (&end).Round(time.Second)
			cycle.PlannedEnd = &t
		}

		err = s.data.UpdateCycle(cycle)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
			return
		}

	case "create":
		end, err := time.Parse("2005-01-02", r.PostFormValue("endDate"))
		if err != nil {
			s.l.Error(err.Error())
		} else {
			t := (&end).Round(time.Second)
			plannedEnd = &t
		}

		_, err = s.data.AddCycle(plannedEnd)
		if err != nil {
			s.l.Error("Unable to add cycle: %v", err)
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to add cycle: %v", err), w, r)
			return
		}

		// Re-enable voting after successfully starting a new cycle
		err = s.data.SetCfgBool(ConfigVotingEnabled, true)
		if err != nil {
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to enable voting: %v", err), w, r)
			return
		}
	}

	http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
}

func (s *Server) handlerAdminCycles(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var action string
	if r.Method == "POST" {
		r.ParseForm()
		action = r.PostFormValue("action")
		s.l.Debug("POSTed values: %s", r.PostForm)
	}

	// URL parameters override POST
	if val := r.URL.Query().Get("action"); val != "" {
		action = val
	}

	s.l.Debug("action: %q", r.URL.Query().Get("action"))
	switch action {
	case "end":
		//adminEndCycle(w, r)
		s.cycleStage0(w, r)
		return

	case "cancel":
		s.l.Info("Canceling cycle end")
		err := s.data.SetCfgBool(ConfigVotingEnabled, true)
		if err != nil {
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to enable voting: %v", err), w, r)
			return
		}

		r.Method = "GET"
		http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
		return

	case "select":
		s.cycleStage1(w, r)
		return
	}

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	data := struct {
		dataPageBase
		Cycle *common.Cycle
		Past  []*common.Cycle
	}{
		dataPageBase: s.newPageBase("Admin - Cycles", w, r),

		Cycle: cycle,
		Past:  []*common.Cycle{},
	}

	pastCycles, err := s.data.GetPastCycles(-1, 5)
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get past cycles: %v", err), w, r)
		return
	}

	data.Past = pastCycles
	s.l.Debug("found %d past cycles: %s", len(pastCycles), pastCycles)

	s.l.Debug("Executing admin cycles template")
	if err := s.executeTemplate(w, "adminCycles", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
