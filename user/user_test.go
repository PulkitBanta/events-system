package user_test

import (
	"database/sql"
	"events-system/user"
	"regexp"
	"testing"

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
		createdUser, err := a.InsertUser(t.Context(), user.User{
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
			rows := sqlmock.NewRows([]string{"user_id", "user_name", "user_email"}).
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
