package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"events-system/api"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventsAPI(t *testing.T) (*api.API, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	a := api.NewAPI(db)
	a.RegisterRoutes()
	return a, dbMock
}

func TestEventsAPI(t *testing.T) {
	t.Parallel()

	t.Run("create event", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		organizerID := uuid.New()
		startTime := time.Now().Add(24 * time.Hour)
		endTime := startTime.Add(2 * time.Hour)

		insertQuery := `INSERT INTO events (id, title, duration_hours, user_id, slots, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
		dbMock.ExpectExec(regexp.QuoteMeta(insertQuery)).
			WithArgs(sqlmock.AnyArg(), "Team Meeting", 2, organizerID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		body := map[string]any{
			"title":          "Team Meeting",
			"duration_hours": 2,
			"organizer_id":   organizerID.String(),
			"slots":          []map[string]int64{{"start_time": startTime.Unix(), "end_time": endTime.Unix()}},
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusCreated, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusCreated, res.Status)
		evt, ok := res.Response.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Team Meeting", evt["title"])
		assert.NotEmpty(t, evt["id"])
	})

	t.Run("create event invalid body", func(t *testing.T) {
		t.Parallel()
		a, _ := setupEventsAPI(t)

		req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString("invalid"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("create event validation error", func(t *testing.T) {
		t.Parallel()
		a, _ := setupEventsAPI(t)

		body := `{"title":"","duration_hours":2,"organizer_id":"00000000-0000-0000-0000-000000000000","slots":[]}`
		req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("get event", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		organizerID := uuid.New()
		now := time.Now()
		startTime := now.Add(24 * time.Hour)
		endTime := startTime.Add(2 * time.Hour)
		// Slots stored in DB as JSONB with ISO8601 strings (TIMESTAMPTZ)
		slotsJSON := []byte(`[{"start_time":"` + startTime.Format(time.RFC3339) + `","end_time":"` + endTime.Format(time.RFC3339) + `"}]`)

		selectQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(selectQuery).
			WithArgs(eventID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
				AddRow(eventID, "Team Meeting", 2, organizerID, slotsJSON, now))

		// Mock GetUser for organizer
		getUserQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(getUserQuery).
			WithArgs(organizerID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(organizerID, "Organizer", "organizer@example.com"))

		req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID.String(), nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusOK, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusOK, res.Status)
		evt, ok := res.Response.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, eventID.String(), evt["id"])
		assert.Equal(t, "Team Meeting", evt["title"])
		// Check organizer is included
		organizer, ok := evt["organizer"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, organizerID.String(), organizer["id"])
		assert.Equal(t, "Organizer", organizer["name"])
	})

	t.Run("get event not found", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		selectQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(selectQuery).
			WithArgs(eventID).
			WillReturnError(sql.ErrNoRows)

		req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID.String(), nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("get event invalid id", func(t *testing.T) {
		t.Parallel()
		a, _ := setupEventsAPI(t)

		req := httptest.NewRequest(http.MethodGet, "/api/events/not-a-uuid", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("update event", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		organizerID := uuid.New()
		now := time.Now()
		startTime := now.Add(24 * time.Hour)
		endTime := startTime.Add(2 * time.Hour)

		slotsJSON := []byte(`[{"start_time":"` + startTime.Format(time.RFC3339) + `","end_time":"` + endTime.Format(time.RFC3339) + `"}]`)

		getQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(getQuery).
			WithArgs(eventID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
				AddRow(eventID, "Old Title", 2, organizerID, slotsJSON, now))

		updateQuery := regexp.QuoteMeta(`UPDATE events SET title = $1, duration_hours = $2, slots = $3 WHERE id = $4`)
		dbMock.ExpectExec(updateQuery).
			WithArgs("Updated Title", 3, sqlmock.AnyArg(), eventID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// After update, GetEvent is called to return the updated event with original created_at
		getQueryAfterUpdate := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(getQueryAfterUpdate).
			WithArgs(eventID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
				AddRow(eventID, "Updated Title", 3, organizerID, slotsJSON, now))

		body := map[string]any{
			"title":          "Updated Title",
			"duration_hours": 3,
			"organizer_id":   organizerID.String(),
			"slots":          []map[string]int64{{"start_time": startTime.Unix(), "end_time": endTime.Unix()}},
		}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String(), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusOK, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusOK, res.Status)
		evt, ok := res.Response.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Updated Title", evt["title"])
	})

	t.Run("update event not found", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		organizerID := uuid.New()
		getQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(getQuery).
			WithArgs(eventID).
			WillReturnError(sql.ErrNoRows)

		body := `{"title":"Updated","duration_hours":2,"organizer_id":"` + organizerID.String() + `","slots":[]}`
		req := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("delete event", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		organizerID := uuid.New()
		now := time.Now()

		getQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(getQuery).
			WithArgs(eventID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
				AddRow(eventID, "Event", 2, organizerID, []byte("[]"), now))

		deleteQuery := regexp.QuoteMeta(`DELETE FROM events WHERE id = $1`)
		dbMock.ExpectExec(deleteQuery).
			WithArgs(eventID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest(http.MethodDelete, "/api/events/"+eventID.String(), nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("delete event not found", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		getQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(getQuery).
			WithArgs(eventID).
			WillReturnError(sql.ErrNoRows)

		req := httptest.NewRequest(http.MethodDelete, "/api/events/"+eventID.String(), nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("get possible event slot", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupEventsAPI(t)

		eventID := uuid.New()
		organizerID := uuid.New()
		now := time.Now()
		startTime := now.Add(24 * time.Hour)
		endTime := startTime.Add(2 * time.Hour)
		slotsJSON := []byte(`[{"start_time":"` + startTime.Format(time.RFC3339) + `","end_time":"` + endTime.Format(time.RFC3339) + `"}]`)

		getEventQuery := regexp.QuoteMeta(`SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`)
		dbMock.ExpectQuery(getEventQuery).
			WithArgs(eventID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
				AddRow(eventID, "Event", 2, organizerID, slotsJSON, now))

		getUsersQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users`)
		userID := uuid.New()
		dbMock.ExpectQuery(getUsersQuery).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(userID, "Alice", "alice@example.com"))

		getUsersForSlotQuery := `SELECT users\.id, users\.name, users\.email`
		dbMock.ExpectQuery(getUsersForSlotQuery).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(userID, "Alice", "alice@example.com"))

		req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID.String()+"/possible-slot", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusOK, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusOK, res.Status)
		possible, ok := res.Response.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, possible, "slot")
		assert.Contains(t, possible, "users")
		assert.Contains(t, possible, "not_working_users")
	})

	t.Run("get possible event slot invalid id", func(t *testing.T) {
		t.Parallel()
		a, _ := setupEventsAPI(t)

		req := httptest.NewRequest(http.MethodGet, "/api/events/bad-id/possible-slot", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
