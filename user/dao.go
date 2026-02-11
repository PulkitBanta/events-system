package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
)

func (a *Accessor) CreateUser(ctx context.Context, user User) (*User, error) {
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	id := uuid.New()

	query := `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`
	if _, err := a.db.ExecContext(ctx, query, id, user.Name, user.Email); err != nil {
		return nil, fmt.Errorf("exec context: %w", err)
	}

	return &User{
		ID:    id,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (a *Accessor) GetUsers(ctx context.Context) ([]User, error) {
	query := `SELECT id, name, email FROM users`
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return users, nil
}

func (a *Accessor) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `SELECT id, name, email FROM users WHERE id = $1`
	row := a.db.QueryRowContext(ctx, query, id)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan: %w", err)
	}

	return &user, nil
}

// GetUserSlots returns the user's availability slots.
func (a *Accessor) GetUserSlots(ctx context.Context, userID uuid.UUID) ([]Slot, error) {
	query := `SELECT start_time, end_time FROM users_availability WHERE user_id = $1`
	rows, err := a.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	slots := []Slot{}
	for rows.Next() {
		var slot Slot
		if err := rows.Scan(&slot.StartTime, &slot.EndTime); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		slots = append(slots, slot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return slots, nil
}

// CreateUserSlots creates the user's availability slots.
func (a *Accessor) CreateUserSlots(ctx context.Context, userID uuid.UUID, slots []Slot) ([]Slot, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		err := tx.Rollback()
		if err != nil {
			log.Printf("rollback tx: %v", err)
		}
	}()

	for _, slot := range slots {
		query := `INSERT INTO users_availability (user_id, start_time, end_time) VALUES ($1, $2, $3)`
		if _, err := tx.ExecContext(ctx, query, userID, slot.StartTime, slot.EndTime); err != nil {
			return nil, fmt.Errorf("exec context: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return slots, nil
}

// DeleteUserSlots deletes the user's availability slots.
func (a *Accessor) DeleteUserSlots(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM users_availability WHERE user_id = $1`
	if _, err := a.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("exec context: %w", err)
	}
	return nil
}

// GetUsersForSlot returns the users that are available for the given slot and duration hours.
func (a *Accessor) GetUsersForSlot(ctx context.Context, slot Slot, durationHours int) ([]User, error) {
	query := `SELECT users.id, users.name, users.email
	FROM users_availability
	JOIN users ON users_availability.user_id = users.id
	WHERE users_availability.start_time <= $1 AND users_availability.end_time >= $2 AND users_availability.end_time - users_availability.start_time >= make_interval(hours => $3)
	ORDER BY users.name`
	rows, err := a.db.QueryContext(ctx, query, slot.StartTime, slot.EndTime, durationHours)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return users, nil
}
