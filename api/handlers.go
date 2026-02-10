package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type API struct {
	router *mux.Router
	db     *sql.DB
}

func NewAPI(db *sql.DB) *API {
	r := mux.NewRouter()
	r.PathPrefix("/api")
	return &API{
		router: r,
		db:     db,
	}
}

func (a *API) Handler() http.Handler {
	return a.router
}

type Response struct {
	Status int `json:"status"`
	Data   any `json:"data"`
}

func (a *API) Response(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(Response{
		Status: status,
		Data:   data,
	})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (a *API) RegisterRoutes() {
	a.router.HandleFunc("/health", a.health).Methods(http.MethodGet)
	a.router.HandleFunc("/users", a.createUser).Methods(http.MethodPost)
}
