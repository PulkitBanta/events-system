package user

import (
	"context"

	"github.com/google/uuid"
)

func (a *Accessor) CreateUser(ctx context.Context, user User) (User, error) {
	if err := user.Validate(); err != nil {
		return User{}, err
	}

	id := uuid.New()

	query := `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`
	if _, err := a.db.ExecContext(ctx, query, id, user.Name, user.Email); err != nil {
		return User{}, err
	}

	return User{
		ID:    id,
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (a *Accessor) GetUsers(ctx context.Context) ([]User, error) {
	var users []User

	query := `SELECT id, name, email FROM users`
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (a *Accessor) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	var user User

	query := `SELECT id, name, email FROM users WHERE id = $1`
	row := a.db.QueryRowContext(ctx, query, id)
	if err := row.Scan(&user.ID, &user.Name, &user.Email); err != nil {
		return User{}, err
	}

	return user, nil
}
