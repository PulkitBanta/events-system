package event

import (
	"context"
	"database/sql"
	"events-system/user"
)

type UserAccessor interface {
	GetUsers(ctx context.Context) ([]user.User, error)
	GetUsersForSlot(ctx context.Context, slot user.Slot, durationHours int) ([]user.User, error)
}

type Accessor struct {
	db           *sql.DB
	userAccessor UserAccessor
}

func NewAccessor(db *sql.DB, userAccessor UserAccessor) *Accessor {
	return &Accessor{
		db:           db,
		userAccessor: userAccessor,
	}
}
