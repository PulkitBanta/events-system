package api

import (
	"encoding/json"
	"events-system/user"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var payload user.User

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

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
		a.Response(w, http.StatusBadRequest, "user ID is required")
		return
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	userAccessor := user.NewAccessor(a.db)
	user, err := userAccessor.GetUser(r.Context(), parsedID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		a.Response(w, http.StatusNotFound, "user not found")
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
		Users: users,
	}
	a.Response(w, http.StatusOK, response)
}

func (a *API) createUserSlots(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "user ID is required")
		return
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// get user
	userAccessor := user.NewAccessor(a.db)
	u, err := userAccessor.GetUser(r.Context(), userID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if u == nil {
		a.Response(w, http.StatusNotFound, "user not found")
		return
	}

	var req []slot
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.Response(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert int64 epoch timestamps to time.Time
	slots := make([]user.Slot, len(req))
	for i, s := range req {
		slots[i] = user.Slot{
			StartTime: time.Unix(s.StartTime, 0).UTC(),
			EndTime:   time.Unix(s.EndTime, 0).UTC(),
		}
	}

	createdSlots, err := userAccessor.CreateUserSlots(r.Context(), userID, slots)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}

	a.Response(w, http.StatusCreated, createdSlots)
}

func (a *API) deleteUserSlots(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "user ID is required")
		return
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	userAccessor := user.NewAccessor(a.db)
	u, err := userAccessor.GetUser(r.Context(), userID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if u == nil {
		a.Response(w, http.StatusNotFound, "user not found")
		return
	}

	err = userAccessor.DeleteUserSlots(r.Context(), userID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.Response(w, http.StatusNoContent, nil)
}
