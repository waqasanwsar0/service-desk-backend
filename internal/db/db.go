// Package db provides the PostgreSQL connection used when DATABASE_URL is
// set. When it isn't, cmd/server falls back to the in-memory stores in
// internal/store — see main.go.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/lib/pq"
)

//go:embed schema.sql
var SchemaSQL string

// Connect opens a PostgreSQL connection pool and verifies it with a ping.
func Connect(databaseURL string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return conn, nil
}

// NextID pulls the next value from a named sequence and formats it as a
// zero-padded, prefixed ID — e.g. NextID(db, "seq_ticket", "TCK", 4) ->
// "TCK-0001". This mirrors the ID shape the in-memory stores produced.
func NextID(conn *sql.DB, sequence, prefix string, width int) (string, error) {
	var n int64
	if err := conn.QueryRow(fmt.Sprintf("SELECT nextval('%s')", sequence)).Scan(&n); err != nil {
		return "", fmt.Errorf("nextval(%s): %w", sequence, err)
	}
	return fmt.Sprintf("%s-%0*d", prefix, width, n), nil
}
