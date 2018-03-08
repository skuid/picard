package picard

import (
	"database/sql/driver"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	uuid "github.com/satori/go.uuid"
)

// LoadFixturesFromFiles creates a slice of structs from a slice of file names
func LoadFixturesFromFiles(names []string, path string, loadType reflect.Type) (interface{}, error) {

	sliceOfStructs := reflect.New(reflect.SliceOf(loadType)).Elem()

	for _, name := range names {
		testObject := reflect.New(loadType).Interface()
		raw, err := ioutil.ReadFile(path + name + ".json")
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(raw, &testObject)
		if err != nil {
			return nil, err
		}
		sliceOfStructs = reflect.Append(sliceOfStructs, reflect.ValueOf(testObject).Elem())
	}

	return sliceOfStructs.Interface(), nil
}

// ExpectationHelper struct that contains expectations about a particular object
type ExpectationHelper struct {
	TableName        string
	LookupFrom       string
	LookupSelect     string
	LookupWhere      string
	LookupReturnCols []string
	LookupFields     []string
	DBColumns        []string
	DataFields       []string
}

var sampleOrgID = "6ba7b810-9dbd-11d1-80b4-00c04fd430c8"
var sampleUserID = "72c431ec-14ed-4d77-9948-cb92e816a3a7"

func getTestColumnValues(expect ExpectationHelper, object reflect.Value) []driver.Value {
	values := []driver.Value{}

	for _, dataField := range expect.DataFields {

		// Add in Checks for Special Values
		if dataField == "OrganizationID" {
			values = append(values, sampleOrgID)
		} else if dataField == "CreatedByID" {
			values = append(values, sampleUserID)
		} else if dataField == "UpdatedByID" {
			values = append(values, sampleUserID)
		} else if dataField == "CreatedDate" {
			values = append(values, sqlmock.AnyArg())
		} else if dataField == "UpdatedDate" {
			values = append(values, sqlmock.AnyArg())
		} else {
			field := object.FieldByName(dataField)
			structField, _ := object.Type().FieldByName(dataField)
			tagsMap := getStructTagsMap(structField, "picard")
			_, isEncrypted := tagsMap["encrypted"]
			value := field.Interface()
			if isEncrypted {
				values = append(values, sqlmock.AnyArg())
			} else {
				values = append(values, value)
			}
		}
	}

	return values
}

// GetReturnDataForLookup creates sample return data from sample structs
func GetReturnDataForLookup(expect ExpectationHelper, foundObjects interface{}) [][]driver.Value {

	returnData := [][]driver.Value{}

	if foundObjects != nil {
		s := reflect.ValueOf(foundObjects)
		for i := 0; i < s.Len(); i++ {
			object := s.Index(i)
			returnItem := []driver.Value{
				uuid.NewV4().String(),
			}

			for _, lookup := range expect.LookupFields {
				field := object.FieldByName(lookup)
				value := field.String()
				returnItem = append(returnItem, value)
			}

			returnData = append(returnData, returnItem)
		}
	}

	return returnData
}

// GetLookupKeys returns sample object keys from sample objects
func GetLookupKeys(expect ExpectationHelper, objects interface{}) []string {

	returnKeys := []string{}

	if objects != nil {
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			object := s.Index(i)

			keyParts := []string{}

			for _, lookup := range expect.LookupFields {
				field := object.FieldByName(lookup)
				value := field.String()
				keyParts = append(keyParts, value)
			}

			returnKeys = append(returnKeys, strings.Join(keyParts, "|"))
		}
	}

	return returnKeys
}

