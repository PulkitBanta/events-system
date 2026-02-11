package event

import (
	"context"
	"database/sql"
	"errors"
	"events-system/user"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

func (a *Accessor) CreateEvent(ctx context.Context, event Event, now time.Time) (*Event, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	id := uuid.New()

	query := `INSERT INTO events (id, title, duration_hours, user_id, slots, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
	if _, err := a.db.ExecContext(ctx, query, id, event.Title, event.DurationHours, event.UserID, SlotsColumn(event.Slots), now); err != nil {
		return nil, fmt.Errorf("exec context: %w", err)
	}

	return &Event{
		ID:            id,
		Title:         event.Title,
		DurationHours: event.DurationHours,
		UserID:        event.UserID,
		Slots:         event.Slots,
		CreatedAt:     now,
	}, nil
}

func (a *Accessor) UpdateEvent(ctx context.Context, event Event, now time.Time) (*Event, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	// Only update title, duration_hours, and slots. user_id and created_at should not be changed.
	query := `UPDATE events SET title = $1, duration_hours = $2, slots = $3 WHERE id = $4`
	if _, err := a.db.ExecContext(ctx, query, event.Title, event.DurationHours, SlotsColumn(event.Slots), event.ID); err != nil {
		return nil, fmt.Errorf("exec context: %w", err)
	}

	// Fetch the updated event to return the original created_at
	updatedEvent, err := a.GetEvent(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if updatedEvent == nil {
		return nil, fmt.Errorf("event not found after update")
	}

	return updatedEvent, nil
}

func (a *Accessor) GetEvent(ctx context.Context, id uuid.UUID) (*Event, error) {
	var event Event
	var slotsCol SlotsColumn

	query := `SELECT id, title, duration_hours, user_id, slots, created_at FROM events WHERE id = $1`
	row := a.db.QueryRowContext(ctx, query, id)
	if err := row.Scan(&event.ID, &event.Title, &event.DurationHours, &event.UserID, &slotsCol, &event.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	event.Slots = []Slot(slotsCol)

	return &event, nil
}

func (a *Accessor) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM events WHERE id = $1`
	if _, err := a.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("exec context: %w", err)
	}
	return nil
}

// GetPossibleEventSlot returns the possible event slot for the event with maximum user attendance.
// If there is no such time slot found, then it returns the time slots that work for the most number of people (also provides a list for whom it does not work).
func (a *Accessor) GetPossibleEventSlot(ctx context.Context, id uuid.UUID) (*PossibleEventSlot, error) {
	event, err := a.GetEvent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if event == nil || len(event.Slots) == 0 {
		return nil, nil
	}

	allUsers, err := a.userAccessor.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	possibleSlot := PossibleEventSlot{}

	for _, slot := range event.Slots {
		users, err := a.userAccessor.GetUsersForSlot(ctx, user.Slot{StartTime: slot.StartTime, EndTime: slot.EndTime}, event.DurationHours)
		if err != nil {
			return nil, fmt.Errorf("get users for slot: %w", err)
		}
		if len(users) >= len(possibleSlot.Users) {
			possibleSlot.Users = users
			possibleSlot.Slot = slot
			possibleSlot.NotWorkingUsers = []user.User{}
			for _, user := range allUsers {
				if !slices.Contains(users, user) {
					possibleSlot.NotWorkingUsers = append(possibleSlot.NotWorkingUsers, user)
				}
			}

			if len(possibleSlot.Users) == len(allUsers) {
				return &possibleSlot, nil
			}
		}
	}

	if len(possibleSlot.Users) == 0 {
		return nil, nil
	}

	return &possibleSlot, nil
}
