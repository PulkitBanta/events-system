package user_test

import (
	"database/sql"
	"events-system/user"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	a := user.NewAccessor(db)

	const name = "Pulkit"
	const email = "pulkit@example.com"

	insertQuery := `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`
	mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
		WithArgs(sqlmock.AnyArg(), name, email).
		WillReturnResult(sqlmock.NewResult(1, 1))

	t.Run("create user", func(t *testing.T) {
		createdUser, err := a.CreateUser(t.Context(), user.User{
			Name:  name,
			Email: email,
		})
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, createdUser.ID)
		assert.Equal(t, name, createdUser.Name)
		assert.Equal(t, email, createdUser.Email)

		require.NoError(t, mock.ExpectationsWereMet())

		t.Run("get user", func(t *testing.T) {
			selectQuery := `SELECT id, name, email FROM users WHERE id = $1`
			rows := sqlmock.NewRows([]string{"id", "name", "email"}).
				AddRow(createdUser.ID, name, email)

			mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
				WithArgs(createdUser.ID).
				WillReturnRows(rows)

			u, err := a.GetUser(t.Context(), createdUser.ID)
			require.NoError(t, err)
			assert.Equal(t, createdUser.ID, u.ID)
			assert.Equal(t, createdUser.Name, u.Name)
			assert.Equal(t, createdUser.Email, u.Email)

			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("get user - no rows", func(t *testing.T) {
			selectQuery := `SELECT id, name, email FROM users WHERE id = $1`
			mock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
				WithArgs(uuid.New()).
				WillReturnError(sql.ErrNoRows)

			_, err := a.GetUser(t.Context(), uuid.New())
			require.Error(t, err)
		})
	})
}

func TestCreateUserSlots(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	a := user.NewAccessor(db)
	userID := uuid.New()
	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	slots := []user.Slot{
		{StartTime: startTime, EndTime: endTime},
		{StartTime: startTime.Add(24 * time.Hour), EndTime: endTime.Add(24 * time.Hour)},
	}

	t.Run("create user slots successfully", func(t *testing.T) {
		mock.ExpectBegin()

		insertQuery := `INSERT INTO users_availability (user_id, start_time, end_time) VALUES ($1, $2, $3)`
		mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
			WithArgs(userID, slots[0].StartTime, slots[0].EndTime).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
			WithArgs(userID, slots[1].StartTime, slots[1].EndTime).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		createdSlots, err := a.CreateUserSlots(t.Context(), userID, slots)
		require.NoError(t, err)
		assert.Equal(t, slots, createdSlots)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create user slots - transaction rollback on error", func(t *testing.T) {
		mock.ExpectBegin()

		insertQuery := `INSERT INTO users_availability (user_id, start_time, end_time) VALUES ($1, $2, $3)`
		mock.ExpectExec(regexp.QuoteMeta(insertQuery)).
			WithArgs(userID, slots[0].StartTime, slots[0].EndTime).
			WillReturnError(sql.ErrConnDone)

		mock.ExpectRollback()

		createdSlots, err := a.CreateUserSlots(t.Context(), userID, slots)
		require.Error(t, err)
		assert.Nil(t, createdSlots)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDeleteUserSlots(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	a := user.NewAccessor(db)
	userID := uuid.New()

	t.Run("delete user slots successfully", func(t *testing.T) {
		deleteQuery := `DELETE FROM users_availability WHERE user_id = $1`
		mock.ExpectExec(regexp.QuoteMeta(deleteQuery)).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 2)) // 2 rows deleted

		err := a.DeleteUserSlots(t.Context(), userID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete user slots - no rows to delete", func(t *testing.T) {
		deleteQuery := `DELETE FROM users_availability WHERE user_id = $1`
		mock.ExpectExec(regexp.QuoteMeta(deleteQuery)).
			WithArgs(userID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows deleted

		err := a.DeleteUserSlots(t.Context(), userID)
		require.NoError(t, err) // No error even if no rows deleted

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetUsersForSlot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	a := user.NewAccessor(db)
	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	slot := user.Slot{StartTime: startTime, EndTime: endTime}
	durationHours := 2

	user1ID := uuid.New()
	user2ID := uuid.New()
	user1 := user.User{ID: user1ID, Name: "User 1", Email: "user1@example.com"}
	user2 := user.User{ID: user2ID, Name: "User 2", Email: "user2@example.com"}

	t.Run("get users for slot successfully", func(t *testing.T) {
		// Verify the SQL query matches the implementation
		query := `SELECT users.id, users.name, users.email
	FROM users_availability
	JOIN users ON users_availability.user_id = users.id
	WHERE users_availability.start_time >= $1 AND users_availability.end_time <= $2 AND users_availability.end_time - users_availability.start_time > make_interval(hours => $3)
	ORDER BY users.name`

		rows := sqlmock.NewRows([]string{"id", "name", "email"}).
			AddRow(user1ID, user1.Name, user1.Email).
			AddRow(user2ID, user2.Name, user2.Email)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(startTime, endTime, durationHours).
			WillReturnRows(rows)

		users, err := a.GetUsersForSlot(t.Context(), slot, durationHours)
		require.NoError(t, err)
		assert.Equal(t, 2, len(users))
		assert.Equal(t, user1.ID, users[0].ID)
		assert.Equal(t, user1.Name, users[0].Name)
		assert.Equal(t, user1.Email, users[0].Email)
		assert.Equal(t, user2.ID, users[1].ID)
		assert.Equal(t, user2.Name, users[1].Name)
		assert.Equal(t, user2.Email, users[1].Email)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get users for slot - no users available", func(t *testing.T) {
		query := `SELECT users.id, users.name, users.email
	FROM users_availability
	JOIN users ON users_availability.user_id = users.id
	WHERE users_availability.start_time >= $1 AND users_availability.end_time <= $2 AND users_availability.end_time - users_availability.start_time > make_interval(hours => $3)
	ORDER BY users.name`

		rows := sqlmock.NewRows([]string{"id", "name", "email"})

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(startTime, endTime, durationHours).
			WillReturnRows(rows)

		users, err := a.GetUsersForSlot(t.Context(), slot, durationHours)
		require.NoError(t, err)
		assert.Equal(t, 0, len(users))

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get users for slot - query error", func(t *testing.T) {
		query := `SELECT users.id, users.name, users.email
	FROM users_availability
	JOIN users ON users_availability.user_id = users.id
	WHERE users_availability.start_time >= $1 AND users_availability.end_time <= $2 AND users_availability.end_time - users_availability.start_time > make_interval(hours => $3)
	ORDER BY users.name`

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(startTime, endTime, durationHours).
			WillReturnError(sql.ErrConnDone)

		users, err := a.GetUsersForSlot(t.Context(), slot, durationHours)
		require.Error(t, err)
		require.Nil(t, users)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get users for slot - scan error", func(t *testing.T) {
		query := `SELECT users.id, users.name, users.email
	FROM users_availability
	JOIN users ON users_availability.user_id = users.id
	WHERE users_availability.start_time >= $1 AND users_availability.end_time <= $2 AND users_availability.end_time - users_availability.start_time > make_interval(hours => $3)
	ORDER BY users.name`

		// Return invalid data that will cause scan error
		rows := sqlmock.NewRows([]string{"id", "name", "email"}).
			AddRow("invalid-uuid", user1.Name, user1.Email)

		mock.ExpectQuery(regexp.QuoteMeta(query)).
			WithArgs(startTime, endTime, durationHours).
			WillReturnRows(rows)

		users, err := a.GetUsersForSlot(t.Context(), slot, durationHours)
		require.Error(t, err)
		require.Nil(t, users)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
