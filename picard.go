package picard

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	validator "gopkg.in/go-playground/validator.v9"
)

const separator = "|"

// Lookup structure
type Lookup struct {
	TableName           string
	MatchDBColumn       string
	MatchObjectProperty string
	JoinKey             string
	Query               bool
	Value               interface{}
}

// Child structure
type Child struct {
	FieldName     string
	FieldType     reflect.Type
	FieldKind     reflect.Kind
	ForeignKey    string
	KeyMappings   []string
	ValueMappings map[string]string
}

// ForeignKey structure
type ForeignKey struct {
	TableMetadata    *tableMetadata
	FieldName        string
	KeyColumn        string
	RelatedFieldName string
	Required         bool
	NeedsLookup      bool
	LookupResults    map[string]interface{}
	LookupsUsed      []Lookup
}

// DBChange structure
type DBChange struct {
	changes       map[string]interface{}
	originalValue reflect.Value
}

// DBChangeSet structure
type DBChangeSet struct {
	inserts               []DBChange
	updates               []DBChange
	deletes               []DBChange
	insertsHavePrimaryKey bool
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

	tableMetadata, err := getTableMetadata(data)
	if err != nil {
		return err
	}

	changeSet, err := p.generateChanges(data, tableMetadata)
	if err != nil {
		return err
	}

	// Execute Delete Queries

	// Execute Update Queries
	if err := p.performUpdates(changeSet.updates, tableMetadata); err != nil {
		return err
	}

	// Execute Insert Queries
	if err := p.performInserts(changeSet.inserts, changeSet.insertsHavePrimaryKey, tableMetadata); err != nil {
		return err
	}

	combinedOperations := append(changeSet.updates, changeSet.inserts...)

	// Perform Child Upserts
	err = p.performChildUpserts(combinedOperations, tableMetadata)

	if err != nil {
		return err
	}

	return nil
}

