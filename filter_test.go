package picard

import (
	"crypto/rand"
	"reflect"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/MakeNowJust/heredoc"
	"github.com/Masterminds/squirrel"
	"github.com/skuid/picard/crypto"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/query"
	"github.com/skuid/picard/tags"
	"github.com/stretchr/testify/assert"
)

type modelMutitenantPKWithTwoFields struct {
	Metadata              metadata.Metadata `picard:"tablename=test_table"`
	TestMultitenancyField string            `picard:"multitenancy_key,column=test_multitenancy_column"`
	TestPrimaryKeyField   string            `picard:"primary_key,column=primary_key_column"`
	TestFieldOne          string            `picard:"column=test_column_one"`
	TestFieldTwo          string            `picard:"column=test_column_two"`
}

type modelOneField struct {
	Metadata     metadata.Metadata `picard:"tablename=test_table"`
	TestFieldOne string            `picard:"column=test_column_one"`
}

type modelOneFieldEncrypted struct {
	Metadata     metadata.Metadata `picard:"tablename=test_table"`
	TestFieldOne string            `picard:"encrypted,column=test_column_one"`
}

type modelTwoFieldEncrypted struct {
	Metadata     metadata.Metadata `picard:"tablename=test_table"`
	TestFieldOne string            `picard:"encrypted,column=test_column_one"`
	TestFieldTwo string            `picard:"encrypted,column=test_column_two"`
}

type modelOneFieldJSONB struct {
	Metadata     metadata.Metadata    `picard:"tablename=test_table"`
	TestFieldOne TestSerializedObject `picard:"jsonb,column=test_column_one"`
}

type modelOnePointerFieldJSONB struct {
	Metadata     metadata.Metadata     `picard:"tablename=test_table"`
	TestFieldOne *TestSerializedObject `picard:"jsonb,column=test_column_one"`
}

type modelOneArrayFieldJSONB struct {
	Metadata     metadata.Metadata      `picard:"tablename=test_table"`
	TestFieldOne []TestSerializedObject `picard:"jsonb,column=test_column_one"`
}

type modelTwoField struct {
	TestFieldOne string `picard:"column=test_column_one"`
	TestFieldTwo string `picard:"column=test_column_two"`
}

type modelTwoFieldOneTagged struct {
	TestFieldOne string `picard:"column=test_column_one"`
	TestFieldTwo string
}

type modelMultitenant struct {
	TestMultitenancyField string `picard:"multitenancy_key,column=test_multitenancy_column"`
}

type modelPK struct {
	PrimaryKeyField string `picard:"primary_key,column=primary_key_column"`
}

// These all explore relationships where a child table has an FK to the parent
// as a 1:M relationship
type vGrandParentModel struct {
	Metadata       metadata.Metadata `picard:"tablename=grandparentmodel"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	Age            int               `json:"age" picard:"lookup,column=age"`
	Toys           []vToyModel       `json:"toys" picard:"child,foreign_key=ParentID"`
	Children       []vParentModel    `json:"children" picard:"child,foreign_key=ParentID"`
	Animals        []vPetModel       `json:"animals" picard:"child,foreign_key=ParentID"`
}

type vParentModel struct {
	Metadata       metadata.Metadata      `picard:"tablename=parentmodel"`
	ID             string                 `json:"id" picard:"primary_key,column=id"`
	OrganizationID string                 `picard:"multitenancy_key,column=organization_id"`
	Name           string                 `json:"name" picard:"lookup,column=name"`
	ParentID       string                 `picard:"foreign_key,lookup,required,related=GrandParent,column=parent_id"`
	GrandParent    vGrandParentModel      `json:"parent" validate:"-"`
	Children       []vChildModel          `json:"children" picard:"child,foreign_key=ParentID"`
	Animals        []vPetModel            `json:"animals" picard:"child,foreign_key=ParentID"`
	ChildrenMap    map[string]vChildModel `picard:"child,foreign_key=ParentID,key_mapping=Name"`
}

type vChildModel struct {
	Metadata metadata.Metadata `picard:"tablename=childmodel"`

	ID             string       `json:"id" picard:"primary_key,column=id"`
	OrganizationID string       `picard:"multitenancy_key,column=organization_id"`
	Name           string       `json:"name" picard:"lookup,column=name"`
	ParentID       string       `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         vParentModel `json:"parent" validate:"-"`
	Toys           []vToyModel  `json:"children" picard:"child,foreign_key=ParentID"`
}

