// picard_test provides a mock orm for unit testing
package picard_test

import (
	"database/sql"
	"errors"
	"reflect"

	"github.com/skuid/picard"
)

// MockORM can be used to test client functionality that calls picard.ORM behavior.
type MockORM struct {
	FilterModelReturns       []interface{}
	FilterModelError         error
	FilterModelCalledWith    picard.FilterRequest
	SaveModelError           error
	SaveModelCalledWith      interface{}
	CreateModelError         error
	CreateModelCalledWith    interface{}
	DeployError              error
	DeployCalledWith         interface{}
	DeployMultipleError      error
	DeployMultipleCalledWith []interface{}
	DeleteModelRowsAffected  int64
	DeleteModelError         error
	DeleteModelCalledWith    interface{}
	StartTransactionReturns  *sql.Tx
	StartTransactionError    error
	CommitError              error
	RollbackError            error
}

// FilterModel simply returns an error or return objects when set on the MockORM
func (morm *MockORM) FilterModel(request picard.FilterRequest) ([]interface{}, error) {
	morm.FilterModelCalledWith = request
	if morm.FilterModelError != nil {
		return nil, morm.FilterModelError
	}
	return morm.FilterModelReturns, nil
}

// SaveModel returns the error stored in MockORM, and records the call value
func (morm *MockORM) SaveModel(model interface{}) error {
	morm.SaveModelCalledWith = model
	return morm.SaveModelError
}

// CreateModel returns the error stored in MockORM, and records the call value
func (morm *MockORM) CreateModel(model interface{}) error {
	morm.CreateModelCalledWith = model
	return morm.CreateModelError
}

// DeleteModel returns the rows affected number & error stored in MockORM, and records the call value
func (morm *MockORM) DeleteModel(data interface{}) (int64, error) {
	morm.DeleteModelCalledWith = data
	return morm.DeleteModelRowsAffected, morm.DeleteModelError
}

// Deploy returns the error stored in MockORM, and records the call value
func (morm *MockORM) Deploy(data interface{}) error {
	morm.DeployCalledWith = data
	return morm.DeployError
}

// DeployMultiple returns the error stored in MockORM, and records the call value
func (morm *MockORM) DeployMultiple(data []interface{}) error {
	morm.DeployMultipleCalledWith = data
	return morm.DeployMultipleError
}

// StartTransaction returns the error stored in MockORM and returns the value stored in the orm
func (morm *MockORM) StartTransaction() (*sql.Tx, error) {
	if morm.StartTransactionError != nil {
		return nil, morm.StartTransactionError
	}
	return morm.StartTransactionReturns, nil
}

// Commit returns the error stored in MockORM
func (morm *MockORM) Commit() error {
	if morm.CommitError != nil {
		return morm.CommitError
	}
	return nil
}

// Rollback returns the error stored in MockORM
func (morm *MockORM) Rollback() error {
	if morm.CommitError != nil {
		return morm.CommitError
	}
	return nil
}

// MultiMockORM can be used to string together a series of calls to picard.ORM
type MultiMockORM struct {
	MockORMs []MockORM
	index    int
	// If initialized, you can use TypeMap instead of the MockORMs array to return specific types of results for specific
	// requests (for example, when using goroutines to do parallel fetching of many models at once).
	TypeMap map[string]MockORM
}

// Returns the next mock in the series of mocks
func (multi *MultiMockORM) next() (*MockORM, error) {
	currentIndex := multi.index
	if len(multi.MockORMs) > currentIndex {
		multi.index = multi.index + 1
		return &multi.MockORMs[currentIndex], nil
	}
	return nil, errors.New("Mock Function was called but not expected")
}

// FilterModel simply returns an error or return objects when set on the MockORM
func (multi *MultiMockORM) FilterModel(request picard.FilterRequest) ([]interface{}, error) {
	if len(multi.TypeMap) > 0 {
		typeof := reflect.TypeOf(request.FilterModel)
		typename := typeof.Name()
		if next, ok := multi.TypeMap[typename]; ok {
			return next.FilterModel(request)
		}
	}
	next, err := multi.next()
	if err != nil {
		return nil, err
	}
	return next.FilterModel(request)
}

// SaveModel returns the error stored in MockORM, and records the call value
func (multi *MultiMockORM) SaveModel(model interface{}) error {
	next, err := multi.next()
	if err != nil {
		return err
	}
	return next.SaveModel(model)
}

// CreateModel returns the error stored in MockORM, and records the call value
func (multi *MultiMockORM) CreateModel(model interface{}) error {
	next, err := multi.next()
	if err != nil {
		return err
	}
	return next.CreateModel(model)
}

// DeleteModel returns the rows affected number & error stored in MockORM, and records the call value
func (multi *MultiMockORM) DeleteModel(data interface{}) (int64, error) {
	next, err := multi.next()
	if err != nil {
		return 0, err
	}
	return next.DeleteModel(data)
}

// Deploy returns the error stored in MockORM, and records the call value
func (multi *MultiMockORM) Deploy(data interface{}) error {
	next, err := multi.next()
	if err != nil {
		return err
	}
	return next.Deploy(data)
}

// DeployMultiple returns the error stored in MockORM, and records the call value
func (multi *MultiMockORM) DeployMultiple(data []interface{}) error {
	next, err := multi.next()
	if err != nil {
		return err
	}
	return next.DeployMultiple(data)
}

// StartTransaction returns the error stored in MockORM and returns the value stored in the orm
func (multi *MultiMockORM) StartTransaction() (*sql.Tx, error) {
	next, err := multi.next()
	if err != nil {
		return nil, err
	}
	return next.StartTransaction()
}

// Commit returns the error stored in MockORM
func (multi *MultiMockORM) Commit() error {
	next, err := multi.next()
	if err != nil {
		return nil
	}
	return next.Commit()
}

// Commit returns the error stored in MockORM
func (multi *MultiMockORM) Rollback() error {
	next, err := multi.next()
	if err != nil {
		return nil
	}
	return next.Rollback()
}
