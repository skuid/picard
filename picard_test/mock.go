package picard_test

// MockORM can be used to test client functionality that calls picard.ORM behavior.
type MockORM struct {
	FilterModelReturns    []interface{}
	FilterModelError      error
	FilterModelCalledWith interface{}
	SaveModelError        error
	SaveModelCalledWith   interface{}
	CreateModelError      error
	CreateModelCalledWith interface{}
	DeployError           error
	DeployCalledWith      interface{}
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
func (morm *MockORM) SaveModel(model interface{}) error {
	morm.SaveModelCalledWith = model
	return morm.SaveModelError
}

// CreateModel is not implemented yet.
func (morm *MockORM) CreateModel(model interface{}) error {
	morm.CreateModelCalledWith = model
	return morm.CreateModelError
}

// Deploy is not implemented yet.
func (morm *MockORM) Deploy(data interface{}) error {
	morm.DeployCalledWith = data
	return morm.DeployError
}