func (p PersistenceORM) performUpdates(updates []DBChange, tableMetadata *tableMetadata) error {
	if len(updates) > 0 {

		tableName := tableMetadata.tableName

		columnNames := tableMetadata.getColumnNamesWithoutPrimaryKey()

		primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
		multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		for _, update := range updates {
			changes := update.changes
			updateQuery := psql.Update(tableName)

			for _, columnName := range columnNames {
				value, ok := changes[columnName]
				if ok {
					updateQuery = updateQuery.Set(columnName, value)
				}
			}

			if multitenancyKeyColumnName != "" {
				updateQuery = updateQuery.Where(squirrel.Eq{multitenancyKeyColumnName: p.multitenancyValue})
			}
			updateQuery = updateQuery.Where(squirrel.Eq{primaryKeyColumnName: changes[primaryKeyColumnName]})

			_, err := updateQuery.RunWith(p.transaction).Exec()

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p PersistenceORM) performInserts(inserts []DBChange, insertsHavePrimaryKey bool, tableMetadata *tableMetadata) error {
	if len(inserts) > 0 {

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		tableName := tableMetadata.tableName

		primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()

		var columnNames []string

		if insertsHavePrimaryKey {
			columnNames = tableMetadata.getColumnNames()
		} else {
			columnNames = tableMetadata.getColumnNamesWithoutPrimaryKey()
		}

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

func (p PersistenceORM) getExistingObjectByID(tableMetadata *tableMetadata, IDValue interface{}) (map[string]interface{}, error) {
	tableName := tableMetadata.tableName
	IDColumn := tableMetadata.getPrimaryKeyColumnName()
	multitenancyColumn := tableMetadata.getMultitenancyKeyColumnName()
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
	tableMetadata *tableMetadata,
	foreignFieldName string,
) (
	map[string]interface{},
	[]Lookup,
	error,
) {
	tableName := tableMetadata.tableName
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()
	tableAliasCache := map[string]string{}
	lookupsToUse := getLookupsForDeploy(data, tableMetadata, foreignFieldName, tableAliasCache)
	lookupObjectKeys := getLookupObjectKeys(data, lookupsToUse, foreignFieldName)

	if len(lookupObjectKeys) == 0 || len(lookupsToUse) == 0 {
		return map[string]interface{}{}, lookupsToUse, nil
	}

	query := squirrel.Select(fmt.Sprintf("%v.%v", tableName, primaryKeyColumnName))
	query = query.From(tableName)

	columns, joins, whereFields := getQueryParts(tableName, primaryKeyColumnName, lookupsToUse, tableAliasCache)

	wheres := []string{}

	for _, join := range joins {
		query = query.Join(join)
	}

	for _, whereField := range whereFields {
		for whereFieldKey := range whereField {
			wheres = append(wheres, fmt.Sprintf("COALESCE(%v::\"varchar\",'')", whereFieldKey))
		}
	}

	query = query.Columns(columns...)

	query = query.Where(strings.Join(wheres, " || '"+separator+"' || ")+" = ANY($1)", pq.Array(lookupObjectKeys))

	if multitenancyKeyColumnName != "" {
		query = query.Where(fmt.Sprintf("%v.%v = $2", tableName, multitenancyKeyColumnName), p.multitenancyValue)
	}

	rows, err := query.RunWith(p.transaction).Query()
	if err != nil {
		return nil, nil, err
	}

	results, err := getLookupQueryResults(rows, tableName, lookupsToUse, tableAliasCache)
	if err != nil {
		return nil, nil, err
	}

	return results, lookupsToUse, nil
}

func getLookupsFromForeignKeys(foreignKeys []ForeignKey, baseJoinKey string, baseObjectProperty string, tableAliasCache map[string]string) []Lookup {
	lookupsToUse := []Lookup{}

	for _, foreignKey := range foreignKeys {
		if foreignKey.NeedsLookup {
			tableMetadata := foreignKey.TableMetadata
			joinKey := getJoinKey(baseJoinKey, foreignKey.KeyColumn)
			for _, lookup := range tableMetadata.lookups {
				lookupsToUse = append(lookupsToUse, Lookup{
					TableName:           tableMetadata.tableName,
					MatchDBColumn:       lookup.MatchDBColumn,
					MatchObjectProperty: getMatchObjectProperty(baseObjectProperty, foreignKey.RelatedFieldName, lookup.MatchObjectProperty),
					JoinKey:             joinKey,
					Query:               true,
				})
			}
			newBaseJoinKey := getTableAlias(tableMetadata.tableName, joinKey, tableAliasCache)
			lookupsToUse = append(lookupsToUse, getLookupsFromForeignKeys(tableMetadata.foreignKeys, newBaseJoinKey, getNewBaseObjectProperty(baseObjectProperty, foreignKey.RelatedFieldName), tableAliasCache)...)
		}
	}
	return lookupsToUse
}

func getTableAlias(table string, joinKey string, tableAliasCache map[string]string) string {
	if table == "" || joinKey == "" {
		return table
	}
	// First try to find your table alias in the map
	alias, foundAlias := tableAliasCache[table+"_"+joinKey]

	if foundAlias {
		return alias
	}
	tableAliasCounter := len(tableAliasCache) + 1
	newAlias := "t" + strconv.Itoa(tableAliasCounter)
	tableAliasCache[table+"_"+joinKey] = newAlias
	return newAlias
}

func getJoinKey(baseJoinKey string, keyColumn string) string {
	if baseJoinKey != "" {
		return baseJoinKey + "." + keyColumn
	}
	return keyColumn
}
func getNewBaseObjectProperty(baseObjectProperty string, relatedFieldName string) string {
	if baseObjectProperty != "" {
		return baseObjectProperty + "." + relatedFieldName
	}
	return relatedFieldName
}

func getMatchObjectProperty(baseObjectProperty string, relatedFieldName string, matchObjectProperty string) string {
	return getNewBaseObjectProperty(baseObjectProperty, relatedFieldName) + "." + matchObjectProperty
}

func getLookupsForDeploy(data interface{}, tableMetadata *tableMetadata, dataPath string, tableAliasCache map[string]string) []Lookup {
	lookupsToUse := []Lookup{}
	tableName := tableMetadata.tableName
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	primaryKeyFieldName := tableMetadata.getPrimaryKeyFieldName()
	foreignKeys := tableMetadata.foreignKeys
	lookups := tableMetadata.lookups

	hasValidPK := false
	// Determine which lookups are necessary based on whether keys exist in the data
	s := reflect.ValueOf(data)

	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)

		if dataPath != "" {
			item = item.FieldByName(dataPath)
		}

		// If any piece of data has a primary key we will assume that the data set
		// contains records with the primary key included. We can then just use the
		// primary key to do the lookup.
		pkValue := item.FieldByName(primaryKeyFieldName)
		if pkValue.IsValid() && pkValue.String() != "" && !hasValidPK {
			hasValidPK = true
			lookupsToUse = append([]Lookup{
				{
					TableName:           tableName,
					MatchDBColumn:       primaryKeyColumnName,
					MatchObjectProperty: primaryKeyFieldName,
					Query:               true,
				},
			}, lookupsToUse...)
		}

		for index := range foreignKeys {
			foreignKey := &foreignKeys[index]
			fkValue := item.FieldByName(foreignKey.FieldName)
			if fkValue.IsValid() && fkValue.String() != "" && foreignKey.NeedsLookup {
				foreignKey.NeedsLookup = false
				lookupsToUse = append(lookupsToUse, Lookup{
					MatchDBColumn:       foreignKey.KeyColumn,
					MatchObjectProperty: foreignKey.FieldName,
					Query:               true,
				})
			}
		}
	}

	if !hasValidPK {
		lookupsToUse = append(lookups, lookupsToUse...)
	}

	lookupsToUse = append(lookupsToUse, getLookupsFromForeignKeys(foreignKeys, "", "", tableAliasCache)...)

	return lookupsToUse
}

func getLookupObjectKeys(data interface{}, lookupsToUse []Lookup, dataPath string) []string {
	keys := []string{}
	s := reflect.ValueOf(data)
	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)

		if dataPath != "" {
			item = item.FieldByName(dataPath)
		}
		isZeroField := reflect.DeepEqual(item.Interface(), reflect.Zero(item.Type()).Interface())
		if isZeroField {
			continue
		}
		// Determine the where values that we need for this lookup
		keys = append(keys, getObjectKeyReflect(item, lookupsToUse))
	}
	return keys
}

