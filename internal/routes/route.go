package routes

import (
	"net/http"

	"github.com/EugeneKrivoshein/14_11_2025_linkChecker/internal/handlers"

	"github.com/gorilla/mux"
)

func NewRouter(h *handlers.Handler) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", h.Handle).Methods("POST")

	return r
}
