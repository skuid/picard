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
		expected []interface{}
	}{
		{
			"should hydrate a single table with a few columns",
			field{
				Name: "pops",
			},
			map[string]FieldDescriptor{
				"t0.id": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "id",
				},
				"t0.organization_id": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "organization_id",
				},
				"t0.name": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "name",
				},
			},
			sqlmock.NewRows([]string{
				"t0.id",
				"t0.organization_id",
				"t0.name",
			}).
				AddRow(
					"00000000-0000-0000-0000-000000000002",
					orgID,
					"pops",
				),
			[]interface{}{
				field{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
				},
			},
		},
		{
			"should hydrate a set of joined tables",
			field{
				Name: "a_field",
			},
			map[string]FieldDescriptor{
				"t0.id": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "id",
				},
				"t0.organization_id": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "organization_id",
				},
				"t0.name": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "name",
				},
				"t0.object_id": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "object_id",
				},
				"t0.reference_id": FieldDescriptor{
					Alias: "t0",
					Table: "field",
					Field: "reference_id",
				},
				"t1.id": FieldDescriptor{
					Alias: "t1",
					Table: "reference_to",
					Field: "id",
				},
				"t1.organization_id": FieldDescriptor{
					Alias: "t1",
					Table: "reference_to",
					Field: "organization_id",
				},
				"t1.reference_field_id": FieldDescriptor{
					Alias: "t1",
					Table: "reference_to",
					Field: "reference_field_id",
				},
				"t2.id": FieldDescriptor{
					Alias: "t2",
					Table: "field",
					Field: "id",
				},
				"t2.organization_id": FieldDescriptor{
					Alias: "t2",
					Table: "field",
					Field: "organization_id",
				},
				"t2.name": FieldDescriptor{
					Alias: "t2",
					Table: "field",
					Field: "name",
				},
				"t2.reference_object_id": FieldDescriptor{
					Alias: "t2",
					Table: "field",
					Field: "reference_object_id",
				},
				"t3.id": FieldDescriptor{
					Alias: "t3",
					Table: "object",
					Field: "id",
				},
				"t3.organization_id": FieldDescriptor{
					Alias: "t3",
					Table: "object",
					Field: "organization_id",
				},
				"t3.name": FieldDescriptor{
					Alias: "t3",
					Table: "object",
					Field: "name",
				},
			},
			sqlmock.NewRows([]string{
				"t0.id",
				"t0.organization_id",
				"t0.name",
				"t0.object_id",
				"t0.reference_id",
				"t1.id",
				"t1.organization_id",
				"t1.reference_field_id",
				"t2.id",
				"t2.organization_id",
				"t2.name",
				"t2.reference_object_id",
				"t3.id",
				"t3.organization_id",
				"t3.name",
			}).
				AddRow(
					"00000000-0000-0000-0000-000000000002", // t0.id
					orgID,     // t0.organization_id
					"a_field", // t0.name
					"00000000-0000-0000-0000-000000000003", // t0.object_id
					"00000000-0000-0000-0000-000000000004", // t0.reference_id
					"00000000-0000-0000-0000-000000000004", // t1.id
					orgID, // t1.organization_id
					"00000000-0000-0000-0000-000000000005", // t1.reference_field_id
					"00000000-0000-0000-0000-000000000005", // t2.id
					orgID,                                  // t2.organization_id
					"a_referenced_field",                   // t2.name
					"00000000-0000-0000-0000-000000000006", // t2.reference_object_id
					"00000000-0000-0000-0000-000000000006", // t3.id
					orgID, // t3.organization_id
					"a_referenced_object", // t3.name
				),
			[]interface{}{
				field{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "a_field",
					ObjectID:       "00000000-0000-0000-0000-000000000003",
					ReferenceID:    "00000000-0000-0000-0000-000000000004",
					ReferenceTo: referenceTo{
						ID:             "00000000-0000-0000-0000-000000000004",
						OrganizationID: orgID,
						RefFieldID:     "00000000-0000-0000-0000-000000000005",
						RefField: refField{
							ID:             "00000000-0000-0000-0000-000000000005",
							OrganizationID: orgID,
							Name:           "a_referenced_field",
							RefObjectID:    "00000000-0000-0000-0000-000000000006",
							RefObject: refObject{
								ID:             "00000000-0000-0000-0000-000000000006",
								OrganizationID: orgID,
								Name:           "a_referenced_object",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases[1:2] {
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
			actuals, err := Hydrate(tc.model, tc.aliasMap, rows)
			for i, actual := range actuals {
				assert.Equal(tc.expected[i], actual.Interface().(field))
			}
		})
	}
}
