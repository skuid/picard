package picard_test

import (
	"errors"
)

// MockORM can be used to test client functionality that calls picard.ORM behavior.
type MockORM struct {
	FilterModelReturns      []interface{}
	FilterModelError        error
	FilterModelCalledWith   interface{}
	IncludesReturns         []interface{}
	IncludesError           error
	IncludesCalledWith      interface{}
	SaveModelError          error
	SaveModelCalledWith     interface{}
	CreateModelError        error
	CreateModelCalledWith   interface{}
	DeployError             error
	DeployCalledWith        interface{}
	DeleteModelRowsAffected int64
	DeleteModelError        error
	DeleteModelCalledWith   interface{}
}

type FilterAssociations func(interface{}, []interface{}) ([]interface{}, error)

// FilterModel simply returns an error or return objects when set on the MockORM
func (morm *MockORM) FilterModel(filterModel interface{}, filterAssocs ...FilterAssociations) ([]interface{}, error) {
	morm.FilterModelCalledWith = filterModel
	if morm.IncludesReturns != nil {
		for _, filterAssoc := range filterAssocs {
			results, err := filterAssoc(filterModel, morm.IncludesReturns)
			if err != nil {
				return nil, err
			}
			if results != nil {
				return results, nil
			}
		}
	}
	if morm.FilterModelError != nil {
		return nil, morm.FilterModelError
	}
	return morm.FilterModelReturns, nil
}

func (morm *MockORM) Includes(associations string) FilterAssociations {
	morm.IncludesCalledWith = associations
	return func(filterModel interface{}, parentResults []interface{}) ([]interface{}, error) {
		if morm.IncludesError != nil {
			return nil, morm.IncludesError
		}
		return parentResults, nil
	}
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

// MultiMockORM can be used to string together a series of calls to picard.ORM
type MultiMockORM struct {
	MockORMs []MockORM
	index    int
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
func (multi *MultiMockORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	next, err := multi.next()
	if err != nil {
		return nil, err
	}
	return next.FilterModel(filterModel)
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
