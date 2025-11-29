package database

import (
	"database/sql"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func NewFromPostgres(url string) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(url)))

	// Configure connection pool to prevent "database closed" errors
	// SetMaxOpenConns sets the maximum number of open connections to the database
	sqldb.SetMaxOpenConns(25)
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
	sqldb.SetMaxIdleConns(5)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
	sqldb.SetConnMaxLifetime(5 * time.Minute)
	// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle
	sqldb.SetConnMaxIdleTime(10 * time.Minute)

	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}
