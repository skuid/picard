package query

import (
	"errors"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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
			testdata.FmtSQL(`
				SELECT
					t0.id AS "t0.id",
					t0.organization_id AS "t0.organization_id",
					t0.name AS "t0.name",
					t0.object_id AS "t0.object_id",
					t0.reference_id AS "t0.reference_id",
					t1.id AS "t1.id",
					t1.organization_id AS "t1.organization_id",
					t1.reference_field_id AS "t1.reference_field_id"
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
			testdata.FmtSQL(`
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

	for _, tc := range testCases {
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

func TestFindChildrenErrors(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		desc     string
		fixture  interface{}
		assoc    []tags.Association
		expected error
	}{
		{
			"should throw an error if an association is requested that doens't exist on the struct",
			testdata.ParentModel{
				ID: "1",
			},
			[]tags.Association{
				tags.Association{
					Name: "NotChildren",
				},
			},
			errors.New("The association 'NotChildren' was requested, but was not found in the struct of type: 'parentModel'"),
		},
		{
			"should throw an error if a 'child' property is missing the 'foreign_key' tag",
			noForeignKey{
				ID: "1",
			},
			[]tags.Association{
				tags.Association{
					Name: "Children",
				},
			},
			errors.New("Missing 'foreign_key' tag on child 'Children' of type 'noForeignKey'"),
		},
		{
			"should throw an error if the parent doesn't have a primary key for the child table's join",
			noPrimaryKey{},
			[]tags.Association{
				tags.Association{
					Name: "Children",
				},
			},
			errors.New("Missing 'primary_key' tag on type 'noPrimaryKey'"),
		},
		{
			"should throw an error if the child's type is not a slice or map",
			childNotSlice{},
			[]tags.Association{
				tags.Association{
					Name: "Children",
				},
			},
			errors.New("Child type for the field 'Children' on type 'childNotSlice' must be a map or slice. Found 'struct' instead"),
		},
	}

	for _, tc := range testCases[len(testCases)-1:] {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}

			mock.ExpectQuery("^SELECT")

			fixture := reflect.ValueOf(tc.fixture)

			err = FindChildren(db, orgID, &fixture, tc.assoc)
			assert.Equal(tc.expected, err)
		})
	}
}
