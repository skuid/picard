package picard

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
)

const separator = "|"

// Lookup structure
type Lookup struct {
	TableName           string
	MatchDBColumn       string
	MatchObjectProperty string
	JoinKey             string
	Query               bool
}

// Child structure
type Child struct {
	FieldName  string
	FieldType  reflect.Type
	ForeignKey string
}

// DBChange structure
type DBChange struct {
	changes       map[string]interface{}
	originalValue reflect.Value
}

// ORM interface describes the behavior API of any picard ORM
type ORM interface {
	FilterModel(interface{}) ([]interface{}, error)
	SaveModel(model interface{}) error
	CreateModel(model interface{}) error
	DeleteModel(model interface{}) (int64, error)
	Deploy(data interface{}) error
}

// PersistenceORM provides the necessary configuration to perform an upsert of objects without IDs
// into a relational database using lookup fields to match and field name transformations.
type PersistenceORM struct {
	multitenancyValue uuid.UUID
	performedBy       uuid.UUID
	transaction       *sql.Tx
}

// New Creates a new Picard Object and handle defaults
func New(multitenancyValue uuid.UUID, performerID uuid.UUID) ORM {
	return PersistenceORM{
		multitenancyValue: multitenancyValue,
		performedBy:       performerID,
	}
}

// Decode decodes a reader using a specified decoder, but also writes metadata to picard StructMetadata
func Decode(body io.Reader, destination interface{}) error {
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	err = Unmarshal(bytes, destination)
	if err != nil {
		return err
	}
	return nil
}

func getStructValue(v interface{}) (reflect.Value, error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Struct {
		return value, errors.New("Models must be structs")
	}
	return value, nil
}

// Deploy is the public method to start a Picard deployment. Send in a table name and a slice of structs
// and it will attempt a deployment.
func (p PersistenceORM) Deploy(data interface{}) error {
	tx, err := GetConnection().Begin()
	if err != nil {
		return err
	}

	p.transaction = tx

	if err = p.upsert(data); err != nil {
		return err
	}

	return tx.Commit()
}

// Upsert takes data in the form of a slice of structs and performs a series of database
// operations that will sync the database with the state of that deployment payload
func (p PersistenceORM) upsert(data interface{}) error {

	// Verify that we've been passed valid input
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Slice {
		return errors.New("Can only upsert slices")
	}

	picardTags := picardTagsFromType(t.Elem())
	lookups := picardTags.Lookups()
	childOptions := picardTags.Children()
	columnNames := picardTags.DataColumnNames()
	tableName := picardTags.TableName()
	primaryKeyColumnName := picardTags.PrimaryKeyColumnName()
	multitenancyKeyColumnName := picardTags.MultitenancyKeyColumnName()

	if tableName == "" {
		return errors.New("No table name specified in struct metadata")
	}

	results, err := p.checkForExisting(data, tableName, lookups, multitenancyKeyColumnName, primaryKeyColumnName)
	if err != nil {
		return err
	}

	inserts, updates, _ /*deletes*/, err := p.generateChanges(data, results, lookups)
	if err != nil {
		return err
	}

	// Execute Delete Queries

	// Execute Update Queries
	if err := p.performUpdates(updates, tableName, columnNames, multitenancyKeyColumnName, primaryKeyColumnName); err != nil {
		return err
	}

	// Execute Insert Queries
	if err := p.performInserts(inserts, tableName, columnNames, primaryKeyColumnName); err != nil {
		return err
	}

	combinedOperations := append(updates, inserts...)

	// Perform Child Upserts
	err = p.performChildUpserts(combinedOperations, primaryKeyColumnName, childOptions)

	if err != nil {
		return err
	}

	return nil
}

