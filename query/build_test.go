package query

import (
	"testing"

	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/tags"
	"github.com/skuid/picard/testdata"
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
			testdata.ParentModel{
				Name: "pops",
			},
			nil,
			testdata.FmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.parent_id AS "t0.parent_id",
					t0.other_parent_id AS "t0.other_parent_id"
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
			testdata.FmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.secret AS "t0.secret",
					t0.object_id AS "t0.object_id",
					t0.reference_id AS "t0.reference_id",
					t1.id AS "t1.id",
					t1.organization_id AS "t1.organization_id",
					t1.reference_field_id AS "t1.reference_field_id"
				FROM field AS t0
				LEFT JOIN reference_to AS t1 ON
					(t1.id = t0.reference_id AND t1.organization_id = $1)
				WHERE
					t0.organization_id = $2 AND
					t0.name = $3
			`),
			[]interface{}{
				orgID,
				orgID,
				"a_field",
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
			testdata.FmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.secret AS "t0.secret",
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
				LEFT JOIN reference_to AS t1 ON
					(t1.id = t0.reference_id AND t1.organization_id = $1)
				LEFT JOIN field AS t2 ON
					(t2.id = t1.reference_field_id AND t2.organization_id = $2)
				LEFT JOIN object AS t3 ON
					(t3.id = t2.reference_object_id AND t3.organization_id = $3)
				WHERE
					t0.organization_id = $4 AND
					t0.name = $5
			`),
			[]interface{}{
				orgID,
				orgID,
				orgID,
				orgID,
				"a_field",
			},
		},
		{
			"should join any referenced tables without the selected fields if it is not asked for in associations",
			testdata.ChildModel{
				Parent: testdata.ParentModel{
					Name: "my parent",
				},
			},
			nil,
			testdata.FmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.parent_id AS "t0.parent_id",
					t1.id AS "t1.id",
					t1.name AS "t1.name"
				FROM childmodel AS t0
				JOIN parentmodel AS t1 ON
					(t1.id = t0.parent_id AND t1.organization_id = $1)
				WHERE
					t0.organization_id = $2 AND
					t1.name = $3
			`),
			[]interface{}{
				orgID,
				orgID,
				"my parent",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			metadata, err := tags.GetTableMetadata(tc.model)
			if err != nil {
				t.Fatal(err)
			}

			tbl, err := Build(orgID, tc.model, nil, tc.assoc, nil, metadata)
			assert.NoError(err)

			actual, actualArgs, err := tbl.ToSQL()
			assert.NoError(err)
			assert.Equal(tc.expected, actual)
			assert.Equal(tc.expectedArgs, actualArgs)
		})
	}
}

type noForeignKey struct {
	Metadata       metadata.Metadata     `picard:"tablename=parentmodel"`
	ID             string                `json:"id" picard:"primary_key,column=id"`
	OrganizationID string                `picard:"multitenancy_key,column=organization_id"`
	Children       []testdata.ChildModel `json:"children" picard:"child"`
}

type noPrimaryKey struct {
	Metadata       metadata.Metadata     `picard:"tablename=parentmodel"`
	OrganizationID string                `picard:"multitenancy_key,column=organization_id"`
	Children       []testdata.ChildModel `json:"children" picard:"child,foreign_key=parent_id"`
}

type childNotSlice struct {
	Metadata       metadata.Metadata   `picard:"tablename=parentmodel"`
	ID             string              `json:"id" picard:"primary_key,column=id"`
	OrganizationID string              `picard:"multitenancy_key,column=organization_id"`
	Children       testdata.ChildModel `json:"children" picard:"child,foreign_key=parent_id"`
}
