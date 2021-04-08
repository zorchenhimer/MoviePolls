package server

var re_auth = regexp.MustCompile(`^/auth/([^/#?]+)$`)

func (s *Server) handlerAuth(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)

	s.l.Debug("[auth] RawQuery: %s", r.URL.RawQuery)
	s.l.Debug("[auth] Path: %s", r.URL.Path)

	matches := re_auth.FindStringSubmatch(r.URL.Path)
	var urlKey *common.UrlKey
	var ok bool
	if len(matches) != 2 {
		s.l.Debug("[auth] len != 2; matches: %v", matches)
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	if urlKey, ok = s.urlKeys[matches[1]]; !ok {
		s.l.Debug("[auth] map !ok; matches: %v", matches)
		mkeys := []string{}
		for key, _ := range s.urlKeys {
			mkeys = append(mkeys, key)
		}
		s.l.Debug("[auth] map keys: %v", mkeys)
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	var formError string
	var key string
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			s.l.Error("[auth] ParseForm(): %v", err)
			s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
			return
		}

		key = strings.TrimSpace(r.PostFormValue("Key"))
		s.l.Debug("[auth] POST; key: %q", key)
	} else {
		s.l.Debug("[auth] RawQuery: %s", r.URL.RawQuery)
		key = r.URL.RawQuery
	}

	if key != "" && key != urlKey.Key {
		formError = "Invalid Key"
		goto renderPage
	}

	switch urlKey.Type {
	case common.UKT_AdminAuth:
		if user == nil {
			s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
			return
		}

		if key != "" {
			user.Privilege = 2
			err := s.data.UpdateUser(user)
			if err != nil {
				s.doError(
					http.StatusInternalServerError,
					fmt.Sprintf("Unable to update user: %v", err),
					w, r)
				return
			}

			s.l.Info("%s has claimed Admin", user.Name)
			delete(s.urlKeys, key)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

	case common.UKT_PasswordReset:
		s.l.Debug("Password top; key: %q", key)

		if key != "" {
			if r.Method == "POST" {
				s.l.Debug("Password POST")
				err := r.ParseForm()
				if err != nil {
					s.l.Error("[auth] ParseForm(): %v", err)
					s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
					return
				}

				pass1 := r.PostFormValue("password1")
				pass2 := r.PostFormValue("password2")

				if pass1 != pass2 {
					s.l.Debug("Passwords do not match match")
					formError = "Passwords do not match!"
				} else if pass1 == "" {
					s.l.Debug("Passwords are blank")
					formError = "Password cannot be blank!"
				} else {
					s.l.Debug("Passwords match, saving it")
					user, err := s.data.GetUser(urlKey.UserId)
					if err != nil {
						s.l.Error("[auth] GetUser(): %v", err)
						s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
						return
					} else if user == nil {
						s.l.Error("User not found with ID %d", urlKey.UserId)
						s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
						return
					}

					var localAuth *common.AuthMethod
					for _, auth := range user.AuthMethods {
						if auth.Type == common.AUTH_LOCAL {
							localAuth = auth
							break
						}
					}

					localAuth.Password = s.hashPassword(pass1)
					localAuth.Date = time.Now()

					if err = s.data.UpdateAuthMethod(localAuth); err != nil {
						s.l.Error("Unable to save AuthMethod with new password:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
						return
					}

					if err = s.login(user, common.AUTH_LOCAL, w, r); err != nil {
						s.l.Error("Unable to login to session:", err)
						s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
						return
					}

					s.l.Info("User %q has reset their password", user.Name)
					delete(s.urlKeys, key)
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			} // if POST

			s.l.Debug("Rendering password reset form")
			data := struct {
				dataPageBase
				UrlKey *common.UrlKey
				Error  string
			}{
				dataPageBase: s.newPageBase("Auth", w, r),
				UrlKey:       urlKey,
				Error:        formError,
			}

			if err := s.executeTemplate(w, "passwordReset", data); err != nil {
				s.l.Error("Error rendering template: %v", err)
			}
			return
		}
	}

renderPage:

	s.l.Debug("Rendering key form")
	data := struct {
		dataPageBase
		Url   string
		Error string
	}{
		dataPageBase: s.newPageBase("Auth", w, r),
		Url:          urlKey.Url,
		Error:        formError,
	}

	if err := s.executeTemplate(w, "auth", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// Removes the AuthType LOCAL AuthMethod from the currently logged in user
func (s *Server) handlerLocalAuthRemove(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("local remove")

	user := s.getSessionUser(w, r)

	auth, err := user.GetAuthMethod(common.AUTH_LOCAL)

	if err != nil {
		s.l.Info("User %s does not have a password associated with him", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	if len(user.AuthMethods) == 1 {
		s.l.Info("User %v only has the local Authmethod associated with him", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	user, err = s.RemoveAuthMethodFromUser(auth, user)

	if err != nil {
		s.l.Info("Could not remove password from user. %s", err.Error())
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	err = s.data.UpdateUser(user)
	if err != nil {
		s.l.Info("Could not update user %s", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	// Logging the user out to ensure that he is logged in with an existing AuthMethod
	err = s.logout(w, r)
	if err != nil {
		s.l.Info("Could not logout user %s", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	// Logging the user back in
	if _, err := user.GetAuthMethod(common.AUTH_TWITCH); err == nil {
		err = s.login(user, common.AUTH_TWITCH, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_DISCORD); err == nil {
		err = s.login(user, common.AUTH_DISCORD, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_PATREON); err == nil {
		err = s.login(user, common.AUTH_PATREON, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	}
	http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	return
}

func (s *Server) handlerTwitchOAuth(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("twitch login")

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("twitch sign up")

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("twitch add")

	case "remove":
		user := s.getSessionUser(w, r)

		auth, err := user.GetAuthMethod(common.AUTH_TWITCH)

		if err != nil {
			s.l.Info("User %s does not have Twitch Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if len(user.AuthMethods) == 1 {
			s.l.Info("User %v only has Twitch Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		user, err = s.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			s.l.Info("Could not remove Twitch Oauth from user. %s", err.Error())
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.data.UpdateUser(user)
		if err != nil {
			s.l.Info("Could not update user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Log the user out to ensure he uses an existing AuthMethod
		err = s.logout(w, r)
		if err != nil {
			s.l.Info("Could not logout user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Find a new AuthMethod to log the user back in
		if _, err := user.GetAuthMethod(common.AUTH_LOCAL); err == nil {
			err = s.login(user, common.AUTH_LOCAL, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(common.AUTH_DISCORD); err == nil {
			err = s.login(user, common.AUTH_DISCORD, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(common.AUTH_PATREON); err == nil {
			err = s.login(user, common.AUTH_PATREON, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		}

		s.l.Debug("twitch remove")

		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
}

// This function handles all Twitch Callbacks (add/signup/login)
func (s *Server) handlerTwitchOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	ok := false
	for _, expectedState := range openStates {
		if state == expectedState {
			ok = true
		}
	}
	if !ok {
		s.l.Info("Invalid/Unknown OAuth state string: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := twitchOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	// Request the User data from the API
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Client-Id", twitchOAuthConfig.ClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.l.Info("Could not retrieve Userdata from Twitch API: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return

	}

	if resp.StatusCode != 200 {
		s.l.Info("Status Code is not 200, its %v", resp.Status)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	var data map[string][]map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if strings.HasPrefix(state, "signup_") {
		// Handle the sign up process
		auth := &common.AuthMethod{
			Type:         common.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		// check if Twitch Auth is already used
		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			// Create a new user
			newUser := &common.User{
				Name:                data["data"][0]["display_name"].(string),
				Email:               data["data"][0]["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			// add this new user to the database
			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			// add the authmethod to the user
			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			// update the user in the DB with the user having the AuthMethod associated
			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			s.login(newUser, common.AUTH_TWITCH, w, r)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Debug("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		// Handle Twitch Login
		user, err := s.data.UserTwitchLogin(data["data"][0]["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_TWITCH, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		// Handle adding a Twitch AuthMethod to the logged in user

		// get the current user
		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		// check if this oauth is already used
		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			// check if the user already has an other Twitch OAuth connected
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Info("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Info("The provided Oauth login is already used")
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

func (s *Server) handlerDiscordOAuth(w http.ResponseWriter, r *http.Request) {

	action := r.URL.Query().Get("action")

	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("discord login")

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("discord signup")

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("discord add")

	case "remove":
		user := s.getSessionUser(w, r)

		auth, err := user.GetAuthMethod(common.AUTH_DISCORD)

		if err != nil {
			s.l.Info("User %s does not have Discord Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if len(user.AuthMethods) == 1 {
			s.l.Info("User %v only has Discord Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		user, err = s.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			s.l.Info("Could not remove Discord Oauth from user. %s", err.Error())
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.data.UpdateUser(user)
		if err != nil {
			s.l.Info("Could not update user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Log the user out to ensure he is logged in with an existing AuthMethod
		err = s.logout(w, r)
		if err != nil {
			s.l.Info("Could not logout user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Try to log the user back in
		if _, err := user.GetAuthMethod(common.AUTH_TWITCH); err == nil {
			err = s.login(user, common.AUTH_TWITCH, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(common.AUTH_LOCAL); err == nil {
			err = s.login(user, common.AUTH_LOCAL, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(common.AUTH_PATREON); err == nil {
			err = s.login(user, common.AUTH_PATREON, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		}

		s.l.Debug("discord remove")

		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
}

// Handler for the Discord OAuth Callbacks (add/signup/login)
func (s *Server) handlerDiscordOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	ok := false
	for _, expectedState := range openStates {
		if state == expectedState {
			ok = true
		}
	}
	if !ok {
		s.l.Info("Invalid/Unknown OAuth state string: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := discordOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	// Request the User data from the API
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.l.Info("Could not retrieve Userdata from Discord API: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if resp.StatusCode != 200 {
		s.l.Info("Status Code is not 200, its %v", resp.Status)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	var data map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if strings.HasPrefix(state, "signup_") {

		auth := &common.AuthMethod{
			Type:         common.AUTH_DISCORD,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			newUser := &common.User{
				Name:                data["username"].(string),
				Email:               data["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			s.login(newUser, common.AUTH_DISCORD, w, r)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Debug("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		user, err := s.data.UserDiscordLogin(data["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_DISCORD, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_DISCORD,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Info("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Info("The provided Oauth login is already used")
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

func (s *Server) handlerPatreonOAuth(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("patreon login")

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("patreon signup")

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("patreon add")

	case "remove":
		user := s.getSessionUser(w, r)

		auth, err := user.GetAuthMethod(common.AUTH_PATREON)

		if err != nil {
			s.l.Info("User %s does not have Patreon Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if len(user.AuthMethods) == 1 {
			s.l.Info("User %v only has Patreon Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		user, err = s.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			s.l.Info("Could not remove Patreon Oauth from user. %s", err.Error())
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.data.UpdateUser(user)
		if err != nil {
			s.l.Info("Could not update user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.logout(w, r)
		if err != nil {
			s.l.Info("Could not logout user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if _, err := user.GetAuthMethod(common.AUTH_TWITCH); err == nil {
			err = s.login(user, common.AUTH_TWITCH, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(common.AUTH_DISCORD); err == nil {
			err = s.login(user, common.AUTH_DISCORD, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(common.AUTH_LOCAL); err == nil {
			err = s.login(user, common.AUTH_LOCAL, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		}

		s.l.Debug("patreon remove")

		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
}

func (s *Server) handlerPatreonOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	ok := false
	for _, expectedState := range openStates {
		if state == expectedState {
			ok = true
		}
	}
	if !ok {
		s.l.Info("Invalid/Unknown OAuth state string: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := patreonOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	// Request the User data from the API
	req, err := http.NewRequest("GET", "https://www.patreon.com/api/oauth2/v2/identity?fields"+u.QueryEscape("[user]")+"=email,first_name,full_name,last_name,vanity", nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.l.Info("Could not retrieve Userdata from Patreon API: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return

	}
	if resp.StatusCode != 200 {
		s.l.Info("Status Code is not 200, its %v", resp.Status)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	var data map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	data = data["data"].(map[string]interface{})

	if strings.HasPrefix(state, "signup_") {

		auth := &common.AuthMethod{
			Type:         common.AUTH_PATREON,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {

			newUser := &common.User{
				Name:                data["attributes"].(map[string]interface{})["full_name"].(string),
				Email:               data["attributes"].(map[string]interface{})["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			s.login(newUser, common.AUTH_PATREON, w, r)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Info("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		user, err := s.data.UserPatreonLogin(data["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_PATREON, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_PATREON,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Info("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Info("The provided Oauth login is already used")
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}