func getQueryParts(tableName string, primaryKeyColumnName string, lookupsToUse []Lookup, tableAliasCache map[string]string) ([]string, []string, []squirrel.Eq) {
	joinMap := map[string]bool{}
	columns := []string{}
	joins := []string{}
	whereFields := []squirrel.Eq{}

	for _, lookup := range lookupsToUse {
		tableToUse := tableName
		if lookup.TableName != "" {
			tableToUse = lookup.TableName
		}
		tableAlias := tableToUse
		if lookup.JoinKey != "" && lookup.TableName != "" && tableToUse != tableName {
			tableAlias = getTableAlias(tableToUse, lookup.JoinKey, tableAliasCache)
			_, alreadyAddedJoin := joinMap[tableAlias]
			if !alreadyAddedJoin {
				joinMap[tableAlias] = true
				joins = append(joins, fmt.Sprintf("%[1]v as %[4]v on %[4]v.%[2]v = %[3]v", tableToUse, primaryKeyColumnName, lookup.JoinKey, tableAlias))
			}
		}
		columns = append(columns, fmt.Sprintf("%[3]v.%[2]v as %[3]v_%[2]v", tableToUse, lookup.MatchDBColumn, tableAlias))
		if lookup.Query {
			whereFields = append(whereFields, squirrel.Eq{fmt.Sprintf("%v.%v", tableAlias, lookup.MatchDBColumn): lookup.Value})
		}
	}
	return columns, joins, whereFields
}

