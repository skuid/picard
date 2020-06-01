package picard

import (
	"database/sql/driver"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/testdata"
	"github.com/stretchr/testify/assert"
)

var parentObjectHelper = ExpectationHelper{
	FixtureType:      testdata.ParentTestObject{},
	LookupSelect:     "",
	LookupWhere:      "",
	LookupReturnCols: []string{},
	LookupFields:     []string{},
}

var testObjectHelper = ExpectationHelper{
	FixtureType:      testdata.TestObject{},
	LookupSelect:     "testobject.id, testobject.name as testobject_name, testobject.nullable_lookup as testobject_nullable_lookup",
	LookupWhere:      `COALESCE(testobject.name::"varchar",'') || '|' || COALESCE(testobject.nullable_lookup::"varchar",'')`,
	LookupReturnCols: []string{"id", "testobject_name", "testobject_nullable_lookup"},
	LookupFields:     []string{"Name", "NullableLookup"},
}

var testObjectWithPKHelper = ExpectationHelper{
	FixtureType:      testdata.TestObject{},
	LookupSelect:     "testobject.id, testobject.id as testobject_id",
	LookupWhere:      `COALESCE(testobject.id::"varchar",'')`,
	LookupReturnCols: []string{"id", "testobject_id"},
	LookupFields:     []string{"ID"},
}

var testChildObjectHelper = ExpectationHelper{
	FixtureType:      testdata.ChildTestObject{},
	LookupSelect:     "childtest.id, childtest.name as childtest_name, childtest.parent_id as childtest_parent_id",
	LookupWhere:      `COALESCE(childtest.name::"varchar",'') || '|' || COALESCE(childtest.parent_id::"varchar",'')`,
	LookupReturnCols: []string{"id", "childtest_name", "childtest_parent_id"},
	LookupFields:     []string{"Name", "ParentID"},
}

var testChildObjectWithLookupHelper = ExpectationHelper{
	FixtureType:      testdata.ChildTestObject{},
	LookupFrom:       "childtest JOIN testobject as t1 on t1.id = parent_id",
	LookupSelect:     "childtest.id, childtest.name as childtest_name, t1.name as t1_name, t1.nullable_lookup as t1_nullable_lookup",
	LookupWhere:      `COALESCE(childtest.name::"varchar",'') || '|' || COALESCE(t1.name::"varchar",'') || '|' || COALESCE(t1.nullable_lookup::"varchar",'')`,
	LookupReturnCols: []string{"id", "childtest_name", "t1_name", "t1_nullable_lookup"},
	LookupFields:     []string{"Name", "ParentID"},
}

var siblingJunctionHelper = ExpectationHelper{
	FixtureType:      testdata.SiblingJunctionModel{},
	LookupFrom:       "siblingjunction JOIN personmodel as t1 on t1.id = child_id JOIN personmodel as t2 on t2.id = sibling_id",
	LookupSelect:     "siblingjunction.id, t1.name as t1_name, t2.name as t2_name",
	LookupWhere:      `COALESCE(t1.name::"varchar",'') || '|' || COALESCE(t2.name::"varchar",'')`,
	LookupReturnCols: []string{"id", "t1_name", "t2_name"},
	LookupFields:     []string{"ChildID", "SiblingID"},
}

var siblingJunctionHelperWithChildKey = ExpectationHelper{
	FixtureType:      testdata.SiblingJunctionModel{},
	LookupFrom:       "siblingjunction JOIN personmodel as t1 on t1.id = sibling_id",
	LookupSelect:     "siblingjunction.id, siblingjunction.child_id as siblingjunction_child_id, t1.name as t1_name",
	LookupWhere:      `COALESCE(siblingjunction.child_id::"varchar",'') || '|' || COALESCE(t1.name::"varchar",'')`,
	LookupReturnCols: []string{"id", "child_id", "t1_name"},
	LookupFields:     []string{"ChildID", "SiblingID"},
}

var personModelHelper = ExpectationHelper{
	FixtureType:      testdata.PersonModel{},
	LookupSelect:     "personmodel.id, personmodel.name as personmodel_name",
	LookupWhere:      `COALESCE(personmodel.name::"varchar",'')`,
	LookupReturnCols: []string{"id", "personmodel_name"},
	LookupFields:     []string{"Name"},
}

