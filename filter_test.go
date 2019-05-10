package picard

import (
	"crypto/rand"
	"reflect"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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
	GrandParent    vGrandParentModel      `json:"parent" picard:"reference,column=parent_id"`
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
	Metadata       metadata.Metadata `picard:"tablename=toymodel"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	ParentID       string            `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         vChildModel       `json:"parent" validate:"-"`
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
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000023",
					GrandParent: vGrandParentModel{
						ID:             "00000000-0000-0000-0000-000000000023",
						OrganizationID: orgID,
						Name:           "grandpops",
						Age:            77,
					},
				},
			},
			func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id",
						t1.id AS "t1.id",
						t1.organization_id AS "t1.organization_id",
						t1.name AS "t1.name",
						t1.age AS "t1.age" 
					FROM parentmodel AS t0
					LEFT JOIN grandparentmodel AS t1 ON t1.id = t0.parent_id
					WHERE
						t0.organization_id = $1 AND
						t0.name = $2 AND
						t1.organization_id = $3	
				`)).
					WithArgs(orgID, "pops", orgID).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
							"t1.id",
							"t1.organization_id",
							"t1.name",
							"t1.age",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000002",
								orgID,
								"pops",
								"00000000-0000-0000-0000-000000000023",
								"00000000-0000-0000-0000-000000000023",
								orgID,
								"grandpops",
								77,
							),
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
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000004",
					Children: []vChildModel{
						vChildModel{
							ID:             "00000000-0000-0000-0000-000000000011",
							OrganizationID: orgID,
							Name:           "kiddo",
							ParentID:       "00000000-0000-0000-0000-000000000002",
							Toys: []vToyModel{
								{
									ID:             "00000000-0000-0000-0000-000000000022",
									OrganizationID: orgID,
									Name:           "lego",
									ParentID:       "00000000-0000-0000-0000-000000000011",
								},
							},
						},
						vChildModel{
							ID:             "00000000-0000-0000-0000-000000000012",
							OrganizationID: orgID,
							Name:           "another_kid",
							ParentID:       "00000000-0000-0000-0000-000000000002",
							Toys: []vToyModel{
								{
									ID:             "00000000-0000-0000-0000-000000000023",
									OrganizationID: orgID,
									Name:           "Woody",
									ParentID:       "00000000-0000-0000-0000-000000000011",
								},
							},
						},
					},
					Animals: []vPetModel{
						{
							ID:             "00000000-0000-0000-0000-000000000031",
							OrganizationID: orgID,
							Name:           "spots",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
						{
							ID:             "00000000-0000-0000-0000-000000000032",
							OrganizationID: orgID,
							Name:           "muffin",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
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
								"00000000-0000-0000-0000-000000000004",
							),
					)

				// parent is vParentModel
				mock.ExpectQuery(query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM childmodel AS t0
					WHERE 
						t0.organization_id = $1 AND t0.parent_id = $2
				`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000011",
								orgID,
								"kiddo",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000012",
								orgID,
								"another_kid",
								"00000000-0000-0000-0000-000000000002",
							),
					)

				tmSQL := query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM toymodel AS t0
					WHERE
						t0.organization_id = $1 AND t0.parent_id = $2
				`)

				mock.ExpectQuery(tmSQL).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000011").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000022",
								orgID,
								"lego",
								"00000000-0000-0000-0000-000000000011",
							),
					)

				mock.ExpectQuery(tmSQL).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000012").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000023",
								orgID,
								"Woody",
								"00000000-0000-0000-0000-000000000011",
							),
					)

				mock.ExpectQuery(query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM petmodel AS t0
					WHERE
						t0.organization_id = $1 AND t0.parent_id = $2
				`)).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000031",
								orgID,
								"spots",
								"00000000-0000-0000-0000-000000000002",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000032",
								orgID,
								"muffin",
								"00000000-0000-0000-0000-000000000002",
							),
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
					ID:             "00000000-0000-0000-0000-000000000002",
					OrganizationID: orgID,
					Name:           "pops",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ChildrenMap: map[string]vChildModel{
						"kiddo": vChildModel{
							ID:             "00000000-0000-0000-0000-000000000021",
							OrganizationID: orgID,
							Name:           "kiddo",
							ParentID:       "00000000-0000-0000-0000-000000000002",
						},
					},
				},
				vParentModel{
					ID:             "00000000-0000-0000-0000-000000000003",
					OrganizationID: orgID,
					Name:           "uncle",
					ParentID:       "00000000-0000-0000-0000-000000000011",
					ChildrenMap: map[string]vChildModel{
						"coz": vChildModel{
							ID:             "00000000-0000-0000-0000-000000000022",
							OrganizationID: orgID,
							Name:           "coz",
							ParentID:       "00000000-0000-0000-0000-000000000003",
						},
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
								"00000000-0000-0000-0000-000000000011",
							).
							AddRow(
								"00000000-0000-0000-0000-000000000003",
								orgID,
								"uncle",
								"00000000-0000-0000-0000-000000000011",
							),
					)

				msql := query.FmtSQLRegex(`
					SELECT
						t0.id AS "t0.id",
						t0.organization_id AS "t0.organization_id",
						t0.name AS "t0.name",
						t0.parent_id AS "t0.parent_id"
					FROM childmodel AS t0
					WHERE 
						t0.organization_id = $1 AND t0.parent_id = $2
				`)

				mock.ExpectQuery(msql).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000002").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000021",
								orgID,
								"kiddo",
								"00000000-0000-0000-0000-000000000002",
							),
					)

				mock.ExpectQuery(msql).
					WithArgs(orgID, "00000000-0000-0000-0000-000000000003").
					WillReturnRows(
						sqlmock.NewRows([]string{
							"t0.id",
							"t0.organization_id",
							"t0.name",
							"t0.parent_id",
						}).
							AddRow(
								"00000000-0000-0000-0000-000000000022",
								orgID,
								"coz",
								"00000000-0000-0000-0000-000000000003",
							),
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
