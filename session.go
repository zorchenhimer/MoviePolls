package moviepoll

import (
	"fmt"
	"net/http"
)

func (s *Server) getSessionBool(key string, r *http.Request) bool {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		fmt.Printf("[getbool] Unable to get session from store: %v\n", err)
		return false
	}

	val := session.Values[key]
	var boolVal bool
	var ok bool
	if boolVal, ok = val.(bool); !ok {
		boolVal = false
	}

	return boolVal
}

func (s *Server) getSessionInt(key string, r *http.Request) (int, bool) {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		fmt.Printf("[getint] Unable to get session from store: %v\n", err)
		return 0, false
	}

	val := session.Values[key]
	var intVal int
	var ok bool
	if intVal, ok = val.(int); !ok {
		return 0, false
	}

	return intVal, true
}

func (s *Server) setSessionValue(key string, val interface{}, w http.ResponseWriter, r *http.Request) {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		fmt.Printf("[set] Unable to get session from store: %v\n", err)
	}
	session.Values[key] = val

	err = session.Save(r, w)
	if err != nil {
		fmt.Printf("Unable to save cookie: %v\n", err)
	}
}

func (s *Server) deleteSessionValue(key string, w http.ResponseWriter, r *http.Request) {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		fmt.Printf("[del] Unable to get session from store: %v\n", err)
	}

	delete(session.Values, key)

	err = session.Save(r, w)
	if err != nil {
		fmt.Printf("Unable to save cookie: %v\n", err)
	}
}
