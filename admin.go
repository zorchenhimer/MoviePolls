package moviepoll

import (
	"fmt"
	"net/http"
)

type dataAdminHome struct {
	dataPageBase
}

type dataAdminConfig struct {
	dataPageBase
}

func (s *Server) handlerAdmin(w http.ResponseWriter, r *http.Request) {
	userId, ok := s.getSessionInt("userId", r)
	if ok {
		user, err := s.data.GetUser(userId)
		if err != nil {
			ok = false
			fmt.Printf("Unable to get user: %v", err)
		}

		if user.Privilege < PRIV_MOD {
			ok = false
		}
	}

	if !ok {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
			return
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	var page string
	if r.URL.Path != "/admin/" {
		_, err := fmt.Sscanf(r.URL.Path, "/admin/%s", &page)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to parse %q: %v", r.URL.Path, err),
				w, r)
			return
		}
	}

	var data interface{}
	var pageName string
	switch page {
	case "config":
		pageName = "adminConfig"
		dataCfg := dataAdminConfig{
			dataPageBase: s.newPageBase("Admin", w, r),
			Values:       []dataAdminConfigVal{},
		}
		config, err := s.data.GetConfig()
		if err != nil {
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to get config values: %v", err),
				w, r)
			return
		}

		// TODO: get rid of this type cast
		for key, val := range config.(configMap) {
			d := dataAdminConfigVal{Key: key}
			switch val.Type {
			case CVT_STRING:
				d.IsString = true
				d.StrVal = val.Value.(string)
			case CVT_INT:
				d.IsNum = true
				d.NumVal = int(val.Value.(float64))
			case CVT_BOOL:
				d.IsBool = true
				d.BoolVal = val.Value.(bool)
			default:
				fmt.Printf("Unsupported config value type for %s: %v\n", key, val)
				continue
			}

			dataCfg.Values = append(dataCfg.Values, d)
		}

		data = dataCfg

	case "":
		pageName = "adminHome"
		data = dataAdminHome{
			dataPageBase: s.newPageBase("Admin", w, r),
		}

	default:
		s.doError(http.StatusNotFound, fmt.Sprintf("%q doesn't exist", r.URL.Path), w, r)
		return
	}

	if err := s.executeTemplate(w, pageName, data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
