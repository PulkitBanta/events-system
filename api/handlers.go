package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type API struct {
	router *mux.Router
	db     *sql.DB
}

func NewAPI(db *sql.DB) *API {
	r := mux.NewRouter()
	r = r.PathPrefix("/api").Subrouter()
	return &API{
		router: r,
		db:     db,
	}
}

func (a *API) Handler() http.Handler {
	// Use Gorilla's built-in logging handler
	return handlers.LoggingHandler(os.Stdout, a.router)
}

type Response struct {
	Status   int `json:"status"`
	Response any `json:"response"`
}

func (a *API) Response(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(Response{
		Status:   status,
		Response: data,
	})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (a *API) RegisterRoutes() {
	a.router.HandleFunc("/health", a.health).Methods(http.MethodGet)
	a.router.HandleFunc("/users", a.createUser).Methods(http.MethodPost)
	a.router.HandleFunc("/users/{id}", a.getUser).Methods(http.MethodGet)
	a.router.HandleFunc("/users", a.getUsers).Methods(http.MethodGet)
}