func (p PersistenceORM) performUpdates(updates []DBChange, tableName string, columnNames []string, multitenancyKeyColumnName string, primaryKeyColumnName string) error {
	if len(updates) > 0 {

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		for _, update := range updates {
			changes := update.changes
			updateQuery := psql.Update(tableName)

			values := getColumnValues(columnNames, changes)

			for index, columnName := range columnNames {
				updateQuery = updateQuery.Set(columnName, values[index])
			}

			updateQuery = updateQuery.Where(squirrel.Eq{multitenancyKeyColumnName: p.multitenancyValue})
			updateQuery = updateQuery.Where(squirrel.Eq{primaryKeyColumnName: changes[primaryKeyColumnName]})

			_, err := updateQuery.RunWith(p.transaction).Exec()

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p PersistenceORM) performInserts(inserts []DBChange, tableName string, columnNames []string, primaryKeyColumnName string) error {
	if len(inserts) > 0 {

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		insertQuery := psql.Insert(tableName)
		insertQuery = insertQuery.Columns(columnNames...)

		for _, insert := range inserts {
			changes := insert.changes
			insertQuery = insertQuery.Values(getColumnValues(columnNames, changes)...)
		}

		insertQuery = insertQuery.Suffix(fmt.Sprintf("RETURNING \"%s\"", primaryKeyColumnName))

		rows, err := insertQuery.RunWith(p.transaction).Query()
		if err != nil {
			return err
		}

		insertResults, err := getQueryResults(rows)
		if err != nil {
			return err
		}

		// Insert our new keys into the change objects
		for index, insert := range inserts {
			insert.changes[primaryKeyColumnName] = insertResults[index][primaryKeyColumnName]
		}
	}
	return nil
}

func (p PersistenceORM) getExistingObjectByID(tableName string, multitenancyColumn string, IDColumn string, IDValue uuid.UUID) (map[string]interface{}, error) {
	rows, err := squirrel.Select(fmt.Sprintf("%v.%v", tableName, IDColumn)).PlaceholderFormat(squirrel.Dollar).
		From(tableName).
		Where(squirrel.Eq{fmt.Sprintf("%v.%v", tableName, IDColumn): IDValue}).
		Where(squirrel.Eq{fmt.Sprintf("%v.%v", tableName, multitenancyColumn): p.multitenancyValue}).
		RunWith(p.transaction).
		Query()

	if err != nil {
		return nil, err
	}
	results, err := getQueryResults(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func (p PersistenceORM) checkForExisting(
	data interface{},
	tableName string,
	lookups []Lookup,
	multitenancyKeyColumnName string,
	primaryKeyColumnName string,
) (
	map[string]interface{},
	error,
) {

	query := p.getLookupQuery(data, tableName, lookups, multitenancyKeyColumnName, primaryKeyColumnName)

	rows, err := query.RunWith(p.transaction).Query()

	if err != nil {
		return nil, err
	}

	return getLookupQueryResults(rows, tableName, lookups)
}

func (p PersistenceORM) getLookupQuery(data interface{}, tableName string, lookups []Lookup, multitenancyKeyColumnName string, primaryKeyColumnName string) *squirrel.SelectBuilder {
	query := squirrel.Select(fmt.Sprintf("%v.%v", tableName, primaryKeyColumnName))
	query = query.From(tableName)
	wheres := []string{}
	whereValues := []string{}

	for _, lookup := range lookups {
		if lookup.JoinKey != "" && lookup.TableName != "" {
			query = query.Join(fmt.Sprintf("%[1]v on %[1]v.%[2]v = %[3]v", lookup.TableName, primaryKeyColumnName, lookup.JoinKey))
		}
		tableToUse := tableName
		if lookup.TableName != "" {
			tableToUse = lookup.TableName
		}
		query = query.Column(fmt.Sprintf("%[1]v.%[2]v as %[1]v_%[2]v", tableToUse, lookup.MatchDBColumn))
		if lookup.Query {
			wheres = append(wheres, fmt.Sprintf("COALESCE(%v.%v::\"varchar\",'')", tableToUse, lookup.MatchDBColumn))
		}
	}

	s := reflect.ValueOf(data)
	for i := 0; i < s.Len(); i++ {
		whereValues = append(whereValues, getObjectKeyReflect(s.Index(i), lookups))
	}

	query = query.Where(strings.Join(wheres, " || '"+separator+"' || ")+" = ANY($1)", pq.Array(whereValues))
	query = query.Where(fmt.Sprintf("%v.%v = $2", tableName, multitenancyKeyColumnName), p.multitenancyValue)

	return &query
}

func (p PersistenceORM) performChildUpserts(changeObjects []DBChange, primaryKeyColumnName string, children []Child) error {

	for _, child := range children {
		// If it doesn't exist already, create an entry in the upserts map

		data := reflect.New(child.FieldType).Elem()

		for _, changeObject := range changeObjects {
			// Add the id of the parent to any foreign keys on the child
			originalValue := changeObject.originalValue
			childValue := originalValue.FieldByName(child.FieldName)
			for i := 0; i < childValue.Len(); i++ {
				value := childValue.Index(i)
				keyField := value.FieldByName(child.ForeignKey)
				foreignKeyValue := changeObject.changes[primaryKeyColumnName]
				keyField.SetString(foreignKeyValue.(string))
				data = reflect.Append(data, value)
			}
		}

		if data.Len() > 0 {
			err := p.upsert(data.Interface())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// generateChanges takes results from performing lookup and foreign lookup
// queries and creates a set of inserts, updates, and deletes to be
// performed on the database.
func (p PersistenceORM) generateChanges(
	data interface{},
	results map[string]interface{},
	lookups []Lookup,
) (
	[]DBChange,
	[]DBChange,
	[]DBChange,
	error,
) {
	inserts := []DBChange{}
	updates := []DBChange{}
	deletes := []DBChange{}

	s := reflect.ValueOf(data)
	for i := 0; i < s.Len(); i++ {
		value := s.Index(i)
		objectKey := getObjectKeyReflect(value, lookups)
		object := results[objectKey]

		var existingObj map[string]interface{}

		if object != nil {
			existingObj = object.(map[string]interface{})
		}

		// TODO: Implement Delete Conditions
		shouldDelete := false

		if shouldDelete {
			if existingObj != nil {
				deletes = append(deletes, DBChange{
					changes: existingObj,
				})
			}
			continue
		}

		// TODO: Implement Missing/Required Fields
		missingRequiredFields := false

		if missingRequiredFields {
			continue
		}

		dbChange, err := p.processObject(value, existingObj)

		if err != nil {
			return nil, nil, nil, err
		}

		if dbChange.changes == nil {
			continue
		}

		if existingObj != nil {
			updates = append(updates, dbChange)
		} else {
			inserts = append(inserts, dbChange)
		}

	}

	return inserts, updates, deletes, nil
}

func (p PersistenceORM) processObject(
	metadataObject reflect.Value,
	databaseObject map[string]interface{},
) (DBChange, error) {
	returnObject := map[string]interface{}{}

	// Apply Field Mappings
	t := reflect.TypeOf(metadataObject.Interface())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := metadataObject.FieldByName(field.Name).String()
		picardTags := getStructTagsMap(field, "picard")

		columnName, hasColumnName := picardTags["column"]

		if value != "" && hasColumnName {
			returnObject[columnName] = value
		}
	}
	picardTags := picardTagsFromType(metadataObject.Type())
	primaryKeyColumnName := picardTags.PrimaryKeyColumnName()
	multitenancyKeyColumnName := picardTags.MultitenancyKeyColumnName()

	if databaseObject != nil {
		returnObject[primaryKeyColumnName] = databaseObject[primaryKeyColumnName]
	}

	// Handle audit fields
	// TODO: make audit fields somehow less hard-coded / configurable

	returnObject[multitenancyKeyColumnName] = p.multitenancyValue
	returnObject["created_by_id"] = p.performedBy
	returnObject["updated_by_id"] = p.performedBy
	returnObject["created_at"] = time.Now()
	returnObject["updated_at"] = time.Now()

	// Process encrypted columns

	encryptedColumns := picardTags.EncryptedColumns()
	for _, column := range encryptedColumns {
		value := returnObject[column]

		// If value is nil, no point in encrypting it.
		if value == nil {
			continue
		}

		var valueAsBytes []byte

		// Handle both non-interface and interface types as we convert to byte array
		switch value.(type) {
		case string:
			valueAsBytes = []byte(value.(string))
		default:
			assertedBytes, ok := value.([]byte)
			if !ok {
				return DBChange{}, errors.New("can only encrypt values that can be converted to bytes")
			}
			valueAsBytes = assertedBytes
		}

		// Do encryption over bytes
		encryptedValue, err := EncryptBytes(valueAsBytes)
		if err != nil {
			return DBChange{}, err
		}

		// Base64 encode to get standard character set
		encoded := base64.StdEncoding.EncodeToString(encryptedValue)

		// Replace the original value with the newly encrypted value.
		returnObject[column] = encoded
	}

	// TODO: Implement Foreign Key Merges

	return DBChange{
		changes:       returnObject,
		originalValue: metadataObject,
	}, nil
}

func getObjectKey(objects map[string]interface{}, tableName string, lookups []Lookup) string {
	keyValue := []string{}
	for _, lookup := range lookups {
		tableToUse := tableName
		if lookup.TableName != "" {
			tableToUse = lookup.TableName
		}
		keyPart := objects[fmt.Sprintf("%v_%v", tableToUse, lookup.MatchDBColumn)]
		var keyString string
		if keyPart == nil {
			keyString = ""
		} else {
			keyString = keyPart.(string)
		}
		keyValue = append(keyValue, keyString)

	}
	return strings.Join(keyValue, separator)
}

func getObjectKeyReflect(value reflect.Value, lookups []Lookup) string {
	keyValue := []string{}
	for _, lookup := range lookups {
		keyValue = append(keyValue, value.FieldByName(lookup.MatchObjectProperty).String())
	}
	return strings.Join(keyValue, separator)
}

func getQueryResults(rows *sql.Rows) ([]map[string]interface{}, error) {
	defer rows.Close()

	cols, err := rows.Columns()

	if err != nil {
		return nil, err
	}

	results := []map[string]interface{}{}

	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columns[i]
			reflectValue := reflect.ValueOf(val)
			if reflectValue.IsValid() && reflectValue.Type() == reflect.TypeOf([]byte(nil)) && reflectValue.Len() == 36 {
				m[colName] = string(val.([]uint8))
			} else {
				m[colName] = val
			}
		}

		results = append(results, m)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func getLookupQueryResults(rows *sql.Rows, tableName string, lookups []Lookup) (map[string]interface{}, error) {

	results, err := getQueryResults(rows)
	if err != nil {
		return nil, err
	}

	resultsMap := map[string]interface{}{}

	for _, v := range results {
		resultsMap[getObjectKey(v, tableName, lookups)] = v
	}

	return resultsMap, nil
}

func getColumnValues(columnNames []string, data map[string]interface{}) []interface{} {
	columnValues := []interface{}{}
	for _, columnName := range columnNames {
		columnValues = append(columnValues, data[columnName])
	}
	return columnValues
}

func (p PersistenceORM) generateWhereClausesFromModel(filterModelValue reflect.Value, zeroFields []string) ([]squirrel.Eq, error) {
	var returnClauses []squirrel.Eq

	t := filterModelValue.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := filterModelValue.FieldByName(field.Name)

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		_, isMultitenancyColumn := picardTags["multitenancy_key"]
		isZeroField := reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(field.Type).Interface())

		isZeroColumn := false
		for _, zeroField := range zeroFields {
			if field.Name == zeroField {
				isZeroColumn = true
			}
		}

		switch {
		case hasColumn && isMultitenancyColumn:
			returnClauses = append(returnClauses, squirrel.Eq{column: p.multitenancyValue})
		case hasColumn && !isZeroField:
			_, isEncrypted := picardTags["encrypted"]
			if isEncrypted {
				return nil, errors.New("cannot perform queries with where clauses on encrypted fields")
			}

			returnClauses = append(returnClauses, squirrel.Eq{column: fieldValue.Interface()})
		case isZeroColumn:
			returnClauses = append(returnClauses, squirrel.Eq{column: reflect.Zero(field.Type).Interface()})
		}
	}
	return returnClauses, nil
}