// ExpectLookup Mocks a lookup request to the database. Makes a request for the lookup keys
// and returns the rows privided in the returnKeys argument
func ExpectLookup(mock *sqlmock.Sqlmock, expect ExpectationHelper, lookupKeys []string, returnData [][]driver.Value) {

	returnRows := sqlmock.NewRows(expect.LookupReturnCols)

	for _, row := range returnData {
		returnRows.AddRow(row...)
	}

	fromStatement := expect.LookupFrom
	if fromStatement == "" {
		fromStatement = expect.TableName
	}

	expectSQL := `
		SELECT ` + regexp.QuoteMeta(expect.LookupSelect) + ` 
		FROM ` + regexp.QuoteMeta(fromStatement) + ` 
		WHERE ` + regexp.QuoteMeta(expect.LookupWhere) + ` = ANY\(\$1\) AND ` + expect.TableName + `.organization_id = \$2
	`

	expectedArgs := []driver.Value{
		pq.Array(lookupKeys),
		sampleOrgID,
	}

	(*mock).ExpectQuery(expectSQL).WithArgs(expectedArgs...).WillReturnRows(returnRows)
}

func getReturnDataForInsert(expect ExpectationHelper, objects interface{}) [][]driver.Value {
	returnData := [][]driver.Value{}

	if objects != nil {
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			returnData = append(returnData, []driver.Value{
				uuid.NewV4().String(),
			})
		}
	}

	return returnData
}

// ExpectInsert Mocks an insert request to the database.
func ExpectInsert(mock *sqlmock.Sqlmock, expect ExpectationHelper, objects interface{}) [][]driver.Value {

	returnData := getReturnDataForInsert(expect, objects)

	valueStrings := []string{}
	index := 1
	expectedArgs := []driver.Value{}

	if objects != nil {
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			object := s.Index(i)

			valueParams := []string{}

			for range expect.DBColumns {
				valueParams = append(valueParams, `\$`+strconv.Itoa(index))
				index++
			}

			expectedArgs = append(expectedArgs, getTestColumnValues(expect, object)...)
			valueStrings = append(valueStrings, strings.Join(valueParams, ","))

			returnData = append(returnData, []driver.Value{
				uuid.NewV4().String(),
			})
		}
	}

	returnRows := sqlmock.NewRows([]string{"id"})

	for _, row := range returnData {
		returnRows.AddRow(row...)
	}

	expectSQL := `
		INSERT INTO ` + expect.TableName + `
		\(` + strings.Join(expect.DBColumns, ",") + `\)
		VALUES \(` + strings.Join(valueStrings, `\),\(`) + `\) RETURNING "id"
	`

	(*mock).ExpectQuery(expectSQL).WithArgs(expectedArgs...).WillReturnRows(returnRows)

	return returnData
}

// ExpectUpdate Mocks an update request to the database.
func ExpectUpdate(mock *sqlmock.Sqlmock, expect ExpectationHelper, objects interface{}, lookupResults [][]driver.Value) []driver.Result {

	results := []driver.Result{}

	if objects != nil {
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			object := s.Index(i)

			setStrings := []string{}
			index := 1

			for _, name := range expect.DBColumns {
				setStrings = append(setStrings, name+` = \$`+strconv.Itoa(index))
				index++
			}

			expectedArgs := getTestColumnValues(expect, object)
			expectedArgs = append(expectedArgs, sampleOrgID, lookupResults[i][0])

			result := sqlmock.NewResult(0, 1)

			expectSQL := `
				UPDATE ` + expect.TableName + ` SET ` + strings.Join(setStrings, ", ") + `
				WHERE organization_id = \$` + strconv.Itoa(index) + ` AND id = \$` + strconv.Itoa(index+1) + `
			`

			(*mock).ExpectExec(expectSQL).WithArgs(expectedArgs...).WillReturnResult(result)

			results = append(results, result)
		}
	}

	return results
}

// RunImportTest Runs a Test Object Import Test
func RunImportTest(testObjects interface{}, testFunction func(*sqlmock.Sqlmock, interface{})) error {
	// Open new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		return err
	}

	SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))
	SetConnection(db)

	orgID, _ := uuid.FromString(sampleOrgID)
	userID, _ := uuid.FromString(sampleUserID)

	mock.ExpectBegin()

	testFunction(&mock, testObjects)

	mock.ExpectCommit()

	// Deploy the list of data sources
	return New(orgID, userID).Deploy(testObjects)

}
