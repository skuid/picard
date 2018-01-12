package picard_test

import "errors"

// MockORM can be used to test client functionality that calls picard.ORM behavior.
type MockORM struct {
	FilterModelReturns []interface{}
	FilterModelError   error
}

// FilterModel simply returns an error or return objects when set on the MockORM
func (morm MockORM) FilterModel(filterModel interface{}) (interface{}, error) {
	if morm.FilterModelError != nil {
		return nil, morm.FilterModelError
	}
	return morm.FilterModelReturns, nil
}

// SaveModel simply returns an error or return objects when set on the MockORM
func (morm MockORM) SaveModel(model interface{}) error {
	return errors.New("SaveModel not actually implemented for MockORM")
}

// CreateModel simply returns an error or return objects when set on the MockORM
func (morm MockORM) CreateModel(model interface{}) error {
	return errors.New("CreateModel not actually implemented for MockORM")
}

// Deploy simply returns an error or return objects when set on the MockORM
func (morm MockORM) Deploy(data interface{}) error {
	return errors.New("Deploy not actually implemented for MockORM")
}
