package query

import (
	"encoding/base64"
	"testing"

	"github.com/skuid/picard/crypto"
	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/tags"

	"github.com/DATA-DOG/go-sqlmock"
	sql "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
)

func TestHydrate(t *testing.T) {
	unencryptedSecret := []byte("This is a secret!")
	cryptoKey := []byte("the-key-has-to-be-32-bytes-long!")
	crypto.SetEncryptionKey(cryptoKey)
	encryptedSecret, _ := crypto.EncryptBytes(unencryptedSecret)
	hashedSecret := base64.StdEncoding.EncodeToString(encryptedSecret)

	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		desc     string
		model    interface{}
		tblAlias string
		aliasMap map[string]qp.FieldDescriptor
		rows     *sqlmock.Rows
		expected []interface{}
	}{
		{
			"should hydrate a single table with a few columns",
			field{
				Name: "pops",
			},
			"t0",
			map[string]qp.FieldDescriptor{
				"t0.id": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "id",
				},
				"t0.organization_id": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "organization_id",
				},
				"t0.name": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "name",
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
			"should hydrate a table with nulls in the results",
			field{
				Name: "pops",
			},
			"t0",
			map[string]qp.FieldDescriptor{
				"t0.id": {
					Alias:  "t0",
					Table:  "field",
					Column: "id",
				},
				"t0.organization_id": {
					Alias:  "t0",
					Table:  "field",
					Column: "organization_id",
				},
				"t0.name": {
					Alias:  "t0",
					Table:  "field",
					Column: "name",
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
				).
				AddRow(
					"00000000-0000-0000-0000-000000000003",
					orgID,
					nil,
				),
			[]interface{}{
				field{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
				},
				field{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
				},
			},
		},
		{
			"should hydrate a table with encrypted fields",
			field{
				Name: "pops",
			},
			"t0",
			map[string]qp.FieldDescriptor{
				"t0.id": {
					Alias:  "t0",
					Table:  "field",
					Column: "id",
				},
				"t0.organization_id": {
					Alias:  "t0",
					Table:  "field",
					Column: "organization_id",
				},
				"t0.name": {
					Alias:  "t0",
					Table:  "field",
					Column: "name",
				},
				"t0.secret": {
					Alias:  "t0",
					Table:  "field",
					Column: "secret",
				},
			},
			sqlmock.NewRows([]string{
				"t0.id",
				"t0.organization_id",
				"t0.name",
				"t0.secret",
			}).
				AddRow(
					"00000000-0000-0000-0000-000000000002",
					orgID,
					"pops",
					hashedSecret,
				),
			[]interface{}{
				field{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					Secret:         string(unencryptedSecret),
				},
			},
		},
		{
			"should hydrate a set of joined tables",
			field{
				Name: "a_field",
			},
			"t0",
			map[string]qp.FieldDescriptor{
				"t0.id": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "id",
				},
				"t0.organization_id": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "organization_id",
				},
				"t0.name": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "name",
				},
				"t0.object_id": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "object_id",
				},
				"t0.reference_id": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "field",
					Column: "reference_id",
				},
				"t1.id": qp.FieldDescriptor{
					Alias:   "t1",
					Table:   "reference_to",
					Column:  "id",
					RefPath: "ReferenceID",
				},
				"t1.organization_id": qp.FieldDescriptor{
					Alias:   "t1",
					Table:   "reference_to",
					Column:  "organization_id",
					RefPath: "ReferenceID",
				},
				"t1.reference_field_id": qp.FieldDescriptor{
					Alias:   "t1",
					Table:   "reference_to",
					Column:  "reference_field_id",
					RefPath: "ReferenceID",
				},
				"t2.id": qp.FieldDescriptor{
					Alias:   "t2",
					Table:   "field",
					Column:  "id",
					RefPath: "ReferenceID.RefFieldID",
				},
				"t2.organization_id": qp.FieldDescriptor{
					Alias:   "t2",
					Table:   "field",
					Column:  "organization_id",
					RefPath: "ReferenceID.RefFieldID",
				},
				"t2.name": qp.FieldDescriptor{
					Alias:   "t2",
					Table:   "field",
					Column:  "name",
					RefPath: "ReferenceID.RefFieldID",
				},
				"t2.reference_object_id": qp.FieldDescriptor{
					Alias:   "t2",
					Table:   "field",
					Column:  "reference_object_id",
					RefPath: "ReferenceID.RefFieldID",
				},
				"t3.id": qp.FieldDescriptor{
					Alias:   "t3",
					Table:   "object",
					Column:  "id",
					RefPath: "ReferenceID.RefFieldID.RefObjectID",
				},
				"t3.organization_id": qp.FieldDescriptor{
					Alias:   "t3",
					Table:   "object",
					Column:  "organization_id",
					RefPath: "ReferenceID.RefFieldID.RefObjectID",
				},
				"t3.name": qp.FieldDescriptor{
					Alias:   "t3",
					Table:   "object",
					Column:  "name",
					RefPath: "ReferenceID.RefFieldID.RefObjectID",
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
					orgID,                                  // t0.organization_id
					"a_field",                              // t0.name
					"00000000-0000-0000-0000-000000000003", // t0.object_id
					"00000000-0000-0000-0000-000000000004", // t0.reference_id
					"00000000-0000-0000-0000-000000000004", // t1.id
					orgID,                                  // t1.organization_id
					"00000000-0000-0000-0000-000000000005", // t1.reference_field_id
					"00000000-0000-0000-0000-000000000005", // t2.id
					orgID,                                  // t2.organization_id
					"a_referenced_field",                   // t2.name
					"00000000-0000-0000-0000-000000000006", // t2.reference_object_id
					"00000000-0000-0000-0000-000000000006", // t3.id
					orgID,                                  // t3.organization_id
					"a_referenced_object",                  // t3.name
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

			metadata, err := tags.GetTableMetadata(tc.model)
			if err != nil {
				t.Fatal(err)
			}

			// Testing our Hydrate function
			actuals, err := Hydrate(tc.model, tc.tblAlias, tc.aliasMap, rows, metadata)
			assert.NoError(err)
			for i, actual := range actuals {
				assert.Equal(tc.expected[i], actual.Interface().(field))
			}
		})
	}
}
