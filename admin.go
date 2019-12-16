package moviepoll

import (
	"fmt"
	"net/http"
	"strconv"
)

type dataAdminHome struct {
	dataPageBase
}

type dataAdminConfig struct {
	dataPageBase

	ErrorMessage []string

	MaxUserVotes           int
	EntriesRequireApproval bool
	VotingOpen             bool

	ErrMaxUserVotes bool
}

func (d dataAdminConfig) IsErrored() bool {
	return d.ErrMaxUserVotes
}

func (s *Server) checkAdminRights(w http.ResponseWriter, r *http.Request) bool {
	user := s.getSessionUser(w, r)

	ok := true
	if user == nil || user.Privilege < PRIV_MOD {
		ok = false
	}

	if !ok {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
			return false
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return false
	}

	return true
}

func (s *Server) handlerAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	data := dataAdminHome{
		dataPageBase: s.newPageBase("Admin", w, r),
	}

	if err := s.executeTemplate(w, "adminHome", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminConfig(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	data := dataAdminConfig{
		dataPageBase: s.newPageBase("Admin - Config", w, r),
		ErrorMessage: []string{},
	}
	config, err := s.data.GetConfig()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get config values: %v", err),
			w, r)
		return
	}

	if r.Method == "POST" {
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Unable to parse form: %v\n", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		maxVotesStr := r.PostFormValue("MaxUserVotes")
		maxVotes, err := strconv.ParseInt(maxVotesStr, 10, 32)
		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("MaxUserVotes invalid: %v", err))
			data.ErrMaxUserVotes = true
		} else {
			config.SetInt("MaxUserVotes", int(maxVotes))
		}

		appReqStr := r.PostFormValue("EntriesRequireApproval")
		if appReqStr != "" {
			config.SetInt("EntriesRequireApproval", int(maxVotes))
		}

		clearPass := r.PostFormValue("ClearPassSalt")
		if clearPass != "" {
			config.Delete("PassSalt")
		}

		clearCookies := r.PostFormValue("ClearCookies")
		if clearCookies != "" {
			config.Delete("SessionAuth")
			config.Delete("SessionEncrypt")
		}

		err = s.data.SaveConfig(config)
		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("Unable to save config: %v", err))
		}
	}

	data.MaxUserVotes, err = config.GetInt("MaxUserVotes")
	if err != nil {
		data.MaxUserVotes = 5 // FIXME: define defaults elsewhere
	}

	data.EntriesRequireApproval, err = config.GetBool("EntriesRequireApproval")
	if err != nil {
		data.EntriesRequireApproval = false
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
