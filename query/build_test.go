package query

import (
	"testing"

	"github.com/skuid/picard/tags"
	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		desc         string
		model        interface{}
		assoc        []tags.Association
		expected     string
		expectedArgs []interface{}
	}{
		{
			"should return a single table with a few columns",
			parentModel{
				Name: "pops",
			},
			nil,
			fmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.parent_id AS "t0.parent_id"
				FROM parentmodel AS t0
				WHERE t0.organization_id = $1 AND t0.name = $2
			`),
			[]interface{}{
				orgID,
				"pops",
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
			fmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.object_id AS "t0.object_id",
					t0.reference_id AS "t0.reference_id",
					t1.id AS "t1.id",
					t1.organization_id AS "t1.organization_id"
				FROM field AS t0
				LEFT JOIN reference_to AS t1 ON t1.id = t0.reference_id
				WHERE
					t0.organization_id = $1 AND
					t0.name = $2 AND t1.organization_id = $3
			`),
			[]interface{}{
				orgID,
				"a_field",
				orgID,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			tbl, err := Build(orgID, tc.model, tc.assoc)
			assert.NoError(err)

			actual, actualArgs, err := tbl.ToSQL()
			assert.NoError(err)
			assert.Equal(tc.expected, actual)
			assert.Equal(tc.expectedArgs, actualArgs)

			// assert.Equal(tc.expected.Counter, actual.Counter)
			// assert.Equal(tc.expected.Alias, actual.Alias)
			// assert.Equal(tc.expected.Name, actual.Name)

			// for i, actualJoin := range actual.Joins {
			// 	expectedJoin := tc.expected.Joins[i]
			// 	assert.Equal(expectedJoin.Type, actualJoin.Type)
			// 	assert.Equal(expectedJoin.ParentField, actualJoin.ParentField)
			// 	assert.Equal(expectedJoin.JoinField, actualJoin.JoinField)

			// 	actualJTbl := *actualJoin.Table
			// 	expectedJTbl := *expectedJoin.Table

			// 	assert.Equal(expectedJTbl, actualJTbl)
			// }

			// assert.Equal(tc.expected.Wheres, actual.Wheres)

		})
	}
}
