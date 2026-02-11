package event

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"events-system/user"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SlotsColumn []Slot

// Value implements driver.Valuer for INSERT/UPDATE.
func (s SlotsColumn) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner for SELECT.
func (s *SlotsColumn) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("not a []byte: %T", value)
	}
	return json.Unmarshal(b, s)
}

type Event struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	DurationHours int       `json:"duration_hours"`
	UserID        uuid.UUID `json:"user_id"`
	Slots         []Slot    `json:"slots"`
	CreatedAt     time.Time `json:"created_at"`
}

func (e *Event) Validate() error {
	if e.Title == "" {
		return errors.New("title is required")
	}
	if e.DurationHours <= 0 {
		return errors.New("duration hours must be greater than 0")
	}
	if e.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}
	for _, slot := range e.Slots {
		if err := slot.Validate(); err != nil {
			return fmt.Errorf("invalid slot - %v: %w", slot, err)
		}
	}
	return nil
}

type Slot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (s *Slot) Validate() error {
	if s.StartTime.IsZero() {
		return errors.New("start time is required")
	}
	if s.EndTime.IsZero() {
		return errors.New("end time is required")
	}
	if s.StartTime.After(s.EndTime) {
		return errors.New("start time is after end time")
	}
	return nil
}

type PossibleEventSlot struct {
	Slot            Slot        `json:"slot"`
	Users           []user.User `json:"users,omitempty"`
	NotWorkingUsers []user.User `json:"not_working_users,omitempty"`
}
