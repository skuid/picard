package picard_test

import (
	"errors"
)

// MockORM can be used to test client functionality that calls picard.ORM behavior.
type MockORM struct {
	FilterModelReturns    []interface{}
	FilterModelError      error
	FilterModelCalledWith interface{}
}

// FilterModel simply returns an error or return objects when set on the MockORM
func (morm *MockORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	morm.FilterModelCalledWith = filterModel
	if morm.FilterModelError != nil {
		return nil, morm.FilterModelError
	}
	return morm.FilterModelReturns, nil
}

// SaveModel is not implemented yet.
func (morm MockORM) SaveModel(model interface{}) error {
	return errors.New("SaveModel not actually implemented for MockORM")
}

// CreateModel is not implemented yet.
func (morm MockORM) CreateModel(model interface{}) error {
	return errors.New("CreateModel not actually implemented for MockORM")
}

// Deploy is not implemented yet.
func (morm MockORM) Deploy(data interface{}) error {
	return errors.New("Deploy not actually implemented for MockORM")
}
