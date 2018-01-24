package picard

import (
	"database/sql/driver"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

// TestObject sample parent object for tests
type TestObject struct {
	Metadata Metadata `picard:"tablename=testobject"`

	ID             string `json:"id" picard:"primary_key,column=id"`
	OrganizationID string `picard:"multitenancy_key,column=organization_id"`

	Name           string            `json:"name" picard:"lookup,column=name"`
	NullableLookup string            `json:"nullableLookup" picard:"lookup,column=nullable_lookup"`
	Type           string            `json:"type" picard:"column=type"`
	IsActive       bool              `json:"is_active" picard:"column=is_active"`
	Children       []ChildTestObject `json:"children" picard:"child,foreign_key=ParentID"`
}

// ChildTestObject sample child object for tests
type ChildTestObject struct {
	Metadata Metadata `picard:"tablename=childtest"`

	ID             string `json:"id" picard:"primary_key,column=id"`
	OrganizationID string `picard:"multitenancy_key,column=organization_id"`

	ParentID string `picard:"lookup,column=parent_id"`
	Name     string `json:"name" picard:"lookup,column=name"`
}

var testObjectHelper = ExpectationHelper{
	TableName:        "testobject",
	LookupSelect:     "testobject.id, testobject.name as testobject_name, testobject.nullable_lookup as testobject_nullable_lookup",
	LookupWhere:      `COALESCE(testobject.name::"varchar",'') || '|' || COALESCE(testobject.nullable_lookup::"varchar",'')`,
	LookupReturnCols: []string{"id", "testobject_name", "testobject_nullable_lookup"},
	LookupFields:     []string{"Name", "NullableLookup"},
	DBColumns:        []string{"organization_id", "name", "nullable_lookup", "type", "is_active"},
	DataFields:       []string{"OrganizationID", "Name", "NullableLookup", "Type", "IsActive"},
}

var testChildObjectHelper = ExpectationHelper{
	TableName:        "childtest",
	LookupSelect:     "childtest.id, childtest.parent_id as childtest_parent_id, childtest.name as childtest_name",
	LookupWhere:      `COALESCE(childtest.parent_id::"varchar",'') || '|' || COALESCE(childtest.name::"varchar",'')`,
	LookupReturnCols: []string{"id", "childtest_parent_id", "childtest_name"},
	LookupFields:     []string{"ParentID", "Name"},
	DBColumns:        []string{"organization_id", "parent_id", "name"},
	DataFields:       []string{"OrganizationID", "ParentID", "Name"},
}

// Loads in a fixture data source from file
func loadTestObjects(names []string) ([]TestObject, error) {

	fixtures, err := LoadFixturesFromFiles(names, "./testdata/", reflect.TypeOf(TestObject{}))
	if err != nil {
		return nil, err
	}

	return fixtures.([]TestObject), nil

}

func TestDeployments(t *testing.T) {

	cases := []struct {
		TestName            string
		FixtureNames        []string
		ExpectationFunction func(*sqlmock.Sqlmock, interface{})
	}{
		{
			"Single Import with Nothing Existing",
			[]string{"Simple"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				ExpectInsert(mock, testObjectHelper, fixtures)
			},
		},
		{
			"Single Import with That Already Exists",
			[]string{"Simple"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)
			},
		},
		{
			"Single Import with Null Matches Existing value with a Null lookup",
			[]string{"Simple"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := [][]driver.Value{
					[]driver.Value{
						uuid.NewV4().String(),
						"Simple",
						nil,
					},
				}
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)
			},
		},
		{
			"Multiple Import with Nothing Existing",
			[]string{"Simple", "Simple2"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				ExpectInsert(mock, testObjectHelper, fixtures)
			},
		},

		{
			"Multiple Import with Both Already Exist",
			[]string{"Simple", "Simple2"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)
			},
		},
		{
			"Multiple Import with One Already Exists",
			[]string{"Simple", "Simple2"},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, []TestObject{
					fixtures[0],
				})
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				ExpectUpdate(mock, testObjectHelper, []TestObject{
					fixtures[0],
				}, returnData)
				ExpectInsert(mock, testObjectHelper, []TestObject{
					fixtures[1],
				})
			},
		},
		{
			"Single Import with Children",
			[]string{"SimpleWithChildren"},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				ExpectLookup(mock, testObjectHelper, fixtures, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, fixtures)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				ExpectLookup(mock, testChildObjectHelper, childObjects, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, childObjects)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			fixtures, err := loadTestObjects(c.FixtureNames)
			if err != nil {
				t.Fatal(err)
			}

			if err = RunImportTest(fixtures, c.ExpectationFunction); err != nil {
				t.Fatal(err)
			}
		})
	}

}

