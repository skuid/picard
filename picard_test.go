package picard

import (
	"fmt"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

// TestObject sample parent object for tests
type TestObject struct {
	Name           string            `json:"name" picard:"lookup,column=name"`
	Type           string            `json:"type" picard:"column=type"`
	OrganizationID string            `picard:"column=organization_id"`
	Children       []ChildTestObject `json:"children" picard:"child,foreign_key=ParentID"`
	Metadata       StructMetadata    `picard:"tablename=testobject"`
}

// ChildTestObject sample child object for tests
type ChildTestObject struct {
	ParentID       string         `picard:"lookup,column=parent_id"`
	Name           string         `json:"name" picard:"lookup,column=name"`
	OrganizationID string         `picard:"column=organization_id"`
	Metadata       StructMetadata `picard:"tablename=childtest"`
}

var testObjectHelper = ExpectationHelper{
	TableName:        "testobject",
	LookupSelect:     "testobject.id, testobject.name as testobject_name",
	LookupWhere:      `testobject.name`,
	LookupReturnCols: []string{"id", "testobject_name"},
	LookupFields:     []string{"Name"},
	DBColumns:        []string{"name", "type", "organization_id"},
	DataFields:       []string{"Name", "Type", "OrganizationID"},
}

var testChildObjectHelper = ExpectationHelper{
	TableName:        "childtest",
	LookupSelect:     "childtest.id, childtest.parent_id as childtest_parent_id, childtest.name as childtest_name",
	LookupWhere:      `childtest.parent_id || '|' || childtest.name`,
	LookupReturnCols: []string{"id", "childtest_parent_id", "childtest_name"},
	LookupFields:     []string{"ParentID", "Name"},
	DBColumns:        []string{"parent_id", "name", "organization_id"},
	DataFields:       []string{"ParentID", "Name", "OrganizationID"},
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

	for index, c := range cases {
		fmt.Printf("%v: %v", index+1, c.TestName)
		fmt.Println("")
		fixtures, err := loadTestObjects(c.FixtureNames)
		if err != nil {
			t.Fatal(err)
		}

		if err = RunImportTest(fixtures, c.ExpectationFunction); err != nil {
			t.Fatal(err)
		}
	}

}