func (p PersistenceORM) performChildUpserts(changeObjects []DBChange, tableMetadata *tableMetadata) error {

	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()

	for _, child := range tableMetadata.children {

		var data reflect.Value

		if child.FieldKind == reflect.Slice {
			// Creates a new Slice of the same type of elements that were stored in the slice of data.
			data = reflect.New(child.FieldType).Elem()
		} else if child.FieldKind == reflect.Map {
			// Creates a new Slice of the type of elements that were stored in the map of data.
			data = reflect.New(reflect.SliceOf(child.FieldType.Elem())).Elem()
		}

		for _, changeObject := range changeObjects {
			// Add the id of the parent to any foreign keys on the child
			originalValue := changeObject.originalValue
			childValue := originalValue.FieldByName(child.FieldName)
			foreignKeyValue := changeObject.changes[primaryKeyColumnName]

			if childValue.Kind() == reflect.Slice {
				for i := 0; i < childValue.Len(); i++ {
					value := childValue.Index(i)
					if child.ForeignKey != "" {
						keyField := getValueFromLookupString(value, child.ForeignKey)
						keyField.SetString(foreignKeyValue.(string))
					}
					data = reflect.Append(data, value)
				}
			} else if childValue.Kind() == reflect.Map {
				mapKeys := childValue.MapKeys()
				for index, key := range mapKeys {
					value := childValue.MapIndex(key)
					data = reflect.Append(data, value)
					addressibleData := data.Index(index)
					if child.ForeignKey != "" {
						valueToChange := getValueFromLookupString(addressibleData, child.ForeignKey)
						valueToChange.SetString(foreignKeyValue.(string))
					}
					if len(child.KeyMappings) > 0 {
						for _, keyMapping := range child.KeyMappings {
							valueToChange := getValueFromLookupString(addressibleData, keyMapping)
							valueToChange.SetString(key.String())
						}
					}
					if len(child.ValueMappings) > 0 {
						for valueLocation, valueDestination := range child.ValueMappings {
							valueToChange := getValueFromLookupString(addressibleData, valueDestination)
							valueToSet := getValueFromLookupString(originalValue, valueLocation)
							valueToChange.SetString(valueToSet.String())
						}
					}
				}
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
	tableMetadata *tableMetadata,
) (
	*DBChangeSet,
	error,
) {
	foreignKeys := tableMetadata.foreignKeys
	insertsHavePrimaryKey := false
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()

	results, lookups, err := p.checkForExisting(data, tableMetadata, "")
	if err != nil {
		return nil, err
	}

	for index := range foreignKeys {
		foreignKey := &foreignKeys[index]
		foreignResults, foreignLookupsUsed, err := p.checkForExisting(data, foreignKey.TableMetadata, foreignKey.RelatedFieldName)
		if err != nil {
			return nil, err
		}
		foreignKey.LookupResults = foreignResults
		foreignKey.LookupsUsed = foreignLookupsUsed
	}

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

		dbChange, err := p.processObject(value, existingObj, foreignKeys, tableMetadata)

		if err != nil {
			return nil, err
		}

		if dbChange.changes == nil {
			continue
		}

		if existingObj != nil {
			updates = append(updates, dbChange)
		} else {
			if !insertsHavePrimaryKey && dbChange.changes[primaryKeyColumnName] != nil {
				insertsHavePrimaryKey = true
			}
			inserts = append(inserts, dbChange)
		}
	}

	return &DBChangeSet{
		inserts:               inserts,
		updates:               updates,
		deletes:               deletes,
		insertsHavePrimaryKey: insertsHavePrimaryKey,
	}, nil
}

func serializeJSONBColumns(columns []string, returnObject map[string]interface{}) error {
	for _, column := range columns {
		value := returnObject[column]

		// No value to process
		if value == nil || value == "" {
			continue
		}

		serializedValue, err := json.Marshal(value)
		if err != nil {
			return err
		}

		returnObject[column] = serializedValue
	}
	return nil
}

func isFieldDefinedOnStruct(modelMetadata Metadata, fieldName string) bool {
	// Check for nil here instead of the length of the slice.
	// The decode method in picard sets defined fields to an empty slice if it has been run.
	if modelMetadata.DefinedFields == nil {
		return true
	}

	for _, definedFieldName := range modelMetadata.DefinedFields {
		if definedFieldName == fieldName {
			return true
		}
	}
	return false
}

func (p PersistenceORM) processObject(
	metadataObject reflect.Value,
	databaseObject map[string]interface{},
	foreignKeys []ForeignKey,
	tableMetadata *tableMetadata,
) (DBChange, error) {
	returnObject := map[string]interface{}{}

	isUpdate := databaseObject != nil

	// Get Defined Fields if they exist
	modelMetadata := getMetadataFromPicardStruct(metadataObject)

	for _, field := range tableMetadata.getFields() {
		var returnValue interface{}

		// Don't ever update the primary key or the multitenancy key or "create triggered" audit fields
		if isUpdate && !field.includeInUpdate() {
			continue
		}

		auditType := field.audit

		if auditType != "" {
			if auditType == "created_by" {
				returnValue = p.performedBy
			} else if auditType == "updated_by" {
				returnValue = p.performedBy
			} else if auditType == "created_at" {
				returnValue = time.Now()
			} else if auditType == "updated_at" {
				returnValue = time.Now()
			}
		} else {
			if !isFieldDefinedOnStruct(modelMetadata, field.name) {
				continue
			}
			returnValue = metadataObject.FieldByName(field.name).Interface()
		}

		if !isUpdate && field.isPrimaryKey && (returnValue == nil || returnValue == "") {
			continue
		}

		returnObject[field.columnName] = returnValue
	}

	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()

	if isUpdate {
		returnObject[primaryKeyColumnName] = databaseObject[primaryKeyColumnName]
	} else {
		returnObject[multitenancyKeyColumnName] = p.multitenancyValue
	}

	// Process encrypted columns

	encryptedColumns := tableMetadata.getEncryptedColumns()
	for _, column := range encryptedColumns {
		value := returnObject[column]

		// If value is nil or an empty string, no point in encrypting it.
		if value == nil || value == "" {
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

	// Process JSONB columns that need to be serialized prior to storage
	serializeJSONBColumns(tableMetadata.getJSONBColumns(), returnObject)

	for _, foreignKey := range foreignKeys {
		if returnObject[foreignKey.KeyColumn] != "" {
			continue
		}
		foreignValue := metadataObject.FieldByName(foreignKey.RelatedFieldName)
		key := getObjectKeyReflect(foreignValue, foreignKey.LookupsUsed)
		lookupData, foundLookupData := foreignKey.LookupResults[key]

		if foundLookupData {
			lookupDataInterface := lookupData.(map[string]interface{})
			lookupKeyColumnName := foreignKey.TableMetadata.getPrimaryKeyColumnName()
			returnObject[foreignKey.KeyColumn] = lookupDataInterface[lookupKeyColumnName]
		} else {
			// If it's optional we can just keep going, if it's required, throw an error
			if foreignKey.Required {
				return DBChange{}, errors.New("Missing Required Foreign Key Lookup")
			}
		}
	}

	if !isUpdate {
		if err := validator.New().Struct(metadataObject.Interface()); err != nil {
			return DBChange{}, err
		}
	}

	return DBChange{
		changes:       returnObject,
		originalValue: metadataObject,
	}, nil
}

func getObjectKey(objects map[string]interface{}, tableName string, lookups []Lookup, tableAliasCache map[string]string) string {
	keyValue := []string{}
	for _, lookup := range lookups {
		tableToUse := tableName
		if lookup.TableName != "" {
			tableToUse = lookup.TableName
		}

		tableAlias := tableToUse
		if lookup.JoinKey != "" && lookup.TableName != "" && tableToUse != tableName {
			tableAlias = getTableAlias(tableToUse, lookup.JoinKey, tableAliasCache)
		}

		keyPart := objects[fmt.Sprintf("%v_%v", tableAlias, lookup.MatchDBColumn)]
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
		keyValue = append(keyValue, getObjectProperty(value, lookup.MatchObjectProperty))
	}
	return strings.Join(keyValue, separator)
}

func getValueFromLookupString(value reflect.Value, lookupString string) reflect.Value {
	// If the lookupString has a dot in it, recursively look up the property's value
	propertyKeys := strings.Split(lookupString, ".")
	if len(propertyKeys) > 1 {
		subValue := value.FieldByName(propertyKeys[0])
		return getValueFromLookupString(subValue, strings.Join(propertyKeys[1:], "."))
	}
	return value.FieldByName(lookupString)
}

func getObjectProperty(value reflect.Value, lookupString string) string {
	return getValueFromLookupString(value, lookupString).String()
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

func getLookupQueryResults(rows *sql.Rows, tableName string, lookups []Lookup, tableAliasCache map[string]string) (map[string]interface{}, error) {

	results, err := getQueryResults(rows)
	if err != nil {
		return nil, err
	}

	resultsMap := map[string]interface{}{}

	for _, v := range results {
		resultsMap[getObjectKey(v, tableName, lookups, tableAliasCache)] = v
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

func (p PersistenceORM) getFilterLookups(filterModelValue reflect.Value, zeroFields []string, parentTableMetadata *tableMetadata, baseJoinKey string, joinKey string, tableAliasCache map[string]string) ([]Lookup, error) {
	filterModelType := filterModelValue.Type()
	tableName := parentTableMetadata.tableName
	lookups := []Lookup{}

	fullJoinKey := getJoinKey(baseJoinKey, joinKey)

	for i := 0; i < filterModelType.NumField(); i++ {
		field := filterModelType.Field(i)
		fieldValue := filterModelValue.FieldByName(field.Name)

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		_, isMultitenancyColumn := picardTags["multitenancy_key"]
		isZeroField := reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(field.Type).Interface())
		kind := fieldValue.Kind()

		isZeroColumn := false
		for _, zeroField := range zeroFields {
			if field.Name == zeroField {
				isZeroColumn = true
			}
		}

		switch {
		case hasColumn && isMultitenancyColumn:
			lookups = append(lookups, Lookup{
				MatchDBColumn:       column,
				MatchObjectProperty: field.Name,
				TableName:           tableName,
				JoinKey:             fullJoinKey,
				Query:               true,
				Value:               p.multitenancyValue,
			})
		case hasColumn && !isZeroField:
			_, isEncrypted := picardTags["encrypted"]
			if isEncrypted {
				return nil, errors.New("cannot perform queries with where clauses on encrypted fields")
			}

			lookups = append(lookups, Lookup{
				MatchDBColumn:       column,
				MatchObjectProperty: field.Name,
				TableName:           tableName,
				JoinKey:             fullJoinKey,
				Query:               true,
				Value:               fieldValue.Interface(),
			})
		case isZeroColumn:
			lookups = append(lookups, Lookup{
				MatchDBColumn:       column,
				MatchObjectProperty: field.Name,
				TableName:           tableName,
				JoinKey:             fullJoinKey,
				Query:               true,
				Value:               reflect.Zero(field.Type).Interface(),
			})
		case kind == reflect.Struct && !isZeroField:
			foreignKeyValue := ""
			for _, foreignKey := range parentTableMetadata.foreignKeys {
				if foreignKey.RelatedFieldName == field.Name {
					foreignKeyValue = foreignKey.KeyColumn
					break
				}
			}

			if foreignKeyValue == "" {
				return nil, errors.New("No Foreign Key Value Found in Struct Tags")
			}
			relatedPicardTags := tableMetadataFromType(fieldValue.Type())
			tableAlias := getTableAlias(tableName, fullJoinKey, tableAliasCache)
			if tableAlias == tableName {
				tableAlias = ""
			}
			relatedLookups, err := p.getFilterLookups(fieldValue, []string{}, relatedPicardTags, tableAlias, foreignKeyValue, tableAliasCache)
			if err != nil {
				return nil, err
			}
			lookups = append(lookups, relatedLookups...)

		}
	}
	return lookups, nil
}

func (p PersistenceORM) generateWhereClausesFromModel(filterModelValue reflect.Value, zeroFields []string) ([]squirrel.Eq, []string, error) {

	filterModelType := filterModelValue.Type()
	tableMetadata := tableMetadataFromType(filterModelType)
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	tableAliasCache := map[string]string{}

	lookups, err := p.getFilterLookups(filterModelValue, zeroFields, tableMetadata, "", "", tableAliasCache)
	if err != nil {
		return nil, nil, err
	}

	_, joins, whereFields := getQueryParts(tableMetadata.tableName, primaryKeyColumnName, lookups, tableAliasCache)

	return whereFields, joins, nil

}
