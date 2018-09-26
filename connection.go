package picard

import (
	"database/sql"

	"github.com/lib/pq"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
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

// CreateTracedConnection creates a database connection using the provided arguments
func CreateTracedConnection(connstr string) error {
	sqltrace.Register("postgres", &pq.Driver{})
	db, err := sqltrace.Open("postgres", connstr)

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

// SetConnection allows clients to place an external database connection into picard
func SetConnection(db *sql.DB) {
	conn = db
}

// CloseConnection closes the database connection
func CloseConnection() {
	if conn != nil {
		conn.Close()
	}
}