func TestGenerateWhereClausesFromModel(t *testing.T) {

	testMultitenancyValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000001")
	testPerformedByValue, _ := uuid.FromString("00000000-0000-0000-0000-000000000002")

	testCases := []struct {
		description      string
		filterModelValue reflect.Value
		zeroFields       []string
		wantClauses      []squirrel.Eq
		wantErr          string
	}{
		{
			"Filter object with no values should add multitenancy key",
			reflect.ValueOf(struct {
				OrgID string `picard:"multitenancy_key,column=organization_id"`
			}{}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"organization_id": testMultitenancyValue,
				},
			},
			"",
		},
		{
			"Filter object with no values and different multitenancy column should add multitenancy key",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
			}{}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
			},
			"",
		},
		{
			"Filter object with value for multitenancy column should be overwritten with picard multitenancy value",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
			}{
				TestMultitenancyColumn: "this value should be ignored",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
			},
			"",
		},
		{
			"Filter object with one value and multitenancy column should add both where clauses",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestField              string `picard:"column=test_column_one"`
			}{
				TestField: "first test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
				squirrel.Eq{
					"test_column_one": "first test value",
				},
			},
			"",
		},
		{
			"Filter object with two values and multitenancy column should add all where clauses",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestFieldOne           string `picard:"column=test_column_one"`
				TestFieldTwo           string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
				TestFieldTwo: "second test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_multitenancy_column": testMultitenancyValue,
				},
				squirrel.Eq{
					"test_column_one": "first test value",
				},
				squirrel.Eq{
					"test_column_two": "second test value",
				},
			},
			"",
		},
		{
			"Filter object with two values and only one is picard column should add only one where clause",
			reflect.ValueOf(struct {
				TestFieldOne string `picard:"column=test_column_one"`
				TestFieldTwo string
			}{
				TestFieldOne: "first test value",
				TestFieldTwo: "second test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_column_one": "first test value",
				},
			},
			"",
		},
		{
			"Filter object with two values and one is zero value should add only one where clause",
			reflect.ValueOf(struct {
				TestFieldOne string `picard:"column=test_column_one"`
				TestFieldTwo string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
			}),
			nil,
			[]squirrel.Eq{
				squirrel.Eq{
					"test_column_one": "first test value",
				},
			},
			"",
		},
		{
			"Filter object with two values and one is zero value and in zeroFields list should add both where clauses",
			reflect.ValueOf(struct {
				TestFieldOne string `picard:"column=test_column_one"`
				TestFieldTwo string `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
			}),
			[]string{"TestFieldTwo"},
			[]squirrel.Eq{
				squirrel.Eq{
					"test_column_one": "first test value",
				},
				squirrel.Eq{
					"test_column_two": "",
				},
			},
			"",
		},
		{
			"Filter object with value for encrypted field should return error",
			reflect.ValueOf(struct {
				TestMultitenancyColumn string `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestField              string `picard:"encrypted,column=test_column_one"`
			}{
				TestField: "first test value",
			}),
			nil,
			nil,
			"cannot perform queries with where clauses on encrypted fields",
		},
	}

	// Create the Picard instance
	p := PersistenceORM{
		multitenancyValue: testMultitenancyValue,
		performedBy:       testPerformedByValue,
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			results, err := p.generateWhereClausesFromModel(tc.filterModelValue, tc.zeroFields)

			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantClauses, results)
			}
		})
	}
}
