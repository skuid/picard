package picard_test_test

import (
	"errors"
	"testing"

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
			results, err := morm.FilterModel(tc.giveFilterModel)

			if tc.giveError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.giveError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveReturns, results)
				assert.Equal(t, tc.giveFilterModel, morm.FilterModelCalledWith)
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
			"Should set FilterModelCalledWith",
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
