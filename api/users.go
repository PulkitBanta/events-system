package api

import (
	"encoding/json"
	"events-system/user"
	"net/http"
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

	if err := payload.Validate(); err != nil {
		a.Response(w, http.StatusBadRequest, err.Error())
		return
	}

	userAccessor := user.NewAccessor(a.db)

	user, err := userAccessor.InsertUser(r.Context(), payload)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.Response(w, http.StatusCreated, user)
}
