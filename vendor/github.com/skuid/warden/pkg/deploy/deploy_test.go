package deploy

import (
	"fmt"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/ds"
)

var sampleOrgID = "6ba7b810-9dbd-11d1-80b4-00c04fd430c8"
var sampleUserID = "72c431ec-14ed-4d77-9948-cb92e816a3a7"

var dataSourceHelper = picard.ExpectationHelper{
	FixtureType:      ds.DataSourceNew{},
	LookupSelect:     "data_source.id, data_source.name as data_source_name",
	LookupWhere:      `COALESCE(data_source.name::"varchar",'')`,
	LookupReturnCols: []string{"id", "data_source_name"},
	LookupFields:     []string{"Name"},
}

var entityHelper = picard.ExpectationHelper{
	FixtureType:      ds.EntityNew{},
	LookupSelect:     "data_source_object.id, data_source_object.name as data_source_object_name, data_source_object.data_source_id as data_source_object_data_source_id",
	LookupWhere:      `COALESCE(data_source_object.name::"varchar",'') || '|' || COALESCE(data_source_object.data_source_id::"varchar",'')`,
	LookupReturnCols: []string{"id", "data_source_object_name", "data_source_object_data_source_id"},
	LookupFields:     []string{"Name", "DataSourceID"},
}

// Loads in a fixture data source from file
func loadTestDataSources(names []string) ([]ds.DataSourceNew, error) {

	fixtures, err := picard.LoadFixturesFromFiles(names, "./testdata/datasources/", reflect.TypeOf(ds.DataSourceNew{}))
	if err != nil {
		return nil, err
	}

	return fixtures.([]ds.DataSourceNew), nil

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
				returnData := picard.GetReturnDataForLookup(dataSourceHelper, nil)
				lookupKeys := picard.GetLookupKeys(dataSourceHelper, fixtures)
				picard.ExpectLookup(mock, dataSourceHelper, lookupKeys, returnData)
				picard.ExpectInsert(mock, dataSourceHelper, fixtures, false)
			},
		},

		{
			"Single Import with That Already Exists",
			[]string{"Simple"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := picard.GetReturnDataForLookup(dataSourceHelper, fixtures)
				lookupKeys := picard.GetLookupKeys(dataSourceHelper, fixtures)
				parentID := returnData[0][0]
				sampleOrgID := "6ba7b810-9dbd-11d1-80b4-00c04fd430c8"
				picard.ExpectLookup(mock, dataSourceHelper, lookupKeys, returnData)
				picard.ExpectUpdate(mock, dataSourceHelper, fixtures, returnData)
				// Orphan Removal Query
				picard.ExpectQuery(mock, `
						^SELECT data_source_object.id, data_source_object.organization_id, data_source_object.data_source_id, data_source_object.name, data_source_object.schema, data_source_object.label, data_source_object.label_plural, data_source_object.created_by_id, data_source_object.updated_by_id, data_source_object.created_at, data_source_object.updated_at
						FROM data_source_object
						WHERE \(\(data_source_object.organization_id = \$1 AND data_source_object.data_source_id = \$2\)\)$
					`).
					WithArgs(sampleOrgID, parentID).
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "data_source_id"}).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000001", parentID),
					)
				picard.ExpectDelete(mock, entityHelper, []string{"00000000-0000-0000-0000-000000000001"})

			},
		},
		{
			"Multiple Import with Nothing Existing",
			[]string{"Simple", "Simple2"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := picard.GetReturnDataForLookup(dataSourceHelper, nil)
				lookupKeys := picard.GetLookupKeys(dataSourceHelper, fixtures)
				picard.ExpectLookup(mock, dataSourceHelper, lookupKeys, returnData)
				picard.ExpectInsert(mock, dataSourceHelper, fixtures, false)
			},
		},
		{
			"Multiple Import with Both Already Exist",
			[]string{"Simple", "Simple2"},
			func(mock *sqlmock.Sqlmock, fixtures interface{}) {
				returnData := picard.GetReturnDataForLookup(dataSourceHelper, fixtures)
				lookupKeys := picard.GetLookupKeys(dataSourceHelper, fixtures)
				parentID1 := returnData[0][0]
				parentID2 := returnData[1][0]
				sampleOrgID := "6ba7b810-9dbd-11d1-80b4-00c04fd430c8"
				picard.ExpectLookup(mock, dataSourceHelper, lookupKeys, returnData)
				picard.ExpectUpdate(mock, dataSourceHelper, fixtures, returnData)
				// Orphan Removal Query
				picard.ExpectQuery(mock, `
						^SELECT data_source_object.id, data_source_object.organization_id, data_source_object.data_source_id, data_source_object.name, data_source_object.schema, data_source_object.label, data_source_object.label_plural, data_source_object.created_by_id, data_source_object.updated_by_id, data_source_object.created_at, data_source_object.updated_at
						FROM data_source_object
						WHERE \(\(data_source_object.organization_id = \$1 AND data_source_object.data_source_id = \$2\) OR \(data_source_object.organization_id = \$3 AND data_source_object.data_source_id = \$4\)\)$
					`).
					WithArgs(sampleOrgID, parentID1, sampleOrgID, parentID2).
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "data_source_id"}).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000001", parentID1),
					)
				picard.ExpectDelete(mock, entityHelper, []string{"00000000-0000-0000-0000-000000000001"})
			},
		},
		{
			"Multiple Import with One Already Exists",
			[]string{"Simple", "Simple2"},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]ds.DataSourceNew)
				returnData := picard.GetReturnDataForLookup(dataSourceHelper, []ds.DataSourceNew{
					fixtures[0],
				})
				lookupKeys := picard.GetLookupKeys(dataSourceHelper, fixtures)
				parentID1 := returnData[0][0]
				sampleOrgID := "6ba7b810-9dbd-11d1-80b4-00c04fd430c8"
				picard.ExpectLookup(mock, dataSourceHelper, lookupKeys, returnData)
				picard.ExpectUpdate(mock, dataSourceHelper, []ds.DataSourceNew{
					fixtures[0],
				}, returnData)
				picard.ExpectInsert(mock, dataSourceHelper, []ds.DataSourceNew{
					fixtures[1],
				}, false)
				// Orphan Removal Query
				picard.ExpectQuery(mock, `
						^SELECT data_source_object.id, data_source_object.organization_id, data_source_object.data_source_id, data_source_object.name, data_source_object.schema, data_source_object.label, data_source_object.label_plural, data_source_object.created_by_id, data_source_object.updated_by_id, data_source_object.created_at, data_source_object.updated_at
						FROM data_source_object
						WHERE \(\(data_source_object.organization_id = \$1 AND data_source_object.data_source_id = \$2\)\)$
					`).
					WithArgs(sampleOrgID, parentID1).
					WillReturnRows(
						sqlmock.NewRows([]string{"name", "id", "data_source_id"}).
							AddRow("ChildRecord", "00000000-0000-0000-0000-000000000001", parentID1),
					)
				picard.ExpectDelete(mock, entityHelper, []string{"00000000-0000-0000-0000-000000000001"})
			},
		},

		{
			"Single Import with DSOs",
			[]string{"SimpleWithDSOs"},
			func(mock *sqlmock.Sqlmock, fixturesAbstract interface{}) {
				fixtures := fixturesAbstract.([]ds.DataSourceNew)
				returnData := picard.GetReturnDataForLookup(dataSourceHelper, nil)
				lookupKeys := picard.GetLookupKeys(dataSourceHelper, fixtures)
				picard.ExpectLookup(mock, dataSourceHelper, lookupKeys, returnData)
				insertRows := picard.ExpectInsert(mock, dataSourceHelper, fixtures, false)

				childObjects := []ds.EntityNew{}
				for index, fixture := range fixtures {
					for _, childObject := range fixture.Entities {
						childObject.DataSourceID = insertRows[index][0].(string)
						childObjects = append(childObjects, childObject)
					}
				}

				childReturnData := picard.GetReturnDataForLookup(entityHelper, nil)
				childLookupKeys := picard.GetLookupKeys(entityHelper, childObjects)
				picard.ExpectLookup(mock, entityHelper, childLookupKeys, childReturnData)
				picard.ExpectInsert(mock, entityHelper, childObjects, false)

			},
		},
	}

	for index, c := range cases {
		fmt.Printf("%v: %v", index+1, c.TestName)
		fmt.Println("")
		fixtures, err := loadTestDataSources(c.FixtureNames)
		if err != nil {
			t.Fatal(err)
		}

		if err = picard.RunImportTest(fixtures, c.ExpectationFunction); err != nil {
			t.Fatal(err)
		}
	}

}

// Actually imports the data into the local database
/*
func TestDataSourceFunc(t *testing.T) {
	t.Skip()

	connectionString := fmt.Sprintf(
		"postgres://localhost:%d/%s?sslmode=disable&user=%s&password=%s",
		15433,
		"warden",
		"warden",
		"wardenDBpass",
	)
	db, err := sql.Open("postgres", connectionString)

	if err != nil {
		t.Fatal(err)
	}

	orgID := sampleOrgID
	userID := sampleUserID

	dataSources, err := loadTestDataSources([]string{
		"Simple",
		"Simple2",
		"SimpleWithDSOs",
	})

	if err != nil {
		t.Fatal(err)
	}

	err = picard.New(orgID, userID, db).Deploy(dataSources, "data_source")

	if err != nil {
		t.Fatal(err)
	}
}
*/
