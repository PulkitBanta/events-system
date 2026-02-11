package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type API struct {
	router *mux.Router
	db     *sql.DB
	now    time.Time
}

func NewAPI(db *sql.DB) *API {
	r := mux.NewRouter()
	r = r.PathPrefix("/api").Subrouter()
	return &API{
		router: r,
		db:     db,
		now:    time.Now(),
	}
}

func (a *API) Handler() http.Handler {
	return handlers.LoggingHandler(os.Stdout, a.router)
}

func (a *API) Router() http.Handler {
	return a.router
}

type Response struct {
	Status   int `json:"status"`
	Response any `json:"response"`
}

func (a *API) Response(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := Response{
		Status: status,
	}

	if data != nil {
		response.Response = data
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
		return
	}
}

func (a *API) RegisterRoutes() {
	a.router.HandleFunc("/health", a.health).Methods(http.MethodGet)

	// users
	a.router.HandleFunc("/users", a.createUser).Methods(http.MethodPost)
	a.router.HandleFunc("/users/{id}", a.getUser).Methods(http.MethodGet)
	a.router.HandleFunc("/users", a.getUsers).Methods(http.MethodGet)
	a.router.HandleFunc("/users/{id}/slots", a.createUserSlots).Methods(http.MethodPost)
	a.router.HandleFunc("/users/{id}/slots", a.deleteUserSlots).Methods(http.MethodDelete)

	// events
	a.router.HandleFunc("/events", a.createEvent).Methods(http.MethodPost)
	a.router.HandleFunc("/events/{id}", a.getEvent).Methods(http.MethodGet)
	a.router.HandleFunc("/events/{id}", a.deleteEvent).Methods(http.MethodDelete)
	a.router.HandleFunc("/events/{id}", a.updateEvent).Methods(http.MethodPut)
	a.router.HandleFunc("/events/{id}/possible-slot", a.getPossibleEventSlot).Methods(http.MethodGet)
}
