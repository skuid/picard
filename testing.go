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
	FixtureType      interface{}
	TableMetadata    *tableMetadata
	LookupFrom       string
	LookupSelect     string
	LookupWhere      string
	LookupReturnCols []string
	LookupFields     []string
	DBColumns        []string
	DataFields       []string
}

func (eh ExpectationHelper) getTableMetadata() *tableMetadata {
	if eh.TableMetadata == nil {
		tableMetadata := tableMetadataFromType(reflect.TypeOf(eh.FixtureType))
		eh.TableMetadata = tableMetadata
	}
	return eh.TableMetadata
}

func (eh ExpectationHelper) getInsertDBColumns(includePrimaryKey bool) []string {
	tableMetadata := eh.getTableMetadata()
	if includePrimaryKey {
		return tableMetadata.getColumnNames()
	}
	return tableMetadata.getColumnNamesWithoutPrimaryKey()
}

func (eh ExpectationHelper) getUpdateDBColumns() []string {
	tableMetadata := eh.getTableMetadata()
	return tableMetadata.getColumnNamesForUpdate()
}

func (eh ExpectationHelper) getColumnValues(object reflect.Value, isUpdate bool, includePrimaryKey bool) []driver.Value {
	tableMetadata := eh.getTableMetadata()
	values := []driver.Value{}

	for _, dataField := range tableMetadata.getFields() {
		if !includePrimaryKey && !isUpdate && dataField.isPrimaryKey {
			continue
		}

		if isUpdate && !dataField.includeInUpdate() {
			continue
		}

		field := object.FieldByName(dataField.name)
		value := field.Interface()
		if dataField.isMultitenancyKey {
			value = sampleOrgID
		} else if dataField.isEncrypted {
			value = sqlmock.AnyArg()
		} else if dataField.isJSONB {
			serializedValue, err := serializeJSONBColumn(value)
			if err == nil {
				value = serializedValue
			}
		} else if dataField.audit != "" {
			if dataField.audit == "created_by" {
				value = sampleUserID
			} else if dataField.audit == "updated_by" {
				value = sampleUserID
			} else if dataField.audit == "created_at" {
				value = sqlmock.AnyArg()
			} else if dataField.audit == "updated_at" {
				value = sqlmock.AnyArg()
			}
		}

		values = append(values, value)
	}

	return values
}

func (eh ExpectationHelper) getTableName() string {
	tableMetadata := eh.getTableMetadata()
	return tableMetadata.tableName
}

var sampleOrgID = "6ba7b810-9dbd-11d1-80b4-00c04fd430c8"
var sampleUserID = "72c431ec-14ed-4d77-9948-cb92e816a3a7"

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
		fromStatement = expect.getTableName()
	}

	expectSQL := `
		SELECT ` + regexp.QuoteMeta(expect.LookupSelect) + ` 
		FROM ` + regexp.QuoteMeta(fromStatement) + ` 
		WHERE ` + regexp.QuoteMeta(expect.LookupWhere) + ` = ANY\(\$1\) AND ` + expect.getTableName() + `.organization_id = \$2
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
func ExpectInsert(mock *sqlmock.Sqlmock, expect ExpectationHelper, objects interface{}, includePrimaryKey bool) [][]driver.Value {

	returnData := getReturnDataForInsert(expect, objects)

	valueStrings := []string{}
	index := 1
	expectedArgs := []driver.Value{}

	columnNames := expect.getInsertDBColumns(includePrimaryKey)

	if objects != nil {
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			object := s.Index(i)

			valueParams := []string{}

			for range columnNames {
				valueParams = append(valueParams, `\$`+strconv.Itoa(index))
				index++
			}

			expectedArgs = append(expectedArgs, expect.getColumnValues(object, false, includePrimaryKey)...)
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
		INSERT INTO ` + expect.getTableName() + `
		\(` + strings.Join(columnNames, ",") + `\)
		VALUES \(` + strings.Join(valueStrings, `\),\(`) + `\) RETURNING "id"
	`

	(*mock).ExpectQuery(expectSQL).WithArgs(expectedArgs...).WillReturnRows(returnRows)

	return returnData
}

// ExpectUpdate Mocks an update request to the database.
func ExpectUpdate(mock *sqlmock.Sqlmock, expect ExpectationHelper, objects interface{}, lookupResults [][]driver.Value) []driver.Result {

	results := []driver.Result{}
	columnNames := expect.getUpdateDBColumns()

	if objects != nil {
		s := reflect.ValueOf(objects)
		for i := 0; i < s.Len(); i++ {
			object := s.Index(i)

			setStrings := []string{}
			index := 1

			for _, name := range columnNames {
				setStrings = append(setStrings, name+` = \$`+strconv.Itoa(index))
				index++
			}

			expectedArgs := expect.getColumnValues(object, true, false)
			expectedArgs = append(expectedArgs, sampleOrgID, lookupResults[i][0])

			result := sqlmock.NewResult(0, 1)

			expectSQL := `
				UPDATE ` + expect.getTableName() + ` SET ` + strings.Join(setStrings, ", ") + `
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

	orgID := sampleOrgID
	userID, _ := uuid.FromString(sampleUserID)

	mock.ExpectBegin()

	testFunction(&mock, testObjects)

	mock.ExpectCommit()

	// Deploy the list of data sources
	return New(orgID, userID).Deploy(testObjects)

}
