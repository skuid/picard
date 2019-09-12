package picard

import (
	"errors"
	"fmt"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
)

// ModelNotFoundError is returned when functions that expect to return an
// existing model cannot find one
const ModelNotFoundError Error = "Model Not Found"

// Error is a type of error that picard will return
type Error string

func (err Error) Error() string {
	return string(err)
}

func multiErrorOutputter(errs []error) string {
	errorStrings := []string{}
	for _, err := range errs {
		errorStrings = append(errorStrings, err.Error())
	}
	return strings.Join(errorStrings, " - ")
}

// SquashErrors turns a slice of errors into a single error
func SquashErrors(errs []error) error {
	var squashedError *multierror.Error
	errorMap := map[string]bool{}
	for _, err := range errs {
		errorString := err.Error()
		// Make sure our keys are unique
		if _, ok := errorMap[errorString]; !ok {
			errorMap[errorString] = true
			squashedError = multierror.Append(squashedError, err)
		}
	}
	squashedError.ErrorFormat = multiErrorOutputter
	return squashedError
}

// ForeignKeyError has extra information about which lookup failed
type ForeignKeyError struct {
	Err       error
	Table     string
	Key       string
	KeyColumn string
	FieldName string
}

/*
NewForeignKeyError returns a new ForeignKeyError object, populated with
extra information about which lookup failed
*/
func NewForeignKeyError(reason, table, key, keyColumn string, fieldName string) *ForeignKeyError {
	return &ForeignKeyError{
		Err:       errors.New(reason),
		Table:     table,
		Key:       key,
		KeyColumn: keyColumn,
		FieldName: fieldName,
	}
}

func (e *ForeignKeyError) Error() string {
	return fmt.Sprintf("%s: Table '%s', Foreign Key '%s', Key '%s'", e.Err, e.Table, e.KeyColumn, e.Key)
}

// GetFieldName returns the Related Field Name property
func (e *ForeignKeyError) GetFieldName() string {
	return e.FieldName
}

// SplitKey splits the key value into field parts
func (e *ForeignKeyError) SplitKey() []string {
	return strings.Split(e.Key, separator)
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
