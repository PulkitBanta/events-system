package api

import (
	"encoding/json"
	"events-system/user"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var payload user.User

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Println("payload", payload)

	if err := payload.Validate(); err != nil {
		a.Response(w, http.StatusBadRequest, fmt.Errorf("validate: %w", err))
		return
	}

	userAccessor := user.NewAccessor(a.db)

	user, err := userAccessor.CreateUser(r.Context(), payload)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.Response(w, http.StatusCreated, user)
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "User ID is required")
		return
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	userAccessor := user.NewAccessor(a.db)
	user, err := userAccessor.GetUser(r.Context(), parsedID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.Response(w, http.StatusOK, user)
}

type getUsersResponse struct {
	Users []user.User `json:"users"`
}

func (a *API) getUsers(w http.ResponseWriter, r *http.Request) {
	userAccessor := user.NewAccessor(a.db)
	users, err := userAccessor.GetUsers(r.Context())
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := getUsersResponse{
		Users: []user.User{},
	}
	if len(users) > 0 {
		response.Users = users
	}

	a.Response(w, http.StatusOK, response)
}
