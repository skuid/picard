package picard

import (
	"database/sql/driver"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/skuid/picard/crypto"
	"github.com/skuid/picard/decoding"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/tags"
	"github.com/skuid/picard/testdata"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	uuid "github.com/satori/go.uuid"
)

//Test structs for JSONB tests
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
	Metadata     metadata.Metadata             `picard:"tablename=test_table"`
	TestFieldOne testdata.TestSerializedObject `picard:"jsonb,column=test_column_one"`
}

type modelOnePointerFieldJSONB struct {
	Metadata     metadata.Metadata              `picard:"tablename=test_table"`
	TestFieldOne *testdata.TestSerializedObject `picard:"jsonb,column=test_column_one"`
}

type modelOneArrayFieldJSONB struct {
	Metadata     metadata.Metadata               `picard:"tablename=test_table"`
	TestFieldOne []testdata.TestSerializedObject `picard:"jsonb,column=test_column_one"`
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

// LoadFixturesFromFiles creates a slice of structs from a slice of file names
func LoadFixturesFromFiles(names []string, path string, loadType reflect.Type, jsonTagKey string) (interface{}, error) {

	sliceOfStructs := reflect.New(reflect.SliceOf(loadType)).Elem()

	for _, name := range names {
		testObject := reflect.New(loadType).Interface()
		raw, err := ioutil.ReadFile(path + name + ".json")
		if err != nil {
			return nil, err
		}
		err = GetDecoder(&decoding.Config{
			TagKey: jsonTagKey,
		}).Unmarshal(raw, &testObject)
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
	TableMetadata    *tags.TableMetadata
	LookupFrom       string
	LookupSelect     string
	LookupWhere      string
	LookupReturnCols []string
	LookupFields     []string
	DBColumns        []string
	DataFields       []string
}

func (eh ExpectationHelper) getTableMetadata() *tags.TableMetadata {
	if eh.TableMetadata == nil {
		tableMetadata := tags.TableMetadataFromType(reflect.TypeOf(eh.FixtureType))
		eh.TableMetadata = tableMetadata
	}
	return eh.TableMetadata
}

func (eh ExpectationHelper) getPrimaryKeyColumnName() string {
	tableMetadata := eh.getTableMetadata()
	return tableMetadata.GetPrimaryKeyColumnName()
}

// GetInsertDBColumns returns the columns that should inserted into
func (eh ExpectationHelper) GetInsertDBColumns(includePrimaryKey bool) []string {
	tableMetadata := eh.getTableMetadata()
	if includePrimaryKey {
		return tableMetadata.GetColumnNames()
	}
	return tableMetadata.GetColumnNamesWithoutPrimaryKey()
}

func (eh ExpectationHelper) getUpdateDBColumns() []string {
	tableMetadata := eh.getTableMetadata()
	return tableMetadata.GetColumnNamesForUpdate()
}

//GetUpdateDBColumnsForFixture returnst the fields that should be updated for a particular fixture
func (eh ExpectationHelper) GetUpdateDBColumnsForFixture(fixtures interface{}, index int) []string {
	tableMetadata := eh.getTableMetadata()
	definedColumns := []string{}
	fixture := reflect.ValueOf(fixtures).Index(index)
	modelMetadata := metadata.GetMetadataFromPicardStruct(fixture)

	for _, dataField := range tableMetadata.GetFields() {
		definedOnStruct := isFieldDefinedOnStruct(modelMetadata, dataField.GetName(), fixture)
		isUpdateAudit := dataField.GetAudit() == "updated_by" || dataField.GetAudit() == "updated_at"
		if (definedOnStruct && !dataField.IsPrimaryKey()) || isUpdateAudit {
			definedColumns = append(definedColumns, dataField.GetColumnName())
		}
	}
	return definedColumns
}

// GetFixtureValue returns the value of a particular field on a fixture
func (eh ExpectationHelper) GetFixtureValue(fixtures interface{}, index int, fieldName string) driver.Value {
	tableMetadata := eh.getTableMetadata()
	fieldMetadata := tableMetadata.GetField(fieldName)
	fixture := reflect.ValueOf(fixtures).Index(index)
	field := fixture.FieldByName(fieldName)
	if fieldMetadata.IsJSONB() {
		unserializedValue := field.Interface()
		serializedValue, err := serializeJSONBColumn(unserializedValue)
		if err != nil {
			return unserializedValue
		}
		return serializedValue
	}
	return field.Interface()
}

// GetReturnDataKey Returns the first column at a given index of return data
func (eh ExpectationHelper) GetReturnDataKey(returnData [][]driver.Value, index int) string {
	return returnData[index][0].(string)
}

func (eh ExpectationHelper) getTableName() string {
	tableMetadata := eh.getTableMetadata()
	return tableMetadata.GetTableName()
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
		WHERE `

	var expectedArgs []driver.Value
	expectSQL = expectSQL + regexp.QuoteMeta(expect.LookupWhere) + ` = ANY\(\$1\) AND ` + expect.getTableName() + `.organization_id = \$2`
	expectedArgs = []driver.Value{
		pq.Array(lookupKeys),
		sampleOrgID,
	}

	(*mock).ExpectQuery(expectSQL).WithArgs(expectedArgs...).WillReturnRows(returnRows)
}

// ExpectQuery is just a wrapper around sqlmock
func ExpectQuery(mock *sqlmock.Sqlmock, expectSQL string) *sqlmock.ExpectedQuery {
	return (*mock).ExpectQuery(expectSQL)
}

// ExpectDelete Mocks a delete request to the database.
func ExpectDelete(mock *sqlmock.Sqlmock, expect ExpectationHelper, expectedIDs []string) [][]driver.Value {
	deletePKField := expect.getPrimaryKeyColumnName()
	valueParams := []string{}
	for index := range expectedIDs {
		valueParams = append(valueParams, `\$`+strconv.Itoa(index+1))
	}
	expectSQL := `
		DELETE FROM ` + expect.getTableName() + `
		WHERE ` + deletePKField + ` IN \(` + strings.Join(valueParams, ",") + `\) AND organization_id = \$` + strconv.Itoa(len(expectedIDs)+1)

	expectedArgs := []driver.Value{}
	for _, ID := range expectedIDs {
		expectedArgs = append(expectedArgs, ID)
	}
	expectedArgs = append(expectedArgs, sampleOrgID)
	(*mock).ExpectExec(expectSQL).WithArgs(expectedArgs...).WillReturnResult(sqlmock.NewResult(1, 1))
	return nil
}

// ExpectInsert Mocks an insert request to the database.
func ExpectInsert(mock *sqlmock.Sqlmock, expect ExpectationHelper, columnNames []string, insertValues [][]driver.Value) [][]driver.Value {

	columnNames = deDup(columnNames)

	returnData := [][]driver.Value{}
	for range columnNames {
		returnData = append(returnData, []driver.Value{
			uuid.NewV4().String(),
		})
	}

	valueStrings := []string{}
	index := 1
	expectedArgs := []driver.Value{}

	for _, insertValue := range insertValues {
		valueParams := []string{}
		nonNullInsertValues := []driver.Value{}

		for columnIndex := range columnNames {
			var columnValue interface{}
			if columnIndex >= 0 && columnIndex < len(insertValue) {
				columnValue = insertValue[columnIndex]
			}
			if columnValue == nil {
				valueParams = append(valueParams, `DEFAULT`)
			} else {
				valueParams = append(valueParams, `\$`+strconv.Itoa(index))
				nonNullInsertValues = append(nonNullInsertValues, columnValue)
				index++
			}
		}

		expectedArgs = append(expectedArgs, nonNullInsertValues...)
		valueStrings = append(valueStrings, strings.Join(valueParams, ","))

		returnData = append(returnData, []driver.Value{
			uuid.NewV4().String(),
		})
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
func ExpectUpdate(mock *sqlmock.Sqlmock, expect ExpectationHelper, updateColumnNames [][]string, updateValues [][]driver.Value, lookupResults [][]driver.Value) []driver.Result {

	results := []driver.Result{}

	for i, updateValue := range updateValues {

		setStrings := []string{}
		index := 1
		columnNames := updateColumnNames[i]

		for _, name := range columnNames {
			setStrings = append(setStrings, name+` = \$`+strconv.Itoa(index))
			index++
		}

		expectedArgs := updateValue
		expectedArgs = append(expectedArgs, sampleOrgID, lookupResults[i][0])

		result := sqlmock.NewResult(0, 1)

		expectSQL := `
			UPDATE ` + expect.getTableName() + ` SET ` + strings.Join(setStrings, ", ") + `
			WHERE organization_id = \$` + strconv.Itoa(index) + ` AND id = \$` + strconv.Itoa(index+1) + `
		`

		(*mock).ExpectExec(expectSQL).WithArgs(expectedArgs...).WillReturnResult(result)

		results = append(results, result)
	}

	return results
}

// RunImportTest Runs a Test Object Import Test
func RunImportTest(testObjects interface{}, testFunction func(*sqlmock.Sqlmock, interface{}), batchSize int) error {
	// Open new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		return err
	}

	crypto.SetEncryptionKey([]byte("the-key-has-to-be-32-bytes-long!"))
	SetConnection(db)

	orgID := sampleOrgID
	userID := sampleUserID

	mock.ExpectBegin()
	testFunction(&mock, testObjects)
	mock.ExpectCommit()

	p := New(orgID, userID).(*PersistenceORM)
	p.batchSize = batchSize
	// Deploy the list of data sources
	return p.Deploy(testObjects)

}
