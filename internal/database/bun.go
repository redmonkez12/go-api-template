package database

import (
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// NewBunDB creates a new Bun DB instance from an existing sql.DB connection
func NewBunDB(sqlDB *sql.DB) *bun.DB {
	return bun.NewDB(sqlDB, pgdialect.New())
}
