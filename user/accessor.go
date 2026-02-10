package user

import "database/sql"

// Accessor is the DB layer entrypoint for user-related queries.
type Accessor struct {
	db *sql.DB
}

func NewAccessor(db *sql.DB) *Accessor {
	return &Accessor{db: db}
}

