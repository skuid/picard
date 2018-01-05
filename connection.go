package picard

import (
	"database/sql"
)

var conn *sql.DB

// CreateConnection creates a database connection using the provided arguments
func CreateConnection(connstr string) error {
	db, err := sql.Open("postgres", connstr)

	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	conn = db
	return nil
}

// GetConnection gets a connection if it has already been initialized
func GetConnection() *sql.DB {
	return conn
}

// CloseConnection closes the database connection
func CloseConnection() {
	if conn != nil {
		conn.Close()
	}
}
