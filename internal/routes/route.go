package routes

import (
	"14_11_2025_linkChecker/internal/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter(h *handlers.Handler) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", h.Handle).Methods("POST")

	return r
}
