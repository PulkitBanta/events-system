package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"events-system/api"
	"fmt"
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

func setupUsersAPI(t *testing.T) (*api.API, sqlmock.Sqlmock) {
	t.Helper()
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	a := api.NewAPI(db)
	a.RegisterRoutes()
	return a, dbMock
}

func TestUsersAPI(t *testing.T) {
	t.Parallel()

	t.Run("create user", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		insertQuery := `INSERT INTO users \(id, name, email\) VALUES \(\$1, \$2, \$3\)`
		dbMock.ExpectExec(insertQuery).
			WithArgs(sqlmock.AnyArg(), "Alice", "alice@example.com").
			WillReturnResult(sqlmock.NewResult(1, 1))

		body := `{"name":"Alice","email":"alice@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusCreated, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusCreated, res.Status)
		created, ok := res.Response.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Alice", created["name"])
		assert.Equal(t, "alice@example.com", created["email"])
		assert.NotEmpty(t, created["id"])
	})

	t.Run("create user invalid body", func(t *testing.T) {
		t.Parallel()
		a, _ := setupUsersAPI(t)

		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("create user validation error", func(t *testing.T) {
		t.Parallel()
		a, _ := setupUsersAPI(t)

		body := `{"name":"","email":"alice@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("get user", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID := uuid.New()
		selectQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(selectQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(userID, "Bob", "bob@example.com"))

		req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID.String(), nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusOK, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusOK, res.Status)
		u, ok := res.Response.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, userID.String(), u["id"])
		assert.Equal(t, "Bob", u["name"])
		assert.Equal(t, "bob@example.com", u["email"])
	})

	t.Run("get user not found", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID := uuid.New()
		selectQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(selectQuery).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID.String(), nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("get user invalid id", func(t *testing.T) {
		t.Parallel()
		a, _ := setupUsersAPI(t)

		req := httptest.NewRequest(http.MethodGet, "/api/users/not-a-uuid", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("get users", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID1 := uuid.New()
		userID2 := uuid.New()
		selectQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users`)
		dbMock.ExpectQuery(selectQuery).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(userID1, "Alice", "alice@example.com").
				AddRow(userID2, "Bob", "bob@example.com"))

		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusOK, rec.Code)

		var res api.Response
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&res))
		assert.Equal(t, http.StatusOK, res.Status)
		respMap, ok := res.Response.(map[string]any)
		require.True(t, ok)
		users, ok := respMap["users"].([]any)
		require.True(t, ok)
		assert.Len(t, users, 2)
	})

	t.Run("create user slots", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID := uuid.New()

		getUserQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(getUserQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(userID, "Alice", "alice@example.com"))

		dbMock.ExpectBegin()
		dbMock.ExpectExec(regexp.QuoteMeta("INSERT INTO users_availability (user_id, start_time, end_time) VALUES ($1, $2, $3)")).
			WithArgs(userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		startTime := time.Now().Add(24 * time.Hour)
		endTime := startTime.Add(2 * time.Hour)
		body := `[{"start_time":` + fmt.Sprintf("%d", startTime.Unix()) + `,"end_time":` + fmt.Sprintf("%d", endTime.Unix()) + `}]`
		req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID.String()+"/slots", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("create user slots user not found", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID := uuid.New()
		getUserQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(getUserQuery).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Use epoch time (int64) for start_time and end_time
		startTime := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
		endTime := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
		body := `[{"start_time":` + fmt.Sprintf("%d", startTime.Unix()) + `,"end_time":` + fmt.Sprintf("%d", endTime.Unix()) + `}]`
		req := httptest.NewRequest(http.MethodPost, "/api/users/"+userID.String()+"/slots", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("delete user slots", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID := uuid.New()

		getUserQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(getUserQuery).
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(userID, "Alice", "alice@example.com"))

		deleteQuery := regexp.QuoteMeta(`DELETE FROM users_availability WHERE user_id = $1`)
		dbMock.ExpectExec(deleteQuery).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 2))

		req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID.String()+"/slots", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("delete user slots user not found", func(t *testing.T) {
		t.Parallel()
		a, dbMock := setupUsersAPI(t)

		userID := uuid.New()
		getUserQuery := regexp.QuoteMeta(`SELECT id, name, email FROM users WHERE id = $1`)
		dbMock.ExpectQuery(getUserQuery).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		req := httptest.NewRequest(http.MethodDelete, "/api/users/"+userID.String()+"/slots", nil)
		rec := httptest.NewRecorder()

		a.Router().ServeHTTP(rec, req)

		require.NoError(t, dbMock.ExpectationsWereMet())
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