var personModelWithIDHelper = ExpectationHelper{
	FixtureType:      testdata.PersonModel{},
	LookupSelect:     "personmodel.id, personmodel.id as personmodel_id",
	LookupWhere:      `COALESCE(personmodel.id::"varchar",'')`,
	LookupReturnCols: []string{"id", "personmodel_id"},
	LookupFields:     []string{"ID"},
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
				"column_one": testdata.TestSerializedObject{
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

	fixtures, err := LoadFixturesFromFiles(names, "./testdata/", reflect.TypeOf(structType), "")
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
		BatchSize           int
		ExpectationFunction func(*sqlmock.Sqlmock, interface{})
		WantErr             string
	}{
		{
			"Single Import with Primary Key with Nothing Existing",
			[]string{"SimpleWithPrimaryKey"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectWithPKHelper
				returnData := GetReturnDataForLookup(helper, nil)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectInsert(mock, helper, helper.GetInsertDBColumns(true), [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "ID"),
						sampleOrgID,
						helper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})
			},
			"",
		},
		{
			"Single Import with Primary Key That Already Exists",
			[]string{"SimpleWithPrimaryKey"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectWithPKHelper
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)
			},
			"",
		},
		{
			"Single Import with Nothing Existing",
			[]string{"Simple"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectHelper
				returnData := GetReturnDataForLookup(helper, nil)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectInsert(mock, helper, helper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						helper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						nil,
						helper.GetFixtureValue(fixtures, 0, "Config"),
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})
			},
			"",
		},
		{
			"Single Import with That Already Exists",
			[]string{"Simple"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectHelper
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						helper.GetFixtureValue(fixtures, 0, "Config"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)
			},
			"",
		},
		{
			"Single Import Missing Required Field",
			[]string{"Empty"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {},
			"Key: 'TestObject.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
		{
			"Single Import with Null Matches Existing value with a Null lookup",
			[]string{"Simple"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectHelper
				returnData := [][]driver.Value{
					[]driver.Value{
						uuid.NewV4().String(),
						"Simple",
						nil,
					},
				}
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						helper.GetFixtureValue(fixtures, 0, "Config"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)
			},
			"",
		},
		{
			"Multiple Import with Nothing Existing",
			[]string{"Simple", "Simple2"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectHelper
				returnData := GetReturnDataForLookup(helper, nil)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectInsert(mock, helper, helper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						helper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						nil,
						helper.GetFixtureValue(fixtures, 0, "Config"),
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
					[]driver.Value{
						sampleOrgID,
						helper.GetFixtureValue(fixtures, 1, "Name"),
						nil,
						helper.GetFixtureValue(fixtures, 1, "Type"),
						nil,
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})
			},
			"",
		},
		{
			"Multiple Import with Both Already Exist",
			[]string{"Simple", "Simple2"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				helper := testObjectHelper
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
					helper.GetUpdateDBColumnsForFixture(fixtures, 1),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						helper.GetFixtureValue(fixtures, 0, "Config"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 1, "Name"),
						helper.GetFixtureValue(fixtures, 1, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)
			},
			"",
		},
		{
			"Multiple Import with One Already Exists",
			[]string{"Simple", "Simple2"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObject)
				returnData := GetReturnDataForLookup(helper, []testdata.TestObject{
					fixtures[0],
				})
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						helper.GetFixtureValue(fixtures, 0, "IsActive"),
						helper.GetFixtureValue(fixtures, 0, "Config"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)
				ExpectInsert(mock, helper, helper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						helper.GetFixtureValue(fixtures, 1, "Name"),
						nil,
						helper.GetFixtureValue(fixtures, 1, "Type"),
						nil,
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})
			},
			"",
		},
		{
			"Single Import with GrandChildren All Inserts",
			[]string{"SimpleWithGrandChildren"},
			testdata.ParentTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]testdata.ParentTestObject)
				insertRows := ExpectInsert(mock, parentObjectHelper, parentObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						parentObjectHelper.GetFixtureValue(fixtures, 0, "Name"),
					},
				})

				testObjects := []testdata.TestObject{}
				for index, fixture := range fixtures {
					for _, testObject := range fixture.Children {
						testObject.ParentID = insertRows[index][0].(string)
						testObjects = append(testObjects, testObject)
					}
				}

				testReturnData := GetReturnDataForLookup(testObjectHelper, nil)
				testLookupKeys := GetLookupKeys(testObjectHelper, testObjects)
				ExpectLookup(mock, testObjectHelper, testLookupKeys, testReturnData)

				childInsertRows := ExpectInsert(mock, testObjectHelper, testObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testObjectHelper.GetFixtureValue(testObjects, 0, "Name"),
						nil,
						testObjectHelper.GetFixtureValue(testObjects, 0, "Type"),
						nil,
						testChildObjectHelper.GetFixtureValue(testObjects, 0, "ParentID"),
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})

				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children[0].Children {
						childObject.ParentID = childInsertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "ParentID"),
						nil,
					},
				})
			},
			"",
		},
		{
			"Single Import with Children Insert New Parent",
			[]string{"SimpleWithChildren"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]testdata.TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, testObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Type"),
						nil,
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})

				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "ParentID"),
						nil,
					},
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "ParentID"),
						nil,
					},
				})
			},
			"",
		},
		{
			"Single Import with Children Insert New Parent Small Batch Size",
			[]string{"SimpleWithChildren"},
			testdata.TestObject{},
			1,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]testdata.TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, testObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Type"),
						nil,
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})

				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeysBatch1 := GetLookupKeys(testChildObjectHelper, []testdata.ChildTestObject{
					childObjects[0],
				})
				ExpectLookup(mock, testChildObjectHelper, childLookupKeysBatch1, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "ParentID"),
						nil,
					},
				})
				childLookupKeysBatch2 := GetLookupKeys(testChildObjectHelper, []testdata.ChildTestObject{
					childObjects[1],
				})
				ExpectLookup(mock, testChildObjectHelper, childLookupKeysBatch2, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "ParentID"),
						nil,
					},
				})
			},
			"",
		},
		{
			"Single Import with Children Insert New Parent And Orphans",
			[]string{"SimpleWithChildren"},
			testdata.TestObjectWithOrphans{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]testdata.TestObjectWithOrphans)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, testObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Type"),
						nil,
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})

				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "ParentID"),
						nil,
					},
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						nil,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "ParentID"),
						nil,
					},
				})
			},
			"",
		},
		{
			"Single Import with ChildrenMap Insert New Parent",
			[]string{"SimpleWithChildrenMap"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]testdata.TestObject)
				returnData := GetReturnDataForLookup(testObjectHelper, nil)
				lookupKeys := GetLookupKeys(testObjectHelper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				insertRows := ExpectInsert(mock, testObjectHelper, testObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Name"),
						nil,
						testObjectHelper.GetFixtureValue(fixtures, 0, "Type"),
						nil,
						nil,
						nil,
						sampleUserID,
						sampleUserID,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
					},
				})
				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.ChildrenMap {
						childObject.ParentID = insertRows[index][0].(string)
						childObject.Name = "ChildRecord1"
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						// Tests that the key mapping "Name" worked correctly
						"ChildRecord1",
						// Tests that the value mapping "Type->OtherInfo" worked correctly
						testObjectHelper.GetFixtureValue(fixtures, 0, "Type"),
						testObjectHelper.GetReturnDataKey(insertRows, 0),
						nil,
					},
				})
			},
			"",
		},
		{
			"Single Import with Children Existing Parent",
			[]string{"SimpleWithChildren"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObject)
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)

				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = returnData[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
						nil,
					},
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						nil,
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
						nil,
					},
				})
			},
			"",
		},
		{
			"Single Import with Children Existing Parent With Orphans",
			[]string{"SimpleWithChildren"},
			testdata.TestObjectWithOrphans{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObjectWithOrphans)
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, testObjectHelper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)

				parentIDs := []string{}
				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					parentID := returnData[index][0].(string)
					parentIDs = append(parentIDs, parentID)
					for _, childObject := range fixture.Children {
						childObject.ParentID = parentID
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)

				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
						nil,
					},
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						nil,
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
						nil,
					},
				})

				// Expect the lookup to find orphans to delete for the first child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000002", parentIDs[0]).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000003", parentIDs[0]).
							AddRow("ChildRecord2", "00000000-0000-0000-0000-000000000004", parentIDs[0]),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})
			},
			"",
		},
		{
			"Single Import with Children Existing Parent With Orphans And Empty Children Map",
			[]string{"SimpleWithChildrenAndChildrenMap"},
			testdata.TestObjectWithOrphans{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObjectWithOrphans)
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)

				parentIDs := []string{}
				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					parentID := returnData[index][0].(string)
					parentIDs = append(parentIDs, parentID)
					for _, childObject := range fixture.Children {
						childObject.ParentID = parentID
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, nil)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectInsert(mock, testChildObjectHelper, testChildObjectHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
						nil,
					},
					[]driver.Value{
						sampleOrgID,
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						nil,
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
						nil,
					},
				})

				// Expect the lookup to find orphans to delete for the first child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id"}).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000001").
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000002"),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})

				// Expect the lookup to find orphans to delete for the second child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id"}).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000001").
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000002"),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})
			},
			"",
		},
		{
			"Single Import with Children Existing Parent and Existing Child",
			[]string{"SimpleWithChildren"},
			testdata.TestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObject)
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)

				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Children {
						childObject.ParentID = returnData[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, childObjects)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)
				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 0),
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 1),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
					},
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
					},
				}, childReturnData)
			},
			"",
		},
		{
			"Single Import with Children Existing Parent and Existing Child With Orphans",
			[]string{"SimpleWithChildrenAndChildrenMap"},
			testdata.TestObjectWithOrphans{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObjectWithOrphans)
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)

				parentIDs := []string{}
				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					parentID := returnData[index][0].(string)
					parentIDs = append(parentIDs, parentID)
					for _, childObject := range fixture.Children {
						childObject.ParentID = parentID
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, childObjects)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)

				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 0),
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 1),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
					},
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
					},
				}, childReturnData)

				// Expect the lookup to find orphans to delete for the first child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000002", parentIDs[0]),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000002"})

				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000002", parentIDs[0]),
					)

				// Expect the lookup to find orphans to delete for the second child field
				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})
			},
			"",
		},
		{
			"Multiple Import with Children Existing Parent and Existing Child With Orphans",
			[]string{"SimpleWithChildrenAndChildrenMap", "SimpleWithChildren2"},
			testdata.TestObjectWithOrphans{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObjectWithOrphans)
				returnData := GetReturnDataForLookup(helper, fixtures)
				lookupKeys := GetLookupKeys(helper, fixtures)
				ExpectLookup(mock, helper, lookupKeys, returnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(fixtures, 0),
					helper.GetUpdateDBColumnsForFixture(fixtures, 1),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 0, "Name"),
						helper.GetFixtureValue(fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
					[]driver.Value{
						helper.GetFixtureValue(fixtures, 1, "Name"),
						helper.GetFixtureValue(fixtures, 1, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, returnData)

				parentIDs := []string{}
				childObjects := []testdata.ChildTestObject{}
				for index, fixture := range fixtures {
					parentID := returnData[index][0].(string)
					parentIDs = append(parentIDs, parentID)
					for _, childObject := range fixture.Children {
						childObject.ParentID = parentID
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := GetReturnDataForLookup(testChildObjectHelper, childObjects)
				childLookupKeys := GetLookupKeys(testChildObjectHelper, childObjects)
				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeys, childReturnData)

				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 0),
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 1),
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 2),
					testChildObjectHelper.GetUpdateDBColumnsForFixture(childObjects, 3),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
					},
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 1, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 0),
					},
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 2, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 1),
					},
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(childObjects, 3, "Name"),
						testChildObjectHelper.GetReturnDataKey(returnData, 1),
					},
				}, childReturnData)

				// Expect the lookup to find orphans to delete for the first child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE
							t0.organization_id = $1 AND
							((t0.parent_id = $2) OR (t0.parent_id = $3))
					`)).
					WithArgs(sampleOrgID, parentIDs[0], parentIDs[1]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("ChildRecord2", "00000000-0000-0000-0000-000000000002", parentIDs[0]).
							AddRow("ChildRecord3", "00000000-0000-0000-0000-000000000003", parentIDs[1]).
							// Match on name, but not parent id, still should delete
							AddRow("ChildRecord4", "00000000-0000-0000-0000-000000000004", parentIDs[0]).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000005", parentIDs[0]).
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000006", parentIDs[0]),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000006"})

				// Expect the lookup to find orphans to delete for the second child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
						SELECT
							t0.id AS "t0.id",
							t0.organization_id AS "t0.organization_id",
							t0.name AS "t0.name",
							t0.other_info AS "t0.other_info",
							t0.parent_id AS "t0.parent_id",
							t0.optional_parent_id AS "t0.optional_parent_id"
						FROM childtest AS t0
						WHERE
							t0.organization_id = $1 AND ((t0.parent_id = $2))
					`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000002", parentIDs[0]),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})
			},
			"",
		},
		{
			"Multiple Import with Children Existing Parent and Existing Child With Orphans Small Batch",
			[]string{"SimpleWithChildrenAndChildrenMap", "SimpleWithChildren2"},
			testdata.TestObjectWithOrphans{},
			1,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				helper := testObjectHelper
				fixtures := fixturesAbstract.([]testdata.TestObjectWithOrphans)
				batch1Fixtures := []testdata.TestObjectWithOrphans{
					fixtures[0],
				}
				batch2Fixtures := []testdata.TestObjectWithOrphans{
					fixtures[1],
				}
				batch1ReturnData := GetReturnDataForLookup(helper, batch1Fixtures)
				batch1LookupKeys := GetLookupKeys(helper, batch1Fixtures)
				batch2ReturnData := GetReturnDataForLookup(helper, batch2Fixtures)
				batch2LookupKeys := GetLookupKeys(helper, batch2Fixtures)

				parentIDs := []string{}
				childObjects := []testdata.ChildTestObject{}

				ExpectLookup(mock, helper, batch1LookupKeys, batch1ReturnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(batch1Fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(batch1Fixtures, 0, "Name"),
						helper.GetFixtureValue(batch1Fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, batch1ReturnData)

				ExpectLookup(mock, helper, batch2LookupKeys, batch2ReturnData)
				ExpectUpdate(mock, helper, [][]string{
					helper.GetUpdateDBColumnsForFixture(batch2Fixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						helper.GetFixtureValue(batch2Fixtures, 0, "Name"),
						helper.GetFixtureValue(batch2Fixtures, 0, "Type"),
						sampleUserID,
						sqlmock.AnyArg(),
					},
				}, batch2ReturnData)

				for index, fixture := range batch1Fixtures {
					parentID := batch1ReturnData[index][0].(string)
					parentIDs = append(parentIDs, parentID)
					for _, childObject := range fixture.Children {
						childObject.ParentID = parentID
						childObjects = append(childObjects, childObject)
					}
				}

				for index, fixture := range batch2Fixtures {
					parentID := batch2ReturnData[index][0].(string)
					parentIDs = append(parentIDs, parentID)
					for _, childObject := range fixture.Children {
						childObject.ParentID = parentID
						childObjects = append(childObjects, childObject)
					}
				}

				batch1ChildFixtures := []testdata.ChildTestObject{
					childObjects[0],
				}
				batch2ChildFixtures := []testdata.ChildTestObject{
					childObjects[1],
				}
				batch3ChildFixtures := []testdata.ChildTestObject{
					childObjects[2],
				}
				batch4ChildFixtures := []testdata.ChildTestObject{
					childObjects[3],
				}

				childReturnDataBatch1 := GetReturnDataForLookup(testChildObjectHelper, batch1ChildFixtures)
				childLookupKeysBatch1 := GetLookupKeys(testChildObjectHelper, batch1ChildFixtures)
				childReturnDataBatch2 := GetReturnDataForLookup(testChildObjectHelper, batch2ChildFixtures)
				childLookupKeysBatch2 := GetLookupKeys(testChildObjectHelper, batch2ChildFixtures)
				childReturnDataBatch3 := GetReturnDataForLookup(testChildObjectHelper, batch3ChildFixtures)
				childLookupKeysBatch3 := GetLookupKeys(testChildObjectHelper, batch3ChildFixtures)
				childReturnDataBatch4 := GetReturnDataForLookup(testChildObjectHelper, batch4ChildFixtures)
				childLookupKeysBatch4 := GetLookupKeys(testChildObjectHelper, batch4ChildFixtures)

				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeysBatch1, childReturnDataBatch1)

				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(batch1ChildFixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(batch1ChildFixtures, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(batch1ReturnData, 0),
					},
				}, childReturnDataBatch1)

				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeysBatch2, childReturnDataBatch2)

				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(batch2ChildFixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(batch2ChildFixtures, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(batch1ReturnData, 0),
					},
				}, childReturnDataBatch2)

				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeysBatch3, childReturnDataBatch3)

				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(batch3ChildFixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(batch3ChildFixtures, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(batch2ReturnData, 0),
					},
				}, childReturnDataBatch3)

				// Expect the normal lookup
				ExpectLookup(mock, testChildObjectHelper, childLookupKeysBatch4, childReturnDataBatch4)

				ExpectUpdate(mock, testChildObjectHelper, [][]string{
					testChildObjectHelper.GetUpdateDBColumnsForFixture(batch4ChildFixtures, 0),
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectHelper.GetFixtureValue(batch4ChildFixtures, 0, "Name"),
						testChildObjectHelper.GetReturnDataKey(batch2ReturnData, 0),
					},
				}, childReturnDataBatch4)

				// Expect the lookup to find orphans to delete for the first child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
							SELECT
								t0.id AS "t0.id",
								t0.organization_id AS "t0.organization_id",
								t0.name AS "t0.name",
								t0.other_info AS "t0.other_info",
								t0.parent_id AS "t0.parent_id",
								t0.optional_parent_id AS "t0.optional_parent_id"
							FROM childtest AS t0
							WHERE
								t0.organization_id = $1 AND ((t0.parent_id = $2) OR (t0.parent_id = $3))
						`)).
					WithArgs(sampleOrgID, parentIDs[0], parentIDs[1]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("ChildRecord2", "00000000-0000-0000-0000-000000000002", parentIDs[0]).
							AddRow("ChildRecord3", "00000000-0000-0000-0000-000000000003", parentIDs[1]).
							// Match on name, but not parent id, still should delete
							AddRow("ChildRecord4", "00000000-0000-0000-0000-000000000004", parentIDs[0]).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000005", parentIDs[0]).
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000006", parentIDs[0]),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000006"})

				// Expect the lookup to find orphans to delete for the second child field
				ExpectQuery(mock, testdata.FmtSQLRegex(`
							SELECT
								t0.id AS "t0.id",
								t0.organization_id AS "t0.organization_id",
								t0.name AS "t0.name",
								t0.other_info AS "t0.other_info",
								t0.parent_id AS "t0.parent_id",
								t0.optional_parent_id AS "t0.optional_parent_id"
							FROM childtest AS t0
							WHERE
								t0.organization_id = $1 AND ((t0.parent_id = $2))
						`)).
					WithArgs(sampleOrgID, parentIDs[0]).
					WillReturnRows(
						sqlmock.NewRows([]string{"t0.name", "t0.id", "t0.parent_id"}).
							AddRow("Orphan1", "00000000-0000-0000-0000-000000000001", parentIDs[0]).
							AddRow("Orphan2", "00000000-0000-0000-0000-000000000002", parentIDs[0]),
					)

				ExpectDelete(mock, testChildObjectHelper, []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002"})
			},
			"",
		},
		{
			"Import Existing Child with Reference to Parent Name",
			[]string{"ChildWithParentLookup"},
			testdata.ChildTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				childUUID := uuid.NewV4().String()
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObject)
				returnData := [][]driver.Value{
					[]driver.Value{
						childUUID,
						"ChildItem",
						"Simple",
						"",
					},
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, []string{"ChildItem|Simple|"}, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{
					[]driver.Value{
						parentUUID,
						"Simple",
						"",
					},
				})

				ExpectUpdate(mock, testChildObjectWithLookupHelper, [][]string{
					[]string{
						"name",
						"parent_id",
					},
				}, [][]driver.Value{
					[]driver.Value{
						testChildObjectWithLookupHelper.GetFixtureValue(fixtures, 0, "Name"),
						parentUUID,
					},
				}, returnData)
			},
			"",
		},
		{
			"Import New Child with Reference to Parent Name",
			[]string{"ChildWithParentLookup"},
			testdata.ChildTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []testdata.ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, testdata.ChildTestObject{
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

				ExpectInsert(mock, testChildObjectWithLookupHelper, testChildObjectWithLookupHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectWithLookupHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						parentUUID,
						nil,
					},
				})
			},
			"",
		},
		{
			"Import New Child with Reference to Parent Name Using Key Map",
			[]string{"ChildWithParentLookupAndKeyMap"},
			testdata.ChildTestObjectWithKeyMap{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObjectWithKeyMap)
				lookupKeys := []string{"ChildItem|Simple"}
				returnData := [][]driver.Value{}

				childObjects := []testdata.ChildTestObjectWithKeyMap{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, testdata.ChildTestObjectWithKeyMap{
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
				ExpectInsert(mock, testChildObjectWithLookupHelper, testChildObjectWithLookupHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectWithLookupHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						parentUUID,
						nil,
					},
				})
			},
			"",
		},

		{
			"Import New Child with Reference to Parent Name And Optional Parent",
			[]string{"ChildWithParentLookupAndOptionalLookup"},
			testdata.ChildTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				optionalParentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []testdata.ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, testdata.ChildTestObject{
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
				ExpectInsert(mock, testChildObjectWithLookupHelper, testChildObjectWithLookupHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						testChildObjectWithLookupHelper.GetFixtureValue(childObjects, 0, "Name"),
						nil,
						parentUUID,
						optionalParentUUID,
					},
				})
			},
			"",
		},

		{
			"Import New Child with Bad Reference to Parent Name",
			[]string{"ChildWithParentLookup"},
			testdata.ChildTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []testdata.ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, testdata.ChildTestObject{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{})
			},
			"Missing Required Foreign Key Lookup: Table 'childtest', Foreign Key 'parent_id', Key 'Simple|'",
		},

		{
			"Import New Children with Bad References to Parent Name",
			[]string{"ChildWithParentLookup", "ChildWithParentLookup2"},
			testdata.ChildTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|", "ChildItem2|Simple2|"}
				returnData := [][]driver.Value{}

				childObjects := []testdata.ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, testdata.ChildTestObject{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|", "Simple2|"}, [][]driver.Value{})
			},
			"Missing Required Foreign Key Lookup: Table 'childtest', Foreign Key 'parent_id', Key 'Simple|' - Missing Required Foreign Key Lookup: Table 'childtest', Foreign Key 'parent_id', Key 'Simple2|'",
		},

		{
			"Import New Children with Bad References to same Parent Name",
			[]string{"ChildWithParentLookup", "ChildWithParentLookup"},
			testdata.ChildTestObject{},
			100,
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				parentUUID := uuid.NewV4().String()
				fixtures := fixturesAbstract.([]testdata.ChildTestObject)
				lookupKeys := []string{"ChildItem|Simple|"}
				returnData := [][]driver.Value{}

				childObjects := []testdata.ChildTestObject{}
				for _, fixture := range fixtures {
					childObjects = append(childObjects, testdata.ChildTestObject{
						Name:     fixture.Name,
						ParentID: parentUUID,
					})
				}

				ExpectLookup(mock, testChildObjectWithLookupHelper, lookupKeys, returnData)

				// Expect the foreign key lookup next
				ExpectLookup(mock, testObjectHelper, []string{"Simple|"}, [][]driver.Value{})
			},
			"Missing Required Foreign Key Lookup: Table 'childtest', Foreign Key 'parent_id', Key 'Simple|'",
		},
		{
			"Single Import with Multiple Foreign Key Lookups",
			[]string{"Sibling"},
			testdata.SiblingJunctionModel{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				junctionHelper := siblingJunctionHelper
				personHelper := personModelHelper
				junctionReturnData := GetReturnDataForLookup(junctionHelper, nil)
				personReturnData1 := GetReturnDataForLookup(personHelper, []testdata.PersonModel{
					{
						Name: "George",
					},
				})
				personReturnData2 := GetReturnDataForLookup(personHelper, []testdata.PersonModel{
					{
						Name: "Fred",
					},
				})
				junctionLookupKeys := []string{"George|Fred"}

				ExpectLookup(mock, junctionHelper, junctionLookupKeys, junctionReturnData)

				ExpectLookup(mock, personHelper, []string{"George"}, personReturnData1)
				ExpectLookup(mock, personHelper, []string{"Fred"}, personReturnData2)

				ExpectInsert(mock, junctionHelper, junctionHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						personReturnData1[0][0],
						personReturnData2[0][0],
					},
				})
			},
			"",
		},

		{
			"Single Import with Multiple Foreign Key Lookups One has key provided nested",
			[]string{"SiblingWithChildNestedKeyProvided"},
			testdata.SiblingJunctionModel{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				junctionHelper := siblingJunctionHelper
				personHelper := personModelHelper
				personWithPKHelper := personModelWithIDHelper
				junctionReturnData := GetReturnDataForLookup(junctionHelper, nil)

				personReturnData1 := [][]driver.Value{
					[]driver.Value{
						"00000000-0000-0000-0000-000000C0FFEE",
						"00000000-0000-0000-0000-000000C0FFEE",
					},
				}

				personReturnData2 := GetReturnDataForLookup(personHelper, []testdata.PersonModel{
					{
						Name: "Fred",
					},
				})
				junctionLookupKeys := []string{"|Fred"}

				ExpectLookup(mock, junctionHelper, junctionLookupKeys, junctionReturnData)

				ExpectLookup(mock, personWithPKHelper, []string{"00000000-0000-0000-0000-000000C0FFEE"}, personReturnData1)
				ExpectLookup(mock, personHelper, []string{"Fred"}, personReturnData2)

				ExpectInsert(mock, junctionHelper, junctionHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						personReturnData1[0][0],
						personReturnData2[0][0],
					},
				})
			},
			"",
		},

		{
			"Single Import with Multiple Foreign Key Lookups One has key provided not nested",
			[]string{"SiblingWithChildKeyProvided"},
			testdata.SiblingJunctionModel{},
			100,
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				junctionHelper := siblingJunctionHelperWithChildKey
				personHelper := personModelHelper
				junctionReturnData := GetReturnDataForLookup(junctionHelper, nil)

				personReturnData1 := GetReturnDataForLookup(personHelper, []testdata.PersonModel{
					{
						Name: "Fred",
					},
				})

				junctionLookupKeys := []string{"00000000-0000-0000-0000-000000C0FFEE|Fred"}

				ExpectLookup(mock, junctionHelper, junctionLookupKeys, junctionReturnData)

				ExpectLookup(mock, personHelper, []string{"Fred"}, personReturnData1)

				ExpectInsert(mock, junctionHelper, junctionHelper.GetInsertDBColumns(false), [][]driver.Value{
					[]driver.Value{
						sampleOrgID,
						"00000000-0000-0000-0000-000000C0FFEE",
						personReturnData1[0][0],
					},
				})

			},
			"",
		},
	}

	// Run first with the default batch size
	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			fixtures, err := loadTestObjects(c.FixtureNames, c.FixtureType)
			if err != nil {
				t.Fatal(err)
			}

			err = RunImportTest(fixtures, c.ExpectationFunction, c.BatchSize)

			if c.WantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, c.WantErr)
			}
		})
	}
}

type Item struct {
	metadata.Metadata `picard:"tablename=test_tablename"`

	PrimaryKeyField        string `picard:"primary_key,column=primary_key_column"`
	TestMultitenancyColumn string `picard:"multitenancy_key,column=multitenancy_key_column"`
	TestFieldOne           string `picard:"column=test_column_one"`
}

func TestStartTransaction(t *testing.T) {
	testCases := []struct {
		testDescription string
		methodFunc      func(ORM) error
		expectationFunc func(ORM, sqlmock.Sqlmock)
	}{
		{
			"ORM methods should use an existing transaction set in StartTransaction",
			func(orm ORM) error {
				_, err := orm.StartTransaction()
				newModel := &Item{
					TestFieldOne: "kayak",
				}
				err = orm.CreateModel(newModel)
				if err != nil {
					return err
				}

				newModel.TestFieldOne = "canoe"
				err = orm.SaveModel(newModel)
				if err != nil {
					return err
				}

				deleteModel := Item{
					PrimaryKeyField: "00000000-0000-0000-0000-000000000555",
				}
				_, err = orm.DeleteModel(deleteModel)
				if err != nil {
					return err
				}

				deployModels := []Item{
					Item{
						TestFieldOne: "ice",
					},
					Item{
						TestFieldOne: "snow",
					},
				}
				err = orm.Deploy(deployModels)
				if err != nil {
					return err
				}

				orm.Commit()

				return nil
			},
			func(orm ORM, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				// createModel
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "kayak").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)

				// saveModel
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET test_column_one = \$1 WHERE multitenancy_key_column = \$2 AND primary_key_column = \$3$`).
					WithArgs("canoe", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))

				// deleteModel
				mock.ExpectExec(testdata.FmtSQLRegex(`
					DELETE FROM test_tablename AS t0
					WHERE
						t0.multitenancy_key_column = $1 AND
						t0.primary_key_column = $2
				`)).
					WithArgs(
						"00000000-0000-0000-0000-000000000005",
						"00000000-0000-0000-0000-000000000555",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))

				// deployModel
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\),\(\$3,\$4\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "ice", "00000000-0000-0000-0000-000000000005", "snow").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).
							AddRow("00000000-0000-0000-0000-000000000001").
							AddRow("00000000-0000-0000-0000-000000000002"),
					)

				mock.ExpectCommit()
			},
		},
		{
			"ORM methods should begin a new transaction if there has been a rollback",
			func(orm ORM) error {
				_, err := orm.StartTransaction()
				newModel := &Item{
					TestFieldOne: "kayak",
				}
				err = orm.CreateModel(newModel)
				if err != nil {
					return err
				}

				orm.Rollback()
				_, err = orm.StartTransaction()
				if err != nil {
					return err
				}

				newModel.TestFieldOne = "canoe"
				err = orm.SaveModel(newModel)
				if err != nil {
					return err
				}
				orm.Rollback()
				_, err = orm.StartTransaction()
				if err != nil {
					return err
				}

				deleteModel := Item{
					PrimaryKeyField: "00000000-0000-0000-0000-000000000555",
				}
				_, err = orm.DeleteModel(deleteModel)
				if err != nil {
					return err
				}
				orm.Rollback()
				_, err = orm.StartTransaction()
				if err != nil {
					return err
				}

				deployModels := []Item{
					Item{
						TestFieldOne: "ice",
					},
					Item{
						TestFieldOne: "snow",
					},
				}
				err = orm.Deploy(deployModels)
				if err != nil {
					return err
				}
				orm.Rollback()

				return nil
			},
			func(orm ORM, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				// createModel
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "kayak").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)

				mock.ExpectRollback()
				mock.ExpectBegin()

				// saveModel
				mock.ExpectQuery(`^SELECT test_tablename.primary_key_column FROM test_tablename WHERE test_tablename.primary_key_column = \$1 AND test_tablename.multitenancy_key_column = \$2$`).
					WithArgs("00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000005").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).AddRow("00000000-0000-0000-0000-000000000001"),
					)
				mock.ExpectExec(`^UPDATE test_tablename SET test_column_one = \$1 WHERE multitenancy_key_column = \$2 AND primary_key_column = \$3$`).
					WithArgs("canoe", "00000000-0000-0000-0000-000000000005", "00000000-0000-0000-0000-000000000001").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectRollback()
				mock.ExpectBegin()

				// deleteModel
				mock.ExpectExec(testdata.FmtSQLRegex(`
						DELETE FROM test_tablename AS t0
						WHERE
							t0.multitenancy_key_column = $1 AND
							t0.primary_key_column = $2
					`)).
					WithArgs(
						"00000000-0000-0000-0000-000000000005",
						"00000000-0000-0000-0000-000000000555",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectRollback()
				mock.ExpectBegin()
				// deployModel
				mock.ExpectQuery(`^INSERT INTO test_tablename \(multitenancy_key_column,test_column_one\) VALUES \(\$1,\$2\),\(\$3,\$4\) RETURNING "primary_key_column"$`).
					WithArgs("00000000-0000-0000-0000-000000000005", "ice", "00000000-0000-0000-0000-000000000005", "snow").
					WillReturnRows(
						sqlmock.NewRows([]string{"primary_key_column"}).
							AddRow("00000000-0000-0000-0000-000000000001").
							AddRow("00000000-0000-0000-0000-000000000002"),
					)

				mock.ExpectRollback()
			},
		},
		{
			"StartTransaction shouldn't start another transaction if there is an existing one",
			func(orm ORM) error {
				orm.StartTransaction()
				orm.StartTransaction()
				orm.Commit()
				return nil
			},
			func(orm ORM, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
		},
		{
			"Commit shouldn't commit if transaction already finished",
			func(orm ORM) error {
				orm.StartTransaction()
				orm.Commit()
				orm.Commit()
				return nil
			},
			func(orm ORM, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
		},
		{
			"Rollback shouldn't commit if transaction already finished",
			func(orm ORM) error {
				orm.StartTransaction()
				orm.Rollback()
				orm.Rollback()
				return nil
			},
			func(orm ORM, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testDescription, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
			}
			SetConnection(db)
			defer CloseConnection()
			orm := &PersistenceORM{
				multitenancyValue: "00000000-0000-0000-0000-000000000005",
				performedBy:       "00000000-0000-0000-0000-000000000006",
				batchSize:         2,
			}
			tc.expectationFunc(orm, mock)
			assert.NoError(t, err)

			err = tc.methodFunc(orm)
			assert.NoError(t, err)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unmet sqlmock expectations: %s", err)
			}
		})
	}
}
