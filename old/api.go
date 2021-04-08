package moviepoll

import (
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "API isn't implemented", http.StatusNotImplemented)
}
