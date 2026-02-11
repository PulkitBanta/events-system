package event_test

import (
	"context"
	"database/sql"
	"events-system/event"
	"events-system/user"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserAccessor is a mock implementation of UserAccessor interface
type MockUserAccessor struct {
	testifymock.Mock
}

func (m *MockUserAccessor) GetUsers(ctx context.Context) ([]user.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]user.User), args.Error(1)
}

func (m *MockUserAccessor) GetUsersForSlot(ctx context.Context, slot user.Slot, durationHours int) ([]user.User, error) {
	args := m.Called(ctx, slot, durationHours)
	return args.Get(0).([]user.User), args.Error(1)
}

func TestEvent(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	userAccessor := new(MockUserAccessor)
	a := event.NewAccessor(db, userAccessor)

	eventID := uuid.New()
	organizerID := uuid.New()
	now := time.Now()
	startTime := now.Add(24 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)

	eventData := event.Event{
		Title:         "Test Event",
		DurationHours: 2,
		UserID:        organizerID,
		Slots: []event.Slot{
			{StartTime: startTime, EndTime: endTime},
		},
	}

	t.Run("create event", func(t *testing.T) {
		insertQuery := `INSERT INTO events (id, title, duration_hours, user_id, slots, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
		dbMock.ExpectExec(regexp.QuoteMeta(insertQuery)).
			WithArgs(sqlmock.AnyArg(), eventData.Title, eventData.DurationHours, eventData.UserID, event.SlotsColumn(eventData.Slots), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		createdEvent, err := a.CreateEvent(t.Context(), eventData, now)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, createdEvent.ID)
		assert.Equal(t, eventData.Title, createdEvent.Title)
		assert.Equal(t, eventData.DurationHours, createdEvent.DurationHours)
		assert.Equal(t, eventData.UserID, createdEvent.UserID)
		assert.Equal(t, eventData.Slots, createdEvent.Slots)
		assert.Equal(t, now, createdEvent.CreatedAt)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("get event", func(t *testing.T) {
		slotsJSON, _ := event.SlotsColumn(eventData.Slots).Value()
		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventID, eventData.Title, eventData.DurationHours, eventData.UserID, slotsJSON, now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		evt, err := a.GetEvent(t.Context(), eventID)
		require.NoError(t, err)
		assert.Equal(t, eventID, evt.ID)
		assert.Equal(t, eventData.Title, evt.Title)
		assert.Equal(t, eventData.DurationHours, evt.DurationHours)
		assert.Equal(t, eventData.UserID, evt.UserID)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("get event - no rows", func(t *testing.T) {
		noRowsID := uuid.New()
		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(noRowsID).
			WillReturnError(sql.ErrNoRows)

		evt, err := a.GetEvent(t.Context(), noRowsID)
		require.NoError(t, err)
		require.Nil(t, evt)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("update event", func(t *testing.T) {
		updatedEvent := event.Event{
			ID:            eventID,
			Title:         "Updated Event",
			DurationHours: 3,
			UserID:   organizerID,
			Slots: []event.Slot{
				{StartTime: startTime, EndTime: endTime},
			},
		}

		updateQuery := `UPDATE events SET title = $1, duration_hours = $2, slots = $3 WHERE id = $4`
		updatedSlotsJSON, _ := event.SlotsColumn(updatedEvent.Slots).Value()
		dbMock.ExpectExec(regexp.QuoteMeta(updateQuery)).
			WithArgs(updatedEvent.Title, updatedEvent.DurationHours, updatedSlotsJSON, updatedEvent.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// After update, GetEvent is called to return the updated event with original created_at
		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(updatedEvent.ID, updatedEvent.Title, updatedEvent.DurationHours, updatedEvent.UserID, updatedSlotsJSON, now)
		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(updatedEvent.ID).
			WillReturnRows(rows)

		result, err := a.UpdateEvent(t.Context(), updatedEvent, now)
		require.NoError(t, err)
		assert.Equal(t, updatedEvent.ID, result.ID)
		assert.Equal(t, updatedEvent.Title, result.Title)
		assert.Equal(t, updatedEvent.DurationHours, result.DurationHours)
		assert.Equal(t, now, result.CreatedAt)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("delete event", func(t *testing.T) {
		deleteQuery := `DELETE FROM events WHERE id = $1`
		dbMock.ExpectExec(regexp.QuoteMeta(deleteQuery)).
			WithArgs(eventID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := a.DeleteEvent(t.Context(), eventID)
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestGetPossibleEventSlot(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	userAccessor := new(MockUserAccessor)
	a := event.NewAccessor(db, userAccessor)

	eventID := uuid.New()
	organizerID := uuid.New()
	now := time.Now()
	startTime1 := now.Add(24 * time.Hour)
	endTime1 := startTime1.Add(2 * time.Hour)
	startTime2 := now.Add(48 * time.Hour)
	endTime2 := startTime2.Add(2 * time.Hour)

	user1 := user.User{ID: uuid.New(), Name: "User 1", Email: "user1@example.com"}
	user2 := user.User{ID: uuid.New(), Name: "User 2", Email: "user2@example.com"}
	user3 := user.User{ID: uuid.New(), Name: "User 3", Email: "user3@example.com"}

	t.Run("event not found", func(t *testing.T) {
		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnError(sql.ErrNoRows)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.NoError(t, err)
		require.Nil(t, result)

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertNotCalled(t, "GetUsers")
		userAccessor.AssertNotCalled(t, "GetUsersForSlot")
	})

	t.Run("event with no slots", func(t *testing.T) {
		eventData := event.Event{
			ID:            eventID,
			Title:         "Test Event",
			DurationHours: 2,
			UserID:   organizerID,
			Slots:         []event.Slot{},
		}

		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventData.ID, eventData.Title, eventData.DurationHours, eventData.UserID, []byte("[]"), now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.NoError(t, err)
		require.Nil(t, result)

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertNotCalled(t, "GetUsers")
		userAccessor.AssertNotCalled(t, "GetUsersForSlot")
	})

	t.Run("all users available for slot", func(t *testing.T) {
		userAccessor.ExpectedCalls = nil
		userAccessor.Calls = nil

		eventData := event.Event{
			ID:            eventID,
			Title:         "Test Event",
			DurationHours: 2,
			UserID:        organizerID,
			Slots: []event.Slot{
				{StartTime: startTime1, EndTime: endTime1},
			},
		}

		allUsers := []user.User{user1, user2, user3}
		availableUsers := []user.User{user1, user2, user3}
		slotsJSON, _ := event.SlotsColumn(eventData.Slots).Value()

		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventData.ID, eventData.Title, eventData.DurationHours, eventData.UserID, slotsJSON, now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		userAccessor.On("GetUsers", testifymock.Anything).Return(allUsers, nil)
		userAccessor.On("GetUsersForSlot", testifymock.Anything, testifymock.MatchedBy(func(s user.Slot) bool {
			return s.StartTime.Unix() == startTime1.Unix() && s.EndTime.Unix() == endTime1.Unix()
		}), 2).Return(availableUsers, nil)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Slot.StartTime.Unix() == eventData.Slots[0].StartTime.Unix() && result.Slot.EndTime.Unix() == eventData.Slots[0].EndTime.Unix())
		assert.Equal(t, 3, len(result.Users))
		assert.Equal(t, 0, len(result.NotWorkingUsers))

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertExpectations(t)
	})

	t.Run("some users available - returns slot with most users", func(t *testing.T) {
		userAccessor.ExpectedCalls = nil
		userAccessor.Calls = nil

		eventData := event.Event{
			ID:            eventID,
			Title:         "Test Event",
			DurationHours: 2,
			UserID:        organizerID,
			Slots: []event.Slot{
				{StartTime: startTime1, EndTime: endTime1},
				{StartTime: startTime2, EndTime: endTime2},
			},
		}

		allUsers := []user.User{user1, user2, user3}
		slot1Users := []user.User{user1, user2}        // 2 users
		slot2Users := []user.User{user1, user2, user3} // 3 users - should be selected
		slotsJSON, _ := event.SlotsColumn(eventData.Slots).Value()

		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventData.ID, eventData.Title, eventData.DurationHours, eventData.UserID, slotsJSON, now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		userAccessor.On("GetUsers", testifymock.Anything).Return(allUsers, nil)
		userAccessor.On("GetUsersForSlot", testifymock.Anything, testifymock.MatchedBy(func(s user.Slot) bool {
			return s.StartTime.Unix() == startTime1.Unix() && s.EndTime.Unix() == endTime1.Unix()
		}), 2).Return(slot1Users, nil)
		userAccessor.On("GetUsersForSlot", testifymock.Anything, testifymock.MatchedBy(func(s user.Slot) bool {
			return s.StartTime.Unix() == startTime2.Unix() && s.EndTime.Unix() == endTime2.Unix()
		}), 2).Return(slot2Users, nil)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Slot.StartTime.Unix() == eventData.Slots[1].StartTime.Unix() && result.Slot.EndTime.Unix() == eventData.Slots[1].EndTime.Unix()) // Second slot has more users
		assert.Equal(t, 3, len(result.Users))
		assert.Equal(t, 0, len(result.NotWorkingUsers)) // All users are available

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertExpectations(t)
	})

	t.Run("no users available for any slot", func(t *testing.T) {
		userAccessor.ExpectedCalls = nil
		userAccessor.Calls = nil

		eventData := event.Event{
			ID:            eventID,
			Title:         "Test Event",
			DurationHours: 2,
			UserID:        organizerID,
			Slots: []event.Slot{
				{StartTime: startTime1, EndTime: endTime1},
			},
		}

		allUsers := []user.User{user1, user2, user3}
		availableUsers := []user.User{} // No users available
		slotsJSON, _ := event.SlotsColumn(eventData.Slots).Value()

		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventData.ID, eventData.Title, eventData.DurationHours, eventData.UserID, slotsJSON, now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		userAccessor.On("GetUsers", testifymock.Anything).Return(allUsers, nil)
		userAccessor.On("GetUsersForSlot", testifymock.Anything, testifymock.MatchedBy(func(s user.Slot) bool {
			return s.StartTime.Unix() == startTime1.Unix() && s.EndTime.Unix() == endTime1.Unix()
		}), 2).Return(availableUsers, nil)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Slot.StartTime.Unix() == eventData.Slots[0].StartTime.Unix() && result.Slot.EndTime.Unix() == eventData.Slots[0].EndTime.Unix())
		assert.Equal(t, 0, len(result.Users))
		assert.Equal(t, 3, len(result.NotWorkingUsers)) // All users not working

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertExpectations(t)
	})

	t.Run("get users error", func(t *testing.T) {
		userAccessor.ExpectedCalls = nil
		userAccessor.Calls = nil

		eventData := event.Event{
			ID:            eventID,
			Title:         "Test Event",
			DurationHours: 2,
			UserID:   organizerID,
			Slots: []event.Slot{
				{StartTime: startTime1, EndTime: endTime1},
			},
		}
		slotsJSON, _ := event.SlotsColumn(eventData.Slots).Value()

		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventData.ID, eventData.Title, eventData.DurationHours, eventData.UserID, slotsJSON, now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		userAccessor.On("GetUsers", testifymock.Anything).Return([]user.User{}, sql.ErrConnDone)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "get users")

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertExpectations(t)
	})

	t.Run("get users for slot error", func(t *testing.T) {
		userAccessor.ExpectedCalls = nil
		userAccessor.Calls = nil

		eventData := event.Event{
			ID:            eventID,
			Title:         "Test Event",
			DurationHours: 2,
			UserID:   organizerID,
			Slots: []event.Slot{
				{StartTime: startTime1, EndTime: endTime1},
			},
		}
		slotsJSON, _ := event.SlotsColumn(eventData.Slots).Value()
		allUsers := []user.User{user1, user2}

		selectQuery := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
		rows := sqlmock.NewRows([]string{"id", "title", "duration_hours", "user_id", "slots", "created_at"}).
			AddRow(eventData.ID, eventData.Title, eventData.DurationHours, eventData.UserID, slotsJSON, now)

		dbMock.ExpectQuery(regexp.QuoteMeta(selectQuery)).
			WithArgs(eventID).
			WillReturnRows(rows)

		userAccessor.On("GetUsers", testifymock.Anything).Return(allUsers, nil)
		userAccessor.On("GetUsersForSlot", testifymock.Anything, testifymock.MatchedBy(func(s user.Slot) bool {
			return s.StartTime.Unix() == startTime1.Unix() && s.EndTime.Unix() == endTime1.Unix()
		}), 2).Return([]user.User{}, sql.ErrConnDone)

		result, err := a.GetPossibleEventSlot(t.Context(), eventID)
		require.Error(t, err)
		require.Nil(t, result)
		assert.Contains(t, err.Error(), "get users for slot")

		require.NoError(t, dbMock.ExpectationsWereMet())
		userAccessor.AssertExpectations(t)
	})
}
