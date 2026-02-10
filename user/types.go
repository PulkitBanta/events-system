package user

import (
	"errors"

	"github.com/google/uuid"
)

type User struct {
	ID    uuid.UUID `json:"user_id"`
	Name  string    `json:"user_name"`
	Email string    `json:"user_email"`
}

func (u *User) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	return nil
}
