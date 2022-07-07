package picard

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
)

var conn *sql.DB

type ConnectionProps struct {
	ConnString   string
	Driver       string
	ServiceName  *string
	MaxIdleConns *int
	MaxOpenConns *int
	MaxIdleTime  *int
	MaxLifeTime  *int
}

func testConnection(db *sql.DB) error {
	var err error
	if err = db.Ping(); err != nil {
		return err
	}

	if conn != nil {
		conn.Close()
	}
	conn = db
	return nil
}

// Deprecated in favor of NewConnection: CreateConnection creates a database connection using the provided arguments
func CreateConnection(connstr string) error {
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return err
	}
	return testConnection(db)
}

// Deprecated in favor of NewConnection: CreateTracedConnection creates a database connection using the provided arguments
func CreateTracedConnection(connstr, serviceName string) error {
	sqltrace.Register(
		"postgres",
		&pq.Driver{},
		sqltrace.WithServiceName(serviceName),
	)
	db, err := sqltrace.Open("postgres", connstr)
	if err != nil {
		return err
	}
	return testConnection(db)
}

// NewConnection creates a database connection using the provided arguments
func NewConnection(props ConnectionProps) error {
	if props.ServiceName != nil {
		sqltrace.Register(
			props.Driver,
			&pq.Driver{},
			sqltrace.WithServiceName(*props.ServiceName),
		)
	}

	if props.Driver == "" {
		props.Driver = "postgres"
	}

	db, err := sql.Open(props.Driver, props.ConnString)
	if err != nil {
		return err
	}

	if props.MaxIdleConns != nil {
		db.SetMaxIdleConns(*props.MaxIdleConns)
	}

	if props.MaxIdleTime != nil {
		db.SetConnMaxIdleTime(time.Duration(*props.MaxIdleTime * int(time.Second)))
	}

	if props.MaxLifeTime != nil {
		db.SetConnMaxLifetime(time.Duration(*props.MaxLifeTime * int(time.Second)))
	}

	if props.MaxOpenConns != nil {
		db.SetMaxOpenConns(*props.MaxOpenConns)
	}

	return testConnection(db)
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
