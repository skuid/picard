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
		{
			"should return a table with columns and a chain of references",
			field{
				Name: "a_field",
			},
			[]tags.Association{
				{
					Name: "ReferenceTo",
					Associations: []tags.Association{
						{
							Name: "RefField",
							Associations: []tags.Association{
								{
									Name: "RefObject",
								},
							},
						},
					},
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
					t1.organization_id AS "t1.organization_id",
					t1.reference_field_id AS "t1.reference_field_id",
					t2.id AS "t2.id",
					t2.organization_id AS "t2.organization_id",
					t2.name AS "t2.name",
					t2.reference_object_id AS "t2.reference_object_id",
					t3.id AS "t3.id",
					t3.organization_id AS "t3.organization_id",
					t3.name AS "t3.name"
				FROM field AS t0
				LEFT JOIN reference_to AS t1 ON t1.id = t0.reference_id
				LEFT JOIN field AS t2 ON t2.id = t1.reference_field_id
				LEFT JOIN object AS t3 ON t3.id = t2.reference_object_id
				WHERE
					t0.organization_id = $1 AND
					t0.name = $2 AND
					t1.organization_id = $3 AND
					t2.organization_id = $4 AND
					t3.organization_id = $5
			`),
			[]interface{}{
				orgID,
				"a_field",
				orgID,
				orgID,
				orgID,
			},
		},
	}

	for _, tc := range testCases[len(testCases)-1:] {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			tbl, err := Build(orgID, tc.model, tc.assoc)
			assert.NoError(err)

			actual, actualArgs, err := tbl.ToSQL()
			assert.NoError(err)
			assert.Equal(tc.expected, actual)
			assert.Equal(tc.expectedArgs, actualArgs)
		})
	}
}