type vToyModel struct {
	Metadata metadata.Metadata `picard:"tablename=toymodel"`

	ID             string      `json:"id" picard:"primary_key,column=id"`
	OrganizationID string      `picard:"multitenancy_key,column=organization_id"`
	Name           string      `json:"name" picard:"lookup,column=name"`
	ParentID       string      `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         vChildModel `json:"parent" validate:"-"`
}

type vPetModel struct {
	Metadata metadata.Metadata `picard:"tablename=petmodel"`

	ID             string       `json:"id" picard:"primary_key,column=id"`
	OrganizationID string       `picard:"multitenancy_key,column=organization_id"`
	Name           string       `json:"name" picard:"lookup,column=name"`
	ParentID       string       `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         vParentModel `json:"parent" validate:"-"`
}

func TestFilterModelAssociations(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		description          string
		filterModel          interface{}
		associations         []tags.Association
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErr              error
	}{
		{
			"happy path for single parent filter w/o eager loading",
			vParentModel{
				Name: "pops",
			},
			nil,
			[]interface{}{
				vParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000003",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.parent_id AS "t0.parent_id"
						FROM parentmodel AS t0
						WHERE t0.organization_id = $1 AND t0.name = $2
					`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
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
					)
			},
			nil,
		},
		{
			"happy path for multiple parent filter w/o eager loading",
			vParentModel{},
			nil,
			[]interface{}{
				vParentModel{
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000004",
				},
				vParentModel{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
					Name:           "uncle",
					ParentID:       "00000000-0000-0000-0000-000000000004",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM parentmodel AS t0
					WHERE t0.organization_id = $1
				`)).
					WithArgs(orgID).
					WillReturnRows(
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
								"00000000-0000-0000-0000-000000000004",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000003",
								orgID,
								"uncle",
								"00000000-0000-0000-0000-000000000004",
							),
					)
			},
			nil,
		},
		{
			"happy path for single parent filter with eager loading parent",
			vParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "GrandParent",
				},
			},
			[]interface{}{
				vParentModel{
					Name:     "pops",
					ID:       "00000000-0000-0000-0000-000000000002",
					ParentID: "00000000-0000-0000-0000-000000000023",
					GrandParent: vGrandParentModel{
						Name: "grandpops",
						ID:   "00000000-0000-0000-0000-000000000023",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM parentmodel AS t0
					WHERE t0.organization_id = $1 AND t0.name = $2
				`)).
					WithArgs(orgID, "pops").
					WillReturnRows(
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
								"00000000-0000-0000-0000-000000000023",
							),
					)

				mock.ExpectQuery(query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.age AS "t0.age"
					FROM grandparentmodel AS t0
					WHERE t0.organization_id = $1 AND 
						t0.id IN ($2)
				`)).
					WithArgs(orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id"}).
							AddRow("grandpops", "00000000-0000-0000-0000-000000000023"),
					)
			},
			nil,
		},
		{
			"happy path for filtering nested results for multiple parents for eager loading multiple associations",
			vParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "Children",
					Associations: []tags.Association{
						{
							Name: "Toys",
						},
					},
				},
				{
					Name: "Animals",
				},
			},
			[]interface{}{
				vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
					Children: []vChildModel{
						vChildModel{
							Name:     "kiddo",
							ID:       "00000000-0000-0000-0000-000000000002",
							ParentID: "00000000-0000-0000-0000-000000000001",
							Toys: []vToyModel{
								vToyModel{
									Name:     "lego",
									ID:       "00000000-0000-0000-0000-000000000001",
									ParentID: "00000000-0000-0000-0000-000000000002",
								},
							},
						},
					},
					Animals: []vPetModel{
						vPetModel{
							Name:     "spots",
							ID:       "00000000-0000-0000-0000-000000000003",
							ParentID: "00000000-0000-0000-0000-000000000001",
						},
					},
				},
				vParentModel{
					Name: "uncle",
					ID:   "00000000-0000-0000-0000-000000000004",
					Children: []vChildModel{
						vChildModel{
							Name:     "coz",
							ID:       "00000000-0000-0000-0000-000000000005",
							ParentID: "00000000-0000-0000-0000-000000000004",
						},
					},
					Animals: []vPetModel{
						vPetModel{
							Name:     "muffin",
							ID:       "00000000-0000-0000-0000-000000000004",
							ParentID: "00000000-0000-0000-0000-000000000004",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT parentmodel.id, parentmodel.organization_id, parentmodel.name, parentmodel.parent_id FROM parentmodel WHERE parentmodel.organization_id = \\$1 AND parentmodel.name = \\$2").
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id"}).
							AddRow("pops", "00000000-0000-0000-0000-000000000001").
							AddRow("uncle", "00000000-0000-0000-0000-000000000004"),
					)
				// parent is vParentModel
				mock.ExpectQuery("^SELECT childmodel.id, childmodel.organization_id, childmodel.name, childmodel.parent_id FROM childmodel JOIN parentmodel as t1 on t1.id = parent_id WHERE childmodel.organization_id = \\$1 AND t1.organization_id = \\$2 AND t1.name = \\$3").
					WithArgs(orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "parent_id"}).
							AddRow("kiddo", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000001").
							AddRow("coz", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000004"),
					)
				// parent is vChildModel
				mock.ExpectQuery("^SELECT toymodel.id, toymodel.organization_id, toymodel.name, toymodel.parent_id FROM toymodel JOIN childmodel as t1 on t1.id = parent_id JOIN parentmodel as t2 on t2.id = t1.parent_id WHERE toymodel.organization_id = \\$1 AND t1.organization_id = \\$2 AND t2.organization_id = \\$3 AND t2.name = \\$4").
					WithArgs(orgID, orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "parent_id"}).
							AddRow("lego", "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"),
					)
				// parent is vParentModel
				mock.ExpectQuery("^SELECT petmodel.id, petmodel.organization_id, petmodel.name, petmodel.parent_id FROM petmodel JOIN parentmodel as t1 on t1.id = parent_id WHERE petmodel.organization_id = \\$1 AND t1.organization_id = \\$2 AND t1.name = \\$3").
					WithArgs(orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "parent_id"}).
							AddRow("spots", "00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000001").
							AddRow("muffin", "00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000004"),
					)
			},
			nil,
		},
		{
			"happy path for filtering nested results for multiple parents for eager loading into a map with key mappings",
			vParentModel{
				Name: "pops",
			},
			[]tags.Association{
				{
					Name: "ChildrenMap",
				},
			},
			[]interface{}{
				vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
					ChildrenMap: map[string]vChildModel{
						"kiddo": vChildModel{
							Name:     "kiddo",
							ID:       "00000000-0000-0000-0000-000000000002",
							ParentID: "00000000-0000-0000-0000-000000000001",
						},
					},
				},
				vParentModel{
					Name: "uncle",
					ID:   "00000000-0000-0000-0000-000000000004",
					ChildrenMap: map[string]vChildModel{
						"coz": vChildModel{
							Name:     "coz",
							ID:       "00000000-0000-0000-0000-000000000005",
							ParentID: "00000000-0000-0000-0000-000000000004",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT parentmodel.id, parentmodel.organization_id, parentmodel.name, parentmodel.parent_id FROM parentmodel WHERE parentmodel.organization_id = \\$1 AND parentmodel.name = \\$2").
					WithArgs(orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id"}).
							AddRow("pops", "00000000-0000-0000-0000-000000000001").
							AddRow("uncle", "00000000-0000-0000-0000-000000000004"),
					)

				// parent is vParentModel
				mock.ExpectQuery("^SELECT childmodel.id, childmodel.organization_id, childmodel.name, childmodel.parent_id FROM childmodel JOIN parentmodel as t1 on t1.id = parent_id WHERE childmodel.organization_id = \\$1 AND t1.organization_id = \\$2 AND t1.name = \\$3").
					WithArgs(orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "parent_id"}).
							AddRow("kiddo", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000001").
							AddRow("coz", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000004"),
					)
			},
			nil,
		},
		{
			"happy path for filtering multiple parent nested results w/ eager loading associations",
			vParentModel{},
			[]tags.Association{
				{
					Name: "Children",
					Associations: []tags.Association{
						{
							Name: "Toys",
						},
					},
				},
			},
			[]interface{}{
				vParentModel{
					Name: "pops",
					ID:   "00000000-0000-0000-0000-000000000001",
					Children: []vChildModel{
						vChildModel{
							Name:     "kiddo",
							ID:       "00000000-0000-0000-0000-000000000002",
							ParentID: "00000000-0000-0000-0000-000000000001",
							Toys: []vToyModel{
								vToyModel{
									Name:     "lego",
									ID:       "00000000-0000-0000-0000-000000000001",
									ParentID: "00000000-0000-0000-0000-000000000002",
								},
							},
						},
					},
				},
				vParentModel{
					Name: "uncle",
					ID:   "00000000-0000-0000-0000-000000000004",
				},
				vParentModel{
					Name: "aunt",
					ID:   "00000000-0000-0000-0000-000000000005",
					Children: []vChildModel{
						vChildModel{
							Name:     "suzy",
							ID:       "00000000-0000-0000-0000-000000000009",
							ParentID: "00000000-0000-0000-0000-000000000005",
							Toys: []vToyModel{
								vToyModel{
									Name:     "beanie baby",
									ID:       "00000000-0000-0000-0000-000000000009",
									ParentID: "00000000-0000-0000-0000-000000000009",
								},
								vToyModel{
									Name:     "polly pocket",
									ID:       "00000000-0000-0000-0000-000000000011",
									ParentID: "00000000-0000-0000-0000-000000000009",
								},
							},
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT parentmodel.id, parentmodel.organization_id, parentmodel.name, parentmodel.parent_id FROM parentmodel WHERE parentmodel.organization_id = \\$1").
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id"}).
							AddRow("pops", "00000000-0000-0000-0000-000000000001").
							AddRow("uncle", "00000000-0000-0000-0000-000000000004").
							AddRow("aunt", "00000000-0000-0000-0000-000000000005"),
					)
				// parent is vParentModel
				mock.ExpectQuery("^SELECT childmodel.id, childmodel.organization_id, childmodel.name, childmodel.parent_id FROM childmodel WHERE childmodel.organization_id = \\$1").
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "parent_id"}).
							AddRow("kiddo", "00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000001").
							AddRow("suzy", "00000000-0000-0000-0000-000000000009", "00000000-0000-0000-0000-000000000005"),
					)
				// parent is vChildModel
				mock.ExpectQuery("^SELECT toymodel.id, toymodel.organization_id, toymodel.name, toymodel.parent_id FROM toymodel WHERE toymodel.organization_id = \\$1").
					WithArgs(orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "parent_id"}).
							AddRow("lego", "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002").
							AddRow("beanie baby", "00000000-0000-0000-0000-000000000009", "00000000-0000-0000-0000-000000000009").
							AddRow("polly pocket", "00000000-0000-0000-0000-000000000011", "00000000-0000-0000-0000-000000000009"),
					)
			},
			nil,
		},
	}

	for _, tc := range testCases[2:3] {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: orgID,
			}

			results, err := p.FilterModelAssociations(tc.filterModel, tc.associations)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

			assert.Equal(t, tc.wantErr, err)
		})
	}
}

