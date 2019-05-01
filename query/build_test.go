package query

import (
	"testing"

	"github.com/skuid/picard/tags"
	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		desc     string
		model    interface{}
		assoc    []tags.Association
		expected *Table
	}{
		{
			"should return a single table with a few columns",
			parentModel{
				Name: "pops",
			},
			nil,
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
						Field: "organization_id",
						Val:   "00000000-0000-0000-0000-000000000001",
					},
					Where{
						Field: "name",
						Val:   "pops",
					},
				},
			},
		},
		{
			"should return a table with columns and a reference",
			field{
				Name: "a_field",
			},
			[]tags.Association{
				{
					Name: "ReferenceTo",
				},
			},
			&Table{
				Counter: 2,
				Alias:   "t0",
				Name:    "field",
				columns: []string{
					"id",
					"organization_id",
					"name",
					"object_id",
					"reference_id",
				},
				Joins: []Join{
					{
						Type:        "left",
						ParentField: "t0.reference_id",
						JoinField:   "t1.id",
						Table: &Table{
							Alias: "t1",
							Name:  "reference_to",
							columns: []string{
								"id",
								"organization_id",
							},
							Wheres: []Where{
								{
									Field: "organization_id",
									Val:   "00000000-0000-0000-0000-000000000001",
								},
							},
						},
					},
				},
				Wheres: []Where{
					Where{
						Field: "organization_id",
						Val:   "00000000-0000-0000-0000-000000000001",
					},
					Where{
						Field: "name",
						Val:   "a_field",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			actual, err := Build(orgID, tc.model, tc.assoc)

			assert.NoError(err)
			assert.Equal(tc.expected.Counter, actual.Counter)
			assert.Equal(tc.expected.Alias, actual.Alias)
			assert.Equal(tc.expected.Name, actual.Name)

			for i, actualJoin := range actual.Joins {
				expectedJoin := tc.expected.Joins[i]
				assert.Equal(expectedJoin.Type, actualJoin.Type)
				assert.Equal(expectedJoin.ParentField, actualJoin.ParentField)
				assert.Equal(expectedJoin.JoinField, actualJoin.JoinField)

				actualJTbl := *actualJoin.Table
				expectedJTbl := *expectedJoin.Table

				assert.Equal(expectedJTbl, actualJTbl)
			}

			assert.Equal(tc.expected.Wheres, actual.Wheres)

		})
	}
}
