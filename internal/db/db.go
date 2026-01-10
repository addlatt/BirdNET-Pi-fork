package db

import (
	"database/sql"
	"fmt"

	generated "github.com/birdnet-pi/birdnet/internal/db/generated"
	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sqlc queries and provides the database connection.
type DB struct {
	conn    *sql.DB
	Queries *generated.Queries
}

// New creates a new database connection with read-only access.
func New(dbPath string) (*DB, error) {
	// Open in read-only mode with shared cache for better concurrency
	connStr := fmt.Sprintf("file:%s?mode=ro&cache=shared&_journal_mode=WAL", dbPath)
	conn, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings for read-only access
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	return &DB{
		conn:    conn,
		Queries: generated.New(conn),
	}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying database connection.
func (db *DB) Conn() *sql.DB {
	return db.conn
}