// Let's explore relationships that run the other way, where the parent has an
// FK to a child.
type vObject struct {
	Metadata       metadata.Metadata `picard:"tablename=object"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	Fields         []vField          `json:"fields" picard:"child,foreign_key=ObjectID"`
}

type vField struct {
	Metadata       metadata.Metadata `picard:"tablename=field"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	ObjectID       string            `picard:"foreign_key,lookup,required,related=Object,column=object_id"`
	Object         vObject           `json:"object" validate:"-"`
	Reference      vReference        `json:"reference" picard:"reference,column=reference_id"`
}

type vReference struct {
	Metadata       metadata.Metadata `picard:"tablename=reference"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Field          vRefField         `picard:"field" picard:"reference,column=reference_field_id"`
}

type vRefField struct {
	Metadata       metadata.Metadata `picard:"tablename=field"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	Object         vRefObject        `json:"object" picard:"reference,column=object_id"`
}

type vRefObject struct {
	Metadata       metadata.Metadata `picard:"tablename=field"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
}

func fmtSQL(sql string) string {
	return strings.Replace(heredoc.Doc(sql), "\n", " ", -1)
}

func TestFilterModelReferenceAssociations(t *testing.T) {
	orgID := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		description  string
		filterModel  interface{}
		associations []tags.Association
		sqlmocks     func(sqlmock.Sqlmock)
		expected     []interface{}
	}{
		{
			"Loading objects with childTables and references",
			vObject{
				Name: "first_object",
			},
			[]tags.Association{
				{
					Name: "Fields",
					Associations: []tags.Association{
						{
							Name: "Reference",
							Associations: []tags.Association{
								{
									Name: "Field",
									Associations: []tags.Association{
										{
											Name: "Object",
										},
									},
								},
							},
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query.FmtSQLRegex(`
						SELECT t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name"
						FROM object AS t0
						WHERE
							t0.organization_id = $1 AND
							t0.name = $2
				`)).
					WithArgs(
						orgID,
						"first_object",
					).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id"}).
							AddRow("first_object", "00000000-0000-0000-0000-000000000002"),
					)

				// All of the reference chains should load in one query since
				// we're just following 1:1 references down
				mock.ExpectQuery(fmtSQL(`
					^SELECT t0.id as "t0.id",
						t0.organization_id as "t0.organization_id",
						t0.name as "t0.name",
						t0.reference_id as "t0.reference_id",
						t1.id as "t1.id",
						t1.reference_field_id as "t1.reference_field_id",
						t2.id as "t2.id",
						t2.name as "t2.name",
						t2.object_id as "t2.object_id",
						t3.id as "t3.id",
						t3.name as "t3.name"
					FROM field as t0
					JOIN
						reference as t1 on t0.reference_id = t1.id AND t0.organization_id = t1.organization_id
					JOIN
						field as t2 on t1.reference_field_id = t2.id AND t1.organization_id = t2.organization_id
					JOIN
						object as t3 on t2.object_id = t3.id AND t2.organization_id = t3.organization_id
					WHERE
						t0.organization_id = $1 AND
						t0.object_id = $2
				`)).
					WithArgs(orgID, orgID, "pops").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.name",
							"t0.reference_id",
							"t1.id",
							"t1.reference_field_id",
							"t2.id",
							"t2.name",
							"t2.object_id",
							"t3.id",
							"t3.name",
						}).
							AddRow( // Add a row with no references
								"00000000-0000-0000-0000-000000000003", // t0.id
								"field-1", // t0.name
								"",        // t0.reference_id
								"",        // t1.id
								"",        // t1.reference_field_id
								"",        // t2.id
								"",        // t2.name
								"",        // t2.object_id
								"",        // t3.id
								"",        // t3.name
							).
							AddRow( // And another row with references
								"00000000-0000-0000-0000-000000000004", // t0.id
								"field-2", // t0.name
								"00000000-0000-0000-0000-000000000005", // t0.reference_id
								"00000000-0000-0000-0000-000000000005", // t1.id
								"00000000-0000-0000-0000-000000000006", // t1.reference_field_id
								"00000000-0000-0000-0000-000000000006", // t2.id
								"referenced-field-1",                   // t2.name
								"00000000-0000-0000-0000-000000000007", // t2.object_id
								"00000000-0000-0000-0000-000000000007", // t3.id
								"referenced-object-1",                  // t3.name
							),
					)
			},
			[]interface{}{
				vObject{
					ID:   "00000000-0000-0000-0000-000000000002",
					Name: "first_object",
					Fields: []vField{
						{
							ID:       "00000000-0000-0000-0000-000000000003",
							Name:     "field-1",
							ObjectID: "00000000-0000-0000-0000-000000000002",
						},
						{
							ID:       "00000000-0000-0000-0000-000000000004",
							Name:     "field-2",
							ObjectID: "00000000-0000-0000-0000-000000000002",
							Reference: vReference{
								ID: "00000000-0000-0000-0000-000000000005",
								Field: vRefField{
									ID:   "00000000-0000-0000-0000-000000000006",
									Name: "referenced-field-1",
									Object: vRefObject{
										ID:   "00000000-0000-0000-0000-000000000007",
										Name: "referenced-object-1",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.sqlmocks(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: orgID,
			}

			actual, err := p.FilterModelAssociations(tc.filterModel, tc.associations)

			assert.NoError(err)
			assert.Equal(tc.expected, actual)

			// sqlmock expectations
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}

		})
	}
}

func TestDoFilterSelect(t *testing.T) {
	testMultitenancyValue := "00000000-0000-0000-0000-000000000001"
	testPerformedByValue := "00000000-0000-0000-0000-000000000002"
	testCases := []struct {
		description          string
		filterModelType      reflect.Type
		whereClauses         []squirrel.Sqlizer
		joinClauses          []string
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErr              error
	}{
		{
			"Should do query correctly and return correct values with single field",
			reflect.TypeOf(modelOneField{}),
			nil,
			nil,
			[]interface{}{
				&modelOneField{
					TestFieldOne: "test value 1",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1"),
				)
			},
			nil,
		},
		{
			"Should do query correctly with where clauses and return correct values with single field",
			reflect.TypeOf(modelOneField{}),
			[]squirrel.Sqlizer{squirrel.Eq{"test_column_one": "test value 1"}},
			nil,
			[]interface{}{
				&modelOneField{
					TestFieldOne: "test value 1",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table WHERE test_column_one = \\$1$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1"),
				)
			},
			nil,
		},
		{
			"Should do query correctly with where clauses and join clauses and return correct values with single field",
			reflect.TypeOf(modelOneField{}),
			[]squirrel.Sqlizer{squirrel.Eq{"test_column_one": "test value 1"}},
			[]string{"joinclause"},
			[]interface{}{
				&modelOneField{
					TestFieldOne: "test value 1",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table JOIN joinclause WHERE test_column_one = \\$1$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1"),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with two results",
			reflect.TypeOf(modelOneField{}),
			nil,
			nil,
			[]interface{}{
				&modelOneField{
					TestFieldOne: "test value 1",
				},
				&modelOneField{
					TestFieldOne: "test value 2",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).AddRow("test value 1").AddRow("test value 2"),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with special fields",
			reflect.TypeOf(modelMutitenantPKWithTwoFields{}),
			nil,
			nil,
			[]interface{}{
				&modelMutitenantPKWithTwoFields{
					TestMultitenancyField: "multitenancy value 1",
					TestPrimaryKeyField:   "primary key value 1",
					TestFieldOne:          "test value 1.1",
					TestFieldTwo:          "test value 1.2",
				},
				&modelMutitenantPKWithTwoFields{
					TestMultitenancyField: "multitenancy value 2",
					TestPrimaryKeyField:   "primary key value 2",
					TestFieldOne:          "test value 2.1",
					TestFieldTwo:          "test value 2.2",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_multitenancy_column, test_table.primary_key_column, test_table.test_column_one, test_table.test_column_two FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_multitenancy_column", "test_column_one", "test_column_two", "primary_key_column"}).
						AddRow("multitenancy value 1", "test value 1.1", "test value 1.2", "primary key value 1").
						AddRow("multitenancy value 2", "test value 2.1", "test value 2.2", "primary key value 2"),
				)
			},
			nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			results, err := p.doFilterSelect(tc.filterModelType, tc.whereClauses, tc.joinClauses, nil)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestDoFilterSelectWithEncrypted(t *testing.T) {

	testMultitenancyValue := "00000000-0000-0000-0000-000000000001"
	testPerformedByValue := "00000000-0000-0000-0000-000000000002"
	testCases := []struct {
		description          string
		filterModelType      reflect.Type
		whereClauses         []squirrel.Sqlizer
		nonce                string
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErrMsg           string
	}{
		{
			"Should do query correctly and return correct values with single encrypted field",
			reflect.TypeOf(modelOneFieldEncrypted{}),
			nil,
			"123412341234",
			[]interface{}{
				&modelOneFieldEncrypted{
					TestFieldOne: "some plaintext for encryption",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow("MTIzNDEyMzQxMjM0ibdgaIgpwjXpIQs645vZ8fXHC85nAKmvoh7MhF+9Bk/mLFTH3FcE4qTKAi5e"),
				)
			},
			"",
		},
		{
			"Should be able to select if encrypted field is nil",
			reflect.TypeOf(modelOneFieldEncrypted{}),
			nil,
			"123412341234",
			[]interface{}{
				&modelOneFieldEncrypted{},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow(nil),
				)
			},
			"",
		},
		{
			"Should be able to select if encrypted field is empty string",
			reflect.TypeOf(modelOneFieldEncrypted{}),
			nil,
			"123412341234",
			[]interface{}{
				&modelOneFieldEncrypted{},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow(""),
				)
			},
			"",
		},
		{
			"Should be able to select if some encrypted fields are nil and others are populated",
			reflect.TypeOf(modelTwoFieldEncrypted{}),
			nil,
			"123412341234",
			[]interface{}{
				&modelTwoFieldEncrypted{
					TestFieldOne: "some plaintext for encryption",
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one, test_table.test_column_two FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one", "test_column_two"}).
						AddRow("MTIzNDEyMzQxMjM0ibdgaIgpwjXpIQs645vZ8fXHC85nAKmvoh7MhF+9Bk/mLFTH3FcE4qTKAi5e", nil),
				)
			},
			"",
		},
		{
			"Should return error if encrypted field is not stored in base64",
			reflect.TypeOf(modelOneFieldEncrypted{}),
			nil,
			"123412341234",
			[]interface{}{
				&modelOneFieldEncrypted{},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow("some other string"),
				)
			},
			"base64 decoding of value failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db
			crypto.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			// Set up known nonce
			oldReader := rand.Reader
			rand.Reader = strings.NewReader(tc.nonce)

			results, err := p.doFilterSelect(tc.filterModelType, tc.whereClauses, []string{}, nil)

			// Tear down known nonce
			rand.Reader = oldReader

			if tc.wantErrMsg != "" {
				assert.EqualError(t, err, tc.wantErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestDoFilterSelectWithJSONBField(t *testing.T) {

	testMultitenancyValue := "00000000-0000-0000-0000-000000000001"
	testPerformedByValue := "00000000-0000-0000-0000-000000000002"
	testCases := []struct {
		description          string
		filterModelType      reflect.Type
		whereClauses         []squirrel.Sqlizer
		wantReturnInterfaces []interface{}
		expectationFunction  func(sqlmock.Sqlmock)
		wantErr              error
	}{
		{
			"Should do query correctly and return correct values with single JSONB field",
			reflect.TypeOf(modelOneFieldJSONB{}),
			nil,
			[]interface{}{
				&modelOneFieldJSONB{
					TestFieldOne: TestSerializedObject{
						Name:               "Matt",
						Active:             true,
						NonSerializedField: "",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow([]byte(`{"name":"Matt","active":true}`)),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with single JSONB field and string return",
			reflect.TypeOf(modelOneFieldJSONB{}),
			nil,
			[]interface{}{
				&modelOneFieldJSONB{
					TestFieldOne: TestSerializedObject{
						Name:               "Matt",
						Active:             true,
						NonSerializedField: "",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow(`{"name":"Matt","active":true}`),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with single pointer JSONB field",
			reflect.TypeOf(modelOnePointerFieldJSONB{}),
			nil,
			[]interface{}{
				&modelOnePointerFieldJSONB{
					TestFieldOne: &TestSerializedObject{
						Name:               "Ben",
						Active:             true,
						NonSerializedField: "",
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow([]byte(`{"name":"Ben","active":true}`)),
				)
			},
			nil,
		},
		{
			"Should do query correctly and return correct values with array JSONB field",
			reflect.TypeOf(modelOneArrayFieldJSONB{}),
			nil,
			[]interface{}{
				&modelOneArrayFieldJSONB{
					TestFieldOne: []TestSerializedObject{
						TestSerializedObject{
							Name:               "Matt",
							Active:             true,
							NonSerializedField: "",
						},
						TestSerializedObject{
							Name:               "Ben",
							Active:             true,
							NonSerializedField: "",
						},
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT test_table.test_column_one FROM test_table$").WillReturnRows(
					sqlmock.NewRows([]string{"test_column_one"}).
						AddRow([]byte(`[{"name":"Matt","active":true},{"name":"Ben","active":true}]`)),
				)
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			conn = db

			tc.expectationFunction(mock)

			// Create the Picard instance
			p := PersistenceORM{
				multitenancyValue: testMultitenancyValue,
				performedBy:       testPerformedByValue,
			}

			results, err := p.doFilterSelect(tc.filterModelType, tc.whereClauses, []string{}, nil)

			if tc.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturnInterfaces, results)

				// sqlmock expectations
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unmet sqlmock expectations: %s", err)
				}
			}

		})
	}
}

func TestHydrateModel(t *testing.T) {
	testCases := []struct {
		description     string
		filterModelType reflect.Type
		hydrationValues map[string]interface{}
		wantValue       interface{}
	}{
		{
			"Should hydrate columns",
			reflect.TypeOf(modelTwoField{}),
			map[string]interface{}{
				"test_column_one": "column one value",
				"test_column_two": "column two value",
			},
			modelTwoField{
				TestFieldOne: "column one value",
				TestFieldTwo: "column two value",
			},
		},
		{
			"Should hydrate multitenancy key like other columns",
			reflect.TypeOf(modelMultitenant{}),
			map[string]interface{}{
				"test_multitenancy_column": "test return value",
			},
			modelMultitenant{
				TestMultitenancyField: "test return value",
			},
		},
		{
			"Should hydrate primary key like other columns",
			reflect.TypeOf(modelPK{}),
			map[string]interface{}{
				"primary_key_column": "primary key column value",
			},
			modelPK{
				PrimaryKeyField: "primary key column value",
			},
		},
		{
			"Should not hydrate columns not provided",
			reflect.TypeOf(modelTwoField{}),
			map[string]interface{}{
				"test_column_one": "column one value",
			},
			modelTwoField{
				TestFieldOne: "column one value",
				TestFieldTwo: "",
			},
		},
		{
			"Should not hydrate columns without tags",
			reflect.TypeOf(modelTwoFieldOneTagged{}),
			map[string]interface{}{
				"test_column_one": "column one value",
				"test_column_two": "column two value",
			},
			modelTwoFieldOneTagged{
				TestFieldOne: "column one value",
				TestFieldTwo: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resultValue := hydrateModel(tc.filterModelType, tags.TableMetadataFromType(tc.filterModelType), tc.hydrationValues)
			assert.True(t, reflect.DeepEqual(tc.wantValue, resultValue.Elem().Interface()))
		})
	}
}
