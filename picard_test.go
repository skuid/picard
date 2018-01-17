package picard

import (
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
	"github.com/magiconair/properties/assert"
	uuid "github.com/satori/go.uuid"
)

// TestObject sample parent object for tests
type TestObject struct {
	Metadata Metadata `picard:"tablename=testobject"`

	ID             string `json:"id" picard:"primary_key,column=id"`
	OrganizationID string `picard:"multitenancy_key,column=organization_id"`

	Name     string            `json:"name" picard:"lookup,column=name"`
	Type     string            `json:"type" picard:"column=type"`
	Children []ChildTestObject `json:"children" picard:"child,foreign_key=ParentID"`
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
	LookupSelect:     "testobject.id, testobject.name as testobject_name",
	LookupWhere:      `testobject.name`,
	LookupReturnCols: []string{"id", "testobject_name"},
	LookupFields:     []string{"Name"},
	DBColumns:        []string{"organization_id", "name", "type"},
	DataFields:       []string{"OrganizationID", "Name", "Type"},
}

var testChildObjectHelper = ExpectationHelper{
	TableName:        "childtest",
	LookupSelect:     "childtest.id, childtest.parent_id as childtest_parent_id, childtest.name as childtest_name",
	LookupWhere:      `childtest.parent_id || '|' || childtest.name`,
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
		ExpectationFunction func(sqlmock.Sqlmock, interface{})
	}{
		{
			"Single Import with Nothing Existing",
			[]string{"Simple"},
			func(mock sqlmock.Sqlmock, fixtures interface{}) {
				ExpectLookup(mock, testObjectHelper, fixtures, nil)
				ExpectInsert(mock, testObjectHelper, fixtures)
			},
		},
		{
			"Single Import with That Already Exists",
			[]string{"Simple"},
			func(mock sqlmock.Sqlmock, fixtures interface{}) {
				rows := ExpectLookup(mock, testObjectHelper, fixtures, fixtures)
				ExpectUpdate(mock, testObjectHelper, fixtures, rows)
			},
		},
		{
			"Multiple Import with Nothing Existing",
			[]string{"Simple", "Simple2"},
			func(mock sqlmock.Sqlmock, fixtures interface{}) {
				ExpectLookup(mock, testObjectHelper, fixtures, nil)
				ExpectInsert(mock, testObjectHelper, fixtures)
			},
		},
		{
			"Multiple Import with Both Already Exist",
			[]string{"Simple", "Simple2"},
			func(mock sqlmock.Sqlmock, fixtures interface{}) {
				rows := ExpectLookup(mock, testObjectHelper, fixtures, fixtures)
				ExpectUpdate(mock, testObjectHelper, fixtures, rows)
			},
		},
		{
			"Multiple Import with One Already Exists",
			[]string{"Simple", "Simple2"},
			func(mock sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				rows := ExpectLookup(mock, testObjectHelper, fixtures, []TestObject{
					fixtures[0],
				})
				ExpectUpdate(mock, testObjectHelper, []TestObject{
					fixtures[0],
				}, rows)
				ExpectInsert(mock, testObjectHelper, []TestObject{
					fixtures[1],
				})
			},
		},
		{
			"Single Import with Children",
			[]string{"SimpleWithChildren"},
			func(mock sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				ExpectLookup(mock, testObjectHelper, fixtures, nil)
				insertRows := ExpectInsert(mock, testObjectHelper, fixtures)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				ExpectLookup(mock, testChildObjectHelper, childObjects, nil)
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
		},
	}

	// Create the Picard instance
	p := PersistenceORM{
		multitenancyValue: testMultitenancyValue,
		performedBy:       testPerformedByValue,
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			results := p.generateWhereClausesFromModel(tc.filterModelValue, tc.zeroFields)
			assert.Equal(t, tc.wantClauses, results)
		})
	}
}
