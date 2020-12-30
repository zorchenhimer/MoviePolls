package moviepoll

import (
	"fmt"
	"net/http"
	"regexp"
	//"strconv"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

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
					localAuth.PassDate = time.Now()

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
