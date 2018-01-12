package picard_test_test

import (
	"errors"
	"testing"

	"github.com/skuid/picard/picard_test"
	"github.com/stretchr/testify/assert"
)

func TestMockFilterModel(t *testing.T) {
	testCases := []struct {
		description string
		giveReturns []interface{}
		giveError   error
	}{
		{
			"Should return error if present, regardless of returns set",
			[]interface{}{
				"test 1",
				"test 2",
			},
			errors.New("Some error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			morm := picard_test.MockORM{
				FilterModelReturns: tc.giveReturns,
				FilterModelError:   tc.giveError,
			}
			results, err := morm.FilterModel(nil)

			if tc.giveError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.giveError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveReturns, results)
			}
		})
	}
}
