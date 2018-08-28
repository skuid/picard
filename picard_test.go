package picard

import (
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

// Config is a sample struct that would go in a jsonb field
type Config struct {
	ConfigA string
	ConfigB string
}

// ParentTestObject sample parent object for tests
type ParentTestObject struct {
	Metadata Metadata `picard:"tablename=parenttest"`

	ID             string       `json:"id" picard:"primary_key,column=id"`
	OrganizationID string       `picard:"multitenancy_key,column=organization_id"`
	Name           string       `json:"name" picard:"column=name"`
	Children       []TestObject `json:"children" picard:"child,foreign_key=ParentID"`
}

// TestObject sample parent object for tests
type TestObject struct {
	Metadata Metadata `picard:"tablename=testobject"`

	ID             string                     `json:"id" picard:"primary_key,column=id"`
	OrganizationID string                     `picard:"multitenancy_key,column=organization_id"`
	Name           string                     `json:"name" picard:"lookup,column=name" validate:"required"`
	NullableLookup string                     `json:"nullableLookup" picard:"lookup,column=nullable_lookup"`
	Type           string                     `json:"type" picard:"column=type"`
	IsActive       bool                       `json:"is_active" picard:"column=is_active"`
	Children       []ChildTestObject          `json:"children" picard:"child,foreign_key=ParentID"`
	ChildrenMap    map[string]ChildTestObject `json:"childrenmap" picard:"child,foreign_key=ParentID,key_mapping=Name,value_mappings=Type->OtherInfo"`
	ParentID       string                     `picard:"foreign_key,related=Parent,column=parent_id"`
	Parent         ParentTestObject           `validate:"-"`
	Config         Config                     `json:"config" picard:"jsonb,column=config"`
	CreatedByID    string                     `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID    string                     `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate    time.Time                  `picard:"column=created_at,audit=created_at"`
	UpdatedDate    time.Time                  `picard:"column=updated_at,audit=updated_at"`
}

// test for deleteExisting option
type TestObjectDelete struct {
	Metadata Metadata `picard:"tablename=testobjectdelete,deleteExisting"`

	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name" validate:"required"`
	IsActive       bool              `json:"is_active" picard:"column=is_active"`
	Children       []ChildTestObject `json:"children" picard:"child,foreign_key=ParentID"`
}

// ChildTestObject sample child object for tests
type ChildTestObject struct {
	Metadata Metadata `picard:"tablename=childtest"`

	ID               string     `json:"id" picard:"primary_key,column=id"`
	OrganizationID   string     `picard:"multitenancy_key,column=organization_id"`
	Name             string     `json:"name" picard:"lookup,column=name"`
	OtherInfo        string     `picard:"column=other_info"`
	ParentID         string     `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent           TestObject `json:"parent" validate:"-"`
	OptionalParentID string     `picard:"foreign_key,related=OptionalParent,column=optional_parent_id"`
	OptionalParent   TestObject `json:"optional_parent" validate:"-"`
}

// ChildTestObjectWithKeyMap sample child object for tests
type ChildTestObjectWithKeyMap struct {
	Metadata Metadata `picard:"tablename=childtest"`

	ID               string     `json:"id" picard:"primary_key,column=id"`
	OrganizationID   string     `picard:"multitenancy_key,column=organization_id"`
	Name             string     `json:"name" picard:"lookup,column=name"`
	OtherInfo        string     `picard:"column=other_info"`
	ParentID         string     `json:"parent" picard:"foreign_key,lookup,required,related=Parent,column=parent_id,key_map=Name"`
	Parent           TestObject `validate:"-"`
	OptionalParentID string     `picard:"foreign_key,related=OptionalParent,column=optional_parent_id"`
	OptionalParent   TestObject `json:"optional_parent" validate:"-"`
}

type TestParentSerializedObject struct {
	Metadata Metadata `picard:"tablename=parent_serialize"`

	ID               string                 `json:"id" picard:"primary_key,column=id"`
	SerializedThings []TestSerializedObject `json:"serialized_things" picard:"jsonb,column=serialized_things"`
}

// SerializedObject sample object to be stored in a JSONB column
type TestSerializedObject struct {
	Name               string `json:"name"`
	Active             bool   `json:"active"`
	NonSerializedField string `json:"-"`
}

var parentObjectHelper = ExpectationHelper{
	FixtureType:      ParentTestObject{},
	LookupSelect:     "",
	LookupWhere:      "",
	LookupReturnCols: []string{},
	LookupFields:     []string{},
}

var testObjectHelper = ExpectationHelper{
	FixtureType:      TestObject{},
	LookupSelect:     "testobject.id, testobject.name as testobject_name, testobject.nullable_lookup as testobject_nullable_lookup",
	LookupWhere:      `COALESCE(testobject.name::"varchar",'') || '|' || COALESCE(testobject.nullable_lookup::"varchar",'')`,
	LookupReturnCols: []string{"id", "testobject_name", "testobject_nullable_lookup"},
	LookupFields:     []string{"Name", "NullableLookup"},
}

var testObjectWithPKHelper = ExpectationHelper{
	FixtureType:      TestObject{},
	LookupSelect:     "testobject.id, testobject.id as testobject_id",
	LookupWhere:      `COALESCE(testobject.id::"varchar",'')`,
	LookupReturnCols: []string{"id", "testobject_id"},
	LookupFields:     []string{"ID"},
}

var testObjectWithDeleteHelper = ExpectationHelper{
	FixtureType:      TestObjectDelete{},
	LookupSelect:     "testobjectdelete.id, testobjectdelete.name as testobjectdelete_name",
	LookupWhere:      "",
	LookupReturnCols: []string{"id", "testobjectdelete_id"},
	LookupFields:     []string{"ID"},
}

var testChildObjectHelper = ExpectationHelper{
	FixtureType:      ChildTestObject{},
	LookupSelect:     "childtest.id, childtest.name as childtest_name, childtest.parent_id as childtest_parent_id",
	LookupWhere:      `COALESCE(childtest.name::"varchar",'') || '|' || COALESCE(childtest.parent_id::"varchar",'')`,
	LookupReturnCols: []string{"id", "childtest_name", "childtest_parent_id"},
	LookupFields:     []string{"Name", "ParentID"},
}

var testChildObjectWithLookupHelper = ExpectationHelper{
	FixtureType:      ChildTestObject{},
	LookupFrom:       "childtest JOIN testobject as t1 on t1.id = parent_id",
	LookupSelect:     "childtest.id, childtest.name as childtest_name, t1.name as t1_name, t1.nullable_lookup as t1_nullable_lookup",
	LookupWhere:      `COALESCE(childtest.name::"varchar",'') || '|' || COALESCE(t1.name::"varchar",'') || '|' || COALESCE(t1.nullable_lookup::"varchar",'')`,
	LookupReturnCols: []string{"id", "childtest_name", "t1_name", "t1_nullable_lookup"},
	LookupFields:     []string{"Name", "ParentID"},
}

func TestSerializeJSONBColumns(t *testing.T) {
	testCases := []struct {
		testDescription string
		giveColumns     []string
		giveObject      map[string]interface{}
		wantObject      map[string]interface{}
		wantErrMsg      string
	}{
		{
			testDescription: "serializes only columns provided into JSON format",
			giveColumns: []string{
				"column_one",
			},
			giveObject: map[string]interface{}{
				"column_one": TestSerializedObject{
					Name:               "Matt",
					Active:             true,
					NonSerializedField: "is this the real life?",
				},
				"column_two": "will not be serialized",
			},
			wantObject: map[string]interface{}{
				"column_one": []byte(`{"name":"Matt","active":true}`),
				"column_two": "will not be serialized",
			},
			wantErrMsg: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			err := serializeJSONBColumns(tc.giveColumns, tc.giveObject)
			if tc.wantErrMsg != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantObject, tc.giveObject)
			}
		})
	}
}

// Loads in a fixture data source from file
func loadTestObjects(names []string, structType interface{}) (interface{}, error) {

	fixtures, err := LoadFixturesFromFiles(names, "./testdata/", reflect.TypeOf(structType))
	if err != nil {
		return nil, err
	}

	return fixtures, nil

}

func TestDeployments(t *testing.T) {

	cases := []struct {
		TestName            string
		FixtureNames        []string
		FixtureType         interface{}
		ExpectationFunction func(*sqlmock.Sqlmock, interface{})
		WantErr             string
	}{
		{
			"Single Import with Primary Key with Nothing Existing",
			[]string{"SimpleWithPrimaryKey"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectWithPKHelper
				returnData := GetReturnDataForLookup(helper, nil)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectInsert(mock, helper, fixtures, true)
			},
			"",
		},
		{
			"Single Import with Primary Key That Already Exists",
			[]string{"SimpleWithPrimaryKey"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectWithPKHelper
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, fixtures, returnData)
			},
			"",
		},
		{
			"Single Import with Delete Existing",
			[]string{"Simple"},
			TestObjectDelete{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObjectDelete)
				helper := testObjectWithDeleteHelper
				fixturesToDelete, _ := loadTestObjects([]string{"SimpleWithPrimaryKey"}, TestObjectDelete{})
				concreteDeleteFixtures := fixturesToDelete.([]TestObjectDelete)
				returnData := GetReturnDataForLookup(helper, append(fixtures, concreteDeleteFixtures...))
				lookupKeys := GetLookupKeys(helper, append(fixtures, concreteDeleteFixtures...))

				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectDelete(mock, helper, concreteDeleteFixtures[0], returnData[1:2])
				ExpectInsert(mock, helper, fixtures, false)
			},
			"",
		},
		{
			"Single Import with Nothing Existing",
			[]string{"Simple"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectHelper
				returnData := GetReturnDataForLookup(helper, nil)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectInsert(mock, helper, fixtures, false)
			},
			"",
		},
		{
			"Single Import with That Already Exists",
			[]string{"Simple"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, fixtures)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)
			},
			"",
		},
		{
			"Single Import Missing Required Field",
			[]string{"Empty"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {},
			"Key: 'TestObject.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
		{
			"Single Import with Null Matches Existing value with a Null lookup",
			[]string{"Simple"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := [][]driver.Value{
					[]driver.Value{
						uuid.NewV4().String(),
						"Simple",
						nil,
					},
				}
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)
			},
			"",
		},
		{
			"Multiple Import with Nothing Existing",
			[]string{"Simple", "Simple2"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectInsert(mock, testObjectHelper, fixtures, false)
			},
			"",
		},
		{
			"Multiple Import with Both Already Exist",
			[]string{"Simple", "Simple2"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := GetReturnDataForLookup(testObjectHelper, fixtures)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)
			},
			"",
		},
		{
			"Multiple Import with One Already Exists",
			[]string{"Simple", "Simple2"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, []TestObject{
					fixtures[0],
				})
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, testObjectHelper, []TestObject{
					fixtures[0],
				}, returnData)
				ExpectInsert(mock, testObjectHelper, []TestObject{
					fixtures[1],
				}, false)
			},
			"",
		},

		{
			"Multiple Import with Delete Existing",
			[]string{"Simple", "Simple2"},
			TestObjectDelete{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObjectDelete)
				helper := testObjectWithDeleteHelper
				fixturesToDelete, _ := loadTestObjects([]string{"SimpleWithPrimaryKey"}, TestObjectDelete{})
				concreteDeleteFixtures := fixturesToDelete.([]TestObjectDelete)
				returnData := GetReturnDataForLookup(helper, append(fixtures, concreteDeleteFixtures...))
				lookupKeys := GetLookupKeys(helper, append(fixtures, concreteDeleteFixtures...))

				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectDelete(mock, helper, concreteDeleteFixtures[0], returnData[2:3])
				ExpectInsert(mock, helper, fixtures, false)
			},
			"",
		},
		{
			"Single Import with GrandChildren All Inserts",
			[]string{"SimpleWithGrandChildren"},
			ParentTestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]ParentTestObject)
				insertRows := ExpectInsert(mock, parentObjectHelper, fixtures, false)

				testObjects := []TestObject{}
				for index, fixture := range fixtures {
					for _, testObject := range fixture.Children {
						testObject.ParentID = insertRows[index][0].(string)
						testObjects = append(testObjects, testObject)
					}
				}

				testReturnData := GetReturnDataForLookup(testObjectHelper, nil)
				testLookupKeys := GetLookupKeys(testObjectHelper, testObjects)
				ExpectLookup(mock, testObjectHelper, testLookupKeys, testReturnData)

				childInsertRows := ExpectInsert(mock, testObjectHelper, testObjects, false)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children[0].Children {
						childObject.ParentID = childInsertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, childObjects, false)
			},
			"",
		},
		{
			"Single Import with Children Insert New Parent",
			[]string{"SimpleWithChildren"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, fixtures, false)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, childObjects, false)
			},
			"",
		},
		{
			"Single Import with ChildrenMap Insert New Parent",
			[]string{"SimpleWithChildrenMap"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, fixtures, false)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.ChildrenMap {
						childObject.ParentID = insertRows[index][0].(string)
						// Tests that the key mapping "Name" worked correctly
						childObject.Name = "ChildRecord1"
						// Tests that the value mapping "Type->OtherInfo" worked correctly
						childObject.OtherInfo = fixtures[0].Type
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)

				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, childObjects, false)
			},
			"",
		},
		{
			"Single Import with Children Existing Parent",
			[]string{"SimpleWithChildren"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, fixtures)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = returnData[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, childObjects, false)
			},
			"",
		},
		{
			"Single Import with Children Existing Parent and Existing Child",
			[]string{"SimpleWithChildren"},
			TestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, fixtures)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, testObjectHelper, fixtures, returnData)

				childObjects := []ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = returnData[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, childObjects)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectUpdate(mock, testChildObjectHelper, childObjects, childReturnData)
			},
			"",
		},
		{
			"Import Existing Child with Reference to Parent Name",
			[]string{"ChildWithParentLookup"},
			ChildTestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				childUUID := uuid.NewV4().String()
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{
					[]driver.Value{
						childUUID,
						"ChildItem",
						"Simple",
						"",
					},
				}

				childObjects := []ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, ChildTestObject{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{
					[]driver.Value{
						parentUUID,
						"Simple",
						"",
					},
				})

				ExpectUpdate(mock, testChildObjectWithLookupHelper, childObjects, returnData)
			},
			"",
		},
		{
			"Import New Child with Reference to Parent Name",
			[]string{"ChildWithParentLookup"},
			ChildTestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, ChildTestObject{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{
					[]driver.Value{
						parentUUID,
						"Simple",
						"",
					},
				})

				ExpectInsert(mock, testChildObjectWithLookupHelper, childObjects, false)
			},
			"",
		},
		{
			"Import New Child with Reference to Parent Name Using Key Map",
			[]string{"ChildWithParentLookupAndKeyMap"},
			ChildTestObjectWithKeyMap{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]ChildTestObjectWithKeyMap)
				lookupKeys := []string{"ChildItem|Simple"}
				returnData := [][]driver.Value{}

				childObjects := []ChildTestObjectWithKeyMap{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, ChildTestObjectWithKeyMap{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{
					[]driver.Value{
						parentUUID,
						"Simple",
						"",
					},
				})

				ExpectInsert(mock, testChildObjectHelper, childObjects, false)
			},
			"",
		},
		{
			"Import New Child with Reference to Parent Name And Optional Parent",
			[]string{"ChildWithParentLookupAndOptionalLookup"},
			ChildTestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				optionalParentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, ChildTestObject{
						Name:             fixture.Name,
						ParentID:         parentUUID,
						OptionalParentID: optionalParentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{
					[]driver.Value{
						parentUUID,
						"Simple",
						"",
					},
				})

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple2|"}, [][]driver.Value{
					[]driver.Value{
						optionalParentUUID,
						"Simple2",
						"",
					},
				})

				ExpectInsert(mock, testChildObjectWithLookupHelper, childObjects, false)
			},
			"",
		},
		{
			"Import New Child with Bad Reference to Parent Name",
			[]string{"ChildWithParentLookup"},
			ChildTestObject{},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, ChildTestObject{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{})

				ExpectInsert(mock, testChildObjectWithLookupHelper, childObjects, false)
			},
			"Missing Required Foreign Key Lookup",
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			fixtures, err := loadTestObjects(c.FixtureNames, c.FixtureType)
			if err != nil {
				t.Fatal(err)
			}

			err = RunImportTest(fixtures, c.ExpectationFunction)

			if c.WantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, c.WantErr)
			}
		})
	}

}

func TestGenerateWhereClausesFromModel(t *testing.T) {

	testMultitenancyValue := "00000000-0000-0000-0000-000000000001"
	testPerformedByValue := "00000000-0000-0000-0000-000000000002"

	testCases := []struct {
		description      string
		filterModelValue reflect.Value
		zeroFields       []string
		wantWhereClauses []squirrel.Sqlizer
		wantJoinClauses  []string
		wantErr          string
	}{
		{
			"Filter object with no values should add multitenancy key",
			reflect.ValueOf(struct {
				Metadata Metadata `picard:"tablename=test_table"`
				OrgID    string   `picard:"multitenancy_key,column=organization_id"`
			}{}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.organization_id": testMultitenancyValue,
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with no values and different multitenancy column should add multitenancy key",
			reflect.ValueOf(struct {
				Metadata               Metadata `picard:"tablename=test_table"`
				TestMultitenancyColumn string   `picard:"multitenancy_key,column=test_multitenancy_column"`
			}{}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_multitenancy_column": testMultitenancyValue,
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with value for multitenancy column should be overwritten with picard multitenancy value",
			reflect.ValueOf(struct {
				Metadata               Metadata `picard:"tablename=test_table"`
				TestMultitenancyColumn string   `picard:"multitenancy_key,column=test_multitenancy_column"`
			}{
				TestMultitenancyColumn: "this value should be ignored",
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_multitenancy_column": testMultitenancyValue,
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with one value and multitenancy column should add both where clauses",
			reflect.ValueOf(struct {
				Metadata               Metadata `picard:"tablename=test_table"`
				TestMultitenancyColumn string   `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestField              string   `picard:"column=test_column_one"`
			}{
				TestField: "first test value",
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_multitenancy_column": testMultitenancyValue,
				},
				squirrel.Eq{
					"test_table.test_column_one": "first test value",
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with two values and multitenancy column should add all where clauses",
			reflect.ValueOf(struct {
				Metadata               Metadata `picard:"tablename=test_table"`
				TestMultitenancyColumn string   `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestFieldOne           string   `picard:"column=test_column_one"`
				TestFieldTwo           string   `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
				TestFieldTwo: "second test value",
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_multitenancy_column": testMultitenancyValue,
				},
				squirrel.Eq{
					"test_table.test_column_one": "first test value",
				},
				squirrel.Eq{
					"test_table.test_column_two": "second test value",
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with two values and only one is picard column should add only one where clause",
			reflect.ValueOf(struct {
				Metadata     Metadata `picard:"tablename=test_table"`
				TestFieldOne string   `picard:"column=test_column_one"`
				TestFieldTwo string
			}{
				TestFieldOne: "first test value",
				TestFieldTwo: "second test value",
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_column_one": "first test value",
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with two values and one is zero value should add only one where clause",
			reflect.ValueOf(struct {
				Metadata     Metadata `picard:"tablename=test_table"`
				TestFieldOne string   `picard:"column=test_column_one"`
				TestFieldTwo string   `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_column_one": "first test value",
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with two values and one is zero value and in zeroFields list should add both where clauses",
			reflect.ValueOf(struct {
				Metadata     Metadata `picard:"tablename=test_table"`
				TestFieldOne string   `picard:"column=test_column_one"`
				TestFieldTwo string   `picard:"column=test_column_two"`
			}{
				TestFieldOne: "first test value",
			}),
			[]string{"TestFieldTwo"},
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"test_table.test_column_one": "first test value",
				},
				squirrel.Eq{
					"test_table.test_column_two": "",
				},
			},
			[]string{},
			"",
		},
		{
			"Filter object with value for encrypted field should return error",
			reflect.ValueOf(struct {
				Metadata               Metadata `picard:"tablename=test_table"`
				TestMultitenancyColumn string   `picard:"multitenancy_key,column=test_multitenancy_column"`
				TestField              string   `picard:"encrypted,column=test_column_one"`
			}{
				TestField: "first test value",
			}),
			nil,
			nil,
			[]string{},
			"cannot perform queries with where clauses on encrypted fields",
		},
		{
			"Filter object with parent values",
			reflect.ValueOf(ChildTestObject{
				Parent: TestObject{
					Name: "blah",
				},
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"childtest.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t1.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t1.name": "blah",
				},
			},
			[]string{"testobject as t1 on t1.id = parent_id"},
			"",
		},
		{
			"Filter object with grandparent values",
			reflect.ValueOf(ChildTestObject{
				Parent: TestObject{
					Parent: ParentTestObject{
						Name: "ugh",
					},
				},
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"childtest.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t1.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t2.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t2.name": "ugh",
				},
			},
			[]string{"testobject as t1 on t1.id = parent_id", "parenttest as t2 on t2.id = t1.parent_id"},
			"",
		},
		{
			"Filter object with multiple parent values",
			reflect.ValueOf(ChildTestObject{
				Parent: TestObject{
					Name: "blah",
				},
				OptionalParent: TestObject{
					Name: "woo",
				},
			}),
			nil,
			[]squirrel.Sqlizer{
				squirrel.Eq{
					"childtest.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t1.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t1.name": "blah",
				},
				squirrel.Eq{
					"t2.organization_id": testMultitenancyValue,
				},
				squirrel.Eq{
					"t2.name": "woo",
				},
			},
			[]string{
				"testobject as t1 on t1.id = parent_id",
				"testobject as t2 on t2.id = optional_parent_id",
			},
			"",
		},
	}

	// Create the Picard instance
	p := PersistenceORM{
		multitenancyValue: testMultitenancyValue,
		performedBy:       testPerformedByValue,
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			filterModelType := tc.filterModelValue.Type()
			tableMetadata := tableMetadataFromType(filterModelType)
			whereClauses, joinClauses, err := p.generateWhereClausesFromModel(tc.filterModelValue, tc.zeroFields, tableMetadata)

			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantWhereClauses, whereClauses)
				assert.Equal(t, tc.wantJoinClauses, joinClauses)
			}
		})
	}
}
