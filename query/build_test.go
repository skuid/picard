package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryBuilderSimple(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		desc     string
		model    interface{}
		expected *Table
	}{
		{
			"should return a single table with a few columns",
			parentModel{
				Name: "pops",
			},
			&Table{
				Counter: 1,
				Alias:   "t0",
				Name:    "parentmodel",
				columns: []string{
					"id",
					"organization_id",
					"name",
					"parent_id",
				},
				Wheres: []Where{
					Where{
						Field: "t0.organization_id",
						Val:   "00000000-0000-0000-0000-000000000001",
					},
					Where{
						Field: "t0.name",
						Val:   "pops",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			actual, err := Build(orgID, tc.model, nil)

			assert.NoError(err)
			assert.Equal(tc.expected, actual, "Build should create the expected structure")

		})
	}
}
