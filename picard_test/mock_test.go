package picard_test_test

import (
	"errors"
	"testing"

	"github.com/skuid/picard"
	"github.com/skuid/picard/picard_test"
	"github.com/stretchr/testify/assert"
)

func TestMockFilterModel(t *testing.T) {
	testCases := []struct {
		description     string
		giveFilterModel interface{}
		giveReturns     []interface{}
		giveError       error
	}{
		{
			"Should return error if present, regardless of returns set",
			nil,
			[]interface{}{
				"test 1",
				"test 2",
			},
			errors.New("Some error"),
		},
		{
			"Should return set return interfaces",
			nil,
			[]interface{}{
				"test 1",
				"test 2",
			},
			nil,
		},
		{
			"Should set FilterModelCalledWith",
			"test filter interface",
			[]interface{}{
				"test 1",
				"test 2",
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			morm := picard_test.MockORM{
				FilterModelReturns: tc.giveReturns,
				FilterModelError:   tc.giveError,
			}

			results, err := morm.FilterModel(picard.FilterRequest{
				FilterModel: tc.giveFilterModel,
			})

			if tc.giveError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.giveError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveReturns, results)
				assert.Equal(t, picard.FilterRequest{
					FilterModel: tc.giveFilterModel,
				}, morm.FilterModelCalledWith)
			}
		})
	}
}

func TestMockSaveModel(t *testing.T) {
	testCases := []struct {
		description   string
		giveSaveModel interface{}
		giveError     error
	}{
		{
			"Should return error if present, regardless of returns set",
			"test filter interface",
			errors.New("Some error"),
		},
		{
			"Should set SaveModelCalledWith",
			"test filter interface",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			morm := picard_test.MockORM{
				SaveModelError: tc.giveError,
			}
			err := morm.SaveModel(tc.giveSaveModel)

			if tc.giveError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.giveError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveSaveModel, morm.SaveModelCalledWith)
			}
		})
	}
}

func TestMockCreateModel(t *testing.T) {
	testCases := []struct {
		description     string
		giveCreateModel interface{}
		giveError       error
	}{
		{
			"Should return error if present, regardless of returns set",
			"test filter interface",
			errors.New("Some error"),
		},
		{
			"Should set CreateModelCalledWith",
			"test filter interface",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			morm := picard_test.MockORM{
				CreateModelError: tc.giveError,
			}
			err := morm.CreateModel(tc.giveCreateModel)

			if tc.giveError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.giveError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveCreateModel, morm.CreateModelCalledWith)
			}
		})
	}
}

func TestMockDeleteModel(t *testing.T) {
	testCases := []struct {
		description      string
		giveDeleteModel  interface{}
		giveRowsAffected int64
		giveError        error
	}{
		{
			"Should return rows affected & error if present",
			"test filter interface",
			1000,
			errors.New("Some error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			morm := picard_test.MockORM{
				DeleteModelRowsAffected: tc.giveRowsAffected,
				DeleteModelError:        tc.giveError,
			}
			rowsAffected, err := morm.DeleteModel(tc.giveDeleteModel)

			assert.Equal(t, tc.giveError, err)
			assert.Equal(t, tc.giveRowsAffected, rowsAffected)
			assert.Equal(t, tc.giveDeleteModel, morm.DeleteModelCalledWith)
		})
	}
}

func TestMockDeploy(t *testing.T) {
	testCases := []struct {
		description    string
		giveDeployData interface{}
		giveError      error
	}{
		{
			"Should return error if present, regardless of returns set",
			"test filter interface",
			errors.New("Some error"),
		},
		{
			"Should set DeployCalledWith",
			"test filter interface",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			morm := picard_test.MockORM{
				DeployError: tc.giveError,
			}
			err := morm.Deploy(tc.giveDeployData)

			if tc.giveError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.giveError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveDeployData, morm.DeployCalledWith)
			}
		})
	}
}

func TestMultiMockFilter(t *testing.T) {

	type simpleObject struct {
		Name string
	}

	testCases := []struct {
		description          string
		giveMultiMock        picard_test.MultiMockORM
		giveExerciseFunction func(t *testing.T, mmorm picard_test.MultiMockORM)
	}{
		{
			"Should run a series of mocks",
			picard_test.MultiMockORM{
				MockORMs: []picard_test.MockORM{
					{
						FilterModelReturns:    []interface{}{},
						FilterModelError:      nil,
						FilterModelCalledWith: picard.FilterRequest{},
					},
					{
						FilterModelReturns:    nil,
						FilterModelError:      errors.New("An Error Here"),
						FilterModelCalledWith: picard.FilterRequest{},
					},
				},
			},
			func(t *testing.T, mmorm picard_test.MultiMockORM) {
				callWith1 := simpleObject{
					Name: "Object1",
				}
				callWith2 := simpleObject{
					Name: "Object2",
				}

				result1, err := mmorm.FilterModel(picard.FilterRequest{
					FilterModel: callWith1,
				})
				assert.Equal(t, result1, mmorm.MockORMs[0].FilterModelReturns)
				assert.Equal(t, err, mmorm.MockORMs[0].FilterModelError)
				assert.Equal(t, picard.FilterRequest{
					FilterModel: callWith1,
				}, mmorm.MockORMs[0].FilterModelCalledWith)
				result2, err := mmorm.FilterModel(picard.FilterRequest{
					FilterModel: callWith2,
				})
				assert.Equal(t, result2, mmorm.MockORMs[1].FilterModelReturns)
				assert.Equal(t, err, mmorm.MockORMs[1].FilterModelError)
				assert.Equal(t, picard.FilterRequest{
					FilterModel: callWith2,
				}, mmorm.MockORMs[1].FilterModelCalledWith)
			},
		},
		{
			"Should return error if too many mocks called",
			picard_test.MultiMockORM{},
			func(t *testing.T, mmorm picard_test.MultiMockORM) {
				callWith := simpleObject{
					Name: "Object1",
				}
				result, err := mmorm.FilterModel(picard.FilterRequest{
					FilterModel: callWith,
				})
				var expectedResult []interface{}
				assert.Equal(t, result, expectedResult)
				assert.Equal(t, err, errors.New("Mock Function was called but not expected"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mmorm := tc.giveMultiMock
			tc.giveExerciseFunction(t, mmorm)
		})
	}
}
