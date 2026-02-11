package api

import (
	"encoding/json"
	"events-system/event"
	"events-system/user"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type getEventsResponse struct {
	Events []event.Event `json:"events"`
}

func (a *API) getEvents(w http.ResponseWriter, r *http.Request) {
	eventAccessor := event.NewAccessor(a.db, user.NewAccessor(a.db))
	events, err := eventAccessor.GetEvents(r.Context())
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := getEventsResponse{
		Events: events,
	}
	a.Response(w, http.StatusOK, response)
}

type slot struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

// createEventRequest is the API DTO that accepts int64 epoch timestamps
type createEventRequest struct {
	Title         string `json:"title"`
	DurationHours int    `json:"duration_hours"`
	OrganizerID   string `json:"organizer_id"`
	Slots         []slot `json:"slots"`
}

func (a *API) createEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.Response(w, http.StatusBadRequest, "invalid request body")
		return
	}

	organizerID, err := uuid.Parse(req.OrganizerID)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid organizer ID")
		return
	}

	// Convert int64 epoch timestamps to time.Time
	slots := make([]event.Slot, len(req.Slots))
	for i, s := range req.Slots {
		slots[i] = event.Slot{
			StartTime: time.Unix(s.StartTime, 0).UTC(),
			EndTime:   time.Unix(s.EndTime, 0).UTC(),
		}
	}

	payload := event.Event{
		Title:         req.Title,
		DurationHours: req.DurationHours,
		UserID:        organizerID,
		Slots:         slots,
	}

	if err := payload.Validate(); err != nil {
		a.Response(w, http.StatusBadRequest, fmt.Errorf("validate: %w", err))
		return
	}

	eventAccessor := event.NewAccessor(a.db, user.NewAccessor(a.db))
	evt, err := eventAccessor.CreateEvent(r.Context(), payload, a.now)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]any{
		"id":             evt.ID.String(),
		"title":          evt.Title,
		"duration_hours": evt.DurationHours,
		"organizer_id":   evt.UserID.String(),
		"slots":          evt.Slots,
		"created_at":     evt.CreatedAt.Unix(),
	}
	a.Response(w, http.StatusCreated, response)
}

func (a *API) getEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "event ID is required")
		return
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	eventAccessor := event.NewAccessor(a.db, user.NewAccessor(a.db))
	evt, err := eventAccessor.GetEvent(r.Context(), parsedID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if evt == nil {
		a.Response(w, http.StatusNotFound, "event not found")
		return
	}

	// Fetch organizer user
	userAccessor := user.NewAccessor(a.db)
	organizer, err := userAccessor.GetUser(r.Context(), evt.UserID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if organizer == nil {
		a.Response(w, http.StatusInternalServerError, "organizer not found")
		return
	}

	response := map[string]any{
		"id":             evt.ID.String(),
		"title":          evt.Title,
		"duration_hours": evt.DurationHours,
		"organizer_id":   evt.UserID.String(),
		"organizer":      organizer,
		"slots":          evt.Slots,
		"created_at":     evt.CreatedAt.Unix(),
	}
	a.Response(w, http.StatusOK, response)
}

func (a *API) deleteEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "event ID is required")
		return
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	eventAccessor := event.NewAccessor(a.db, user.NewAccessor(a.db))

	e, err := eventAccessor.GetEvent(r.Context(), parsedID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if e == nil {
		a.Response(w, http.StatusNotFound, "event not found")
		return
	}

	err = eventAccessor.DeleteEvent(r.Context(), e.ID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.Response(w, http.StatusNoContent, nil)
}

func (a *API) updateEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "event ID is required")
		return
	}
	eventID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	eventAccessor := event.NewAccessor(a.db, user.NewAccessor(a.db))
	e, err := eventAccessor.GetEvent(r.Context(), eventID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}
	if e == nil {
		a.Response(w, http.StatusNotFound, "event not found")
		return
	}

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.Response(w, http.StatusBadRequest, "invalid request body")
		return
	}

	organizerID, err := uuid.Parse(req.OrganizerID)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid organizer ID")
		return
	}

	// Convert int64 epoch timestamps to time.Time
	slots := make([]event.Slot, len(req.Slots))
	for i, s := range req.Slots {
		slots[i] = event.Slot{
			StartTime: time.Unix(s.StartTime, 0).UTC(),
			EndTime:   time.Unix(s.EndTime, 0).UTC(),
		}
	}

	payload := event.Event{
		ID:            e.ID,
		Title:         req.Title,
		DurationHours: req.DurationHours,
		UserID:        organizerID,
		Slots:         slots,
	}

	if err := payload.Validate(); err != nil {
		a.Response(w, http.StatusBadRequest, fmt.Errorf("validate: %w", err))
		return
	}

	updatedEvent, err := eventAccessor.UpdateEvent(r.Context(), payload, a.now)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]any{
		"id":             updatedEvent.ID.String(),
		"title":          updatedEvent.Title,
		"duration_hours": updatedEvent.DurationHours,
		"organizer_id":   updatedEvent.UserID.String(),
		"slots":          updatedEvent.Slots,
		"created_at":     updatedEvent.CreatedAt.Unix(),
	}
	a.Response(w, http.StatusOK, response)
}

func (a *API) getPossibleEventSlot(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		a.Response(w, http.StatusBadRequest, "event ID is required")
		return
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		a.Response(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	eventAccessor := event.NewAccessor(a.db, user.NewAccessor(a.db))
	possibleEventSlot, err := eventAccessor.GetPossibleEventSlot(r.Context(), parsedID)
	if err != nil {
		a.Response(w, http.StatusInternalServerError, err.Error())
		return
	}

	if possibleEventSlot == nil {
		a.Response(w, http.StatusNotFound, "no possible event slot found")
		return
	}

	response := map[string]any{
		"slot":              possibleEventSlot.Slot,
		"users":             possibleEventSlot.Users,
		"not_working_users": possibleEventSlot.NotWorkingUsers,
	}
	a.Response(w, http.StatusOK, response)
}
