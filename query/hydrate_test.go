package query

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	sql "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
)

func TestHydrate(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		desc     string
		model    interface{}
		aliasMap map[string]FieldDescriptor
		rows     *sqlmock.Rows
		expected interface{}
	}{
		{
			"should return a single table with a few columns",
			parentModel{
				Name: "pops",
			},
			map[string]FieldDescriptor{
				"t0.id": FieldDescriptor{
					Table: "parentmodel",
					Field: "id",
				},
				"t0.organization_id": FieldDescriptor{
					Table: "parentmodel",
					Field: "organization_id",
				},
				"t0.name": FieldDescriptor{
					Table: "parentmodel",
					Field: "name",
				},
				"t0.parent_id": FieldDescriptor{
					Table: "parentmodel",
					Field: "parent_id",
				},
			},
			sqlmock.NewRows([]string{
				"t0.id",
				"t0.organization_id",
				"t0.name",
				"t0.parent_id",
			}).
				AddRow(
					"00000000-0000-0000-0000-000000000002",
					orgID,
					"pops",
					"00000000-0000-0000-0000-000000000003",
				),
			parentModel{
				ID:             "00000000-0000-0000-0000-000000000002",
				OrganizationID: orgID,
				Name:           "pops",
				ParentID:       "00000000-0000-0000-0000-000000000003",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			// Setting up a dummy mock so we can get our rows back properly
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectQuery("^SELECT").
				WillReturnRows(tc.rows)

			rows, err := sql.Select("foo").From("bar").RunWith(db).Query()
			if rows != nil {
				defer rows.Close()
			}

			assert.NoError(err)
			// sqlmock expectations
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations:\n%s", err)
			}

			// Testing our Hydrate function
			actual, err := Hydrate(tc.model, tc.aliasMap, rows)
			assert.Equal(tc.expected, actual[0].(parentModel))
		})
	}
}
