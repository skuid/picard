package picard

import (
	"errors"
	"fmt"
)

// ModelNotFoundError is returned when functions that expect to return an
// existing model cannot find one
const ModelNotFoundError Error = "Model Not Found"

// Error is a type of error that picard will return
type Error string

func (err Error) Error() string {
	return string(err)
}

// ForeignKeyError has extra information about which lookup failed
type ForeignKeyError struct {
	Err   error
	Table string
	Key   string
}

/*
NewForeignKeyError returns a new ForeignKeyError object, populated with
extra information about which lookup failed
*/
func NewForeignKeyError(reason, table, key string) *ForeignKeyError {
	return &ForeignKeyError{
		Err:   errors.New(reason),
		Table: table,
		Key:   key,
	}
}

func (e *ForeignKeyError) Error() string {
	return fmt.Sprintf("%s: Table '%s', Foreign Key '%s'", e.Err, e.Table, e.Key)
}

// QueryError holds additional information about an SQL query failure
type QueryError struct {
	Err   error
	Query string
}

/*
NewQueryError returns a new QueryError object, populated with
extra information about which query failed
*/
func NewQueryError(err error, query string) *QueryError {
	return &QueryError{
		Err:   err,
		Query: query,
	}
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("%s: Query: %s", e.Err, e.Query)
}
