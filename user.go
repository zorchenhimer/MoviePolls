package moviepoll

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

func (s *Server) handlerUser(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	totalVotes, err := s.data.GetCfgInt("MaxUserVotes")
	if err != nil {
		fmt.Printf("Error getting MaxUserVotes config setting: %v\n", err)
		totalVotes = 5 // FIXME: define a default somewhere?
	}

	data := dataAccount{
		dataPageBase: s.newPageBase("Account", w, r),

		CurrentVotes: s.data.GetUserVotes(user.Id),
		TotalVotes:   totalVotes,
	}
	data.AvailableVotes = totalVotes - len(data.CurrentVotes)

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			fmt.Printf("ParseForm() error: %v\n", err)
			s.doError(http.StatusInternalServerError, "Form error", w, r)
			return
		}

		formVal := r.PostFormValue("Form")
		if formVal == "ChangePassword" {
			// Do password stuff
			currentPass := s.hashPassword(r.PostFormValue("PasswordCurrent"))
			newPass1_raw := r.PostFormValue("PasswordNew1")
			newPass2_raw := r.PostFormValue("PasswordNew2")

			if currentPass != user.Password {
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

			if !data.IsErrored() {
				// Change pass
				data.SuccessMessage = "Password successfully changed"
				user.Password = s.hashPassword(newPass1_raw)
				user.PassDate = time.Now()

				fmt.Printf("new PassDate: %s\n", user.PassDate)

				err = s.login(user, w, r)
				if err != nil {
					fmt.Println("Unable to login to session: %v", err)
					s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
					return
				}

				if err = s.data.UpdateUser(user); err != nil {
					fmt.Println("Unable to save User with new password:", err)
					s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
					return
				}
			}

		} else if formVal == "Notifications" {
			// Update notifications
		}
	}

	if err := s.executeTemplate(w, "account", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
func (s *Server) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Error parsing login form: %v\n", err)
	}

	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/user", http.StatusFound)
		return
	}

	data := dataLoginForm{}
	doRedirect := false

	if r.Method == "POST" {
		// do login

		un := r.PostFormValue("Username")
		pw := r.PostFormValue("Password")
		user, err = s.data.UserLogin(un, s.hashPassword(pw))
		if err != nil {
			data.ErrorMessage = err.Error()
		} else {
			doRedirect = true
		}

	} else {
		fmt.Printf("> no post: %s\n", r.Method)
	}

	if user != nil {
		err = s.login(user, w, r)
		if err != nil {
			fmt.Printf("Unable to login: %v", err)
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
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerUserLogout(w http.ResponseWriter, r *http.Request) {
	err := s.logout(w, r)
	if err != nil {
		fmt.Printf("Error logging out: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handlerUserNew(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}

	data := dataNewAccount{
		dataPageBase: s.newPageBase("Create Account", w, r),
	}

	doRedirect := false

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			fmt.Printf("Error parsing login form: %v\n", err)
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		}

		un := strings.TrimSpace(r.PostFormValue("Username"))
		// TODO: password requirements
		pw1 := r.PostFormValue("Password1")
		pw2 := r.PostFormValue("Password2")

		data.ValName = un

		if un == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Username cannot be blank!")
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

		newUser := &common.User{
			Name:                un,
			Password:            s.hashPassword(pw1),
			Email:               email,
			NotifyCycleEnd:      data.ValNotifyEnd,
			NotifyVoteSelection: data.ValNotifySelected,
			PassDate:            time.Now(),
		}

		_, err = s.data.AddUser(newUser)
		if err != nil {
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		} else {
			err = s.login(newUser, w, r)
			if err != nil {
				fmt.Printf("Unable to login to session: %v\n", err)
				s.doError(http.StatusInternalServerError, "Login error", w, r)
				return
			}
			doRedirect = true
		}
	}

	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := s.executeTemplate(w, "newaccount", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
