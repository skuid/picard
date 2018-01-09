package picard

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/Masterminds/squirrel"
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

// StructMetadata is a field type that is used for adding metadata to a struct
// through struct tags
type StructMetadata bool

// Picard provides the necessary configuration to perform an upsert
// of an object into a relational database along with key lookups and field name
// transformations.
type Picard struct {
	DBRecordKey    string
	DeleteExisting bool
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Transaction    *sql.Tx
}

// New Creates a new Picard Object and handle defaults
func New(orgID uuid.UUID, userID uuid.UUID) Picard {
	return Picard{
		DBRecordKey:    "id",
		DeleteExisting: false,
		OrganizationID: orgID,
		UserID:         userID,
	}
}

func getStructValue(v interface{}) (reflect.Value, error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Struct {
		return value, errors.New("Models must be structs")
	}
	return value, nil
}

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p Picard) FilterModel(filterModel interface{}) ([]interface{}, error) {
	filterModelValue, err := getStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	whereClauses := p.generateFilterWhereClauses(filterModelValue, true)

	results, err := p.doFilterSelect(filterModelValue.Type(), whereClauses)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (p Picard) doFilterSelect(filterModelType reflect.Type, whereClauses []squirrel.Eq) ([]interface{}, error) {
	var returnModels []interface{}

	tx, err := GetConnection().Begin()
	if err != nil {
		return nil, err
	}

	p.Transaction = tx

	_, _, columnNames, tableName := getAdditionalOptionsFromSchema(filterModelType)

	// Do select query with provided where clauses and columns/tablename
	columnNames = append(columnNames, p.DBRecordKey)
	query := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).Select(columnNames...).
		From(tableName).
		RunWith(p.Transaction)

	for _, where := range whereClauses {
		query = query.Where(where)
	}

	rows, err := query.Query()

	if err != nil {
		return nil, err
	}
	results, err := getQueryResults(rows)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		returnModels = append(returnModels, hydrateModel(filterModelType, result).Interface())
	}

	return returnModels, nil
}

func (p Picard) generateFilterWhereClauses(filterModelValue reflect.Value, ignoreZero bool) []squirrel.Eq {
	returnClauses := []squirrel.Eq{
		squirrel.Eq{
			"organization_id": p.OrganizationID,
		},
	}

	t := filterModelValue.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := filterModelValue.FieldByName(field.Name)

		// If we're ignoring zero-valued fields, and this field is such a field, continue
		if ignoreZero && reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(field.Type).Interface()) {
			continue
		}

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		if hasColumn {
			returnClauses = append(returnClauses, squirrel.Eq{column: fieldValue.Interface()})
		}
		_, isPK := picardTags["pk"]
		if isPK {
			returnClauses = append(returnClauses, squirrel.Eq{p.DBRecordKey: fieldValue.Interface()})
		}

	}
	return returnClauses
}

func hydrateModel(modelType reflect.Type, values map[string]interface{}) reflect.Value {
	model := reflect.Indirect(reflect.New(modelType))
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		_, isPK := picardTags["pk"]
		if hasColumn {
			value, hasValue := values[column]
			if hasValue && reflect.ValueOf(value).IsValid() {
				model.FieldByName(field.Name).Set(reflect.ValueOf(value))
			}
		}
		if isPK {
			value := values["id"]
			if reflect.ValueOf(value).IsValid() {
				model.FieldByName(field.Name).Set(reflect.ValueOf(value))
			}
		}
	}
	return model
}

// SaveModel performs an upsert operation for the provided model.
func (p Picard) SaveModel(model interface{}) error {
	return p.persistModel(model, false)
}

// CreateModel performs an insert operation for the provided model.
func (p Picard) CreateModel(model interface{}) error {
	return p.persistModel(model, true)
}

// persistModel performs an upsert operation for the provided model.
func (p Picard) persistModel(model interface{}, alwaysInsert bool) error {
	// This makes modelValue a reflect.Value of model whether model is a pointer or not.
	modelValue := reflect.Indirect(reflect.ValueOf(model))
	if modelValue.Kind() != reflect.Struct {
		return errors.New("Models must be structs")
	}
	tx, err := GetConnection().Begin()
	if err != nil {
		return err
	}

	p.Transaction = tx

	primaryKeyValue := getPrimaryKey(modelValue)
	_, _, columnNames, tableName := getAdditionalOptionsFromSchema(modelValue.Type())

	if primaryKeyValue == uuid.Nil || alwaysInsert {
		// Empty UUID: the model needs to insert.
		if err := p.insertModel(modelValue, tableName, columnNames); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Non-Empty UUID: the model needs to update.
		if err := p.updateModel(modelValue, tableName, columnNames); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (p Picard) updateModel(modelValue reflect.Value, tableName string, columnNames []string) error {
	primaryKeyValue := getPrimaryKey(modelValue)
	existingObject, err := p.getExistingObjectByID(tableName, p.DBRecordKey, primaryKeyValue)
	if err != nil {
		return err
	}
	change, err := p.processObject(modelValue, existingObject)
	if err != nil {
		return err
	}
	p.performUpdates([]DBChange{change}, tableName, columnNames)
	return nil
}

func (p Picard) insertModel(modelValue reflect.Value, tableName string, columnNames []string) error {
	change, err := p.processObject(modelValue, nil)
	if err != nil {
		return err
	}
	if err := p.performInserts([]DBChange{change}, tableName, columnNames); err != nil {
		return err
	}
	p.setPrimaryKeyFromInsertResult(modelValue, change)
	return nil
}

func getPrimaryKey(v reflect.Value) uuid.UUID {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		picardFieldTags := getStructTagsMap(field, "picard")

		_, isPrimaryKey := picardFieldTags["pk"]
		if isPrimaryKey {
			primaryKeyUUID := v.FieldByName(field.Name)
			// Ignoring error here because ID should always be uuid
			id, _ := uuid.FromString(primaryKeyUUID.Interface().(string))
			return id
		}

	}
	return uuid.Nil
}

func (p Picard) setPrimaryKeyFromInsertResult(v reflect.Value, change DBChange) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		picardFieldTags := getStructTagsMap(field, "picard")

		_, isPrimaryKey := picardFieldTags["pk"]
		if isPrimaryKey {
			v.FieldByName(field.Name).Set(reflect.ValueOf(change.changes[p.DBRecordKey]))
		}

	}
}

// Deploy is the public method to start a Picard deployment. Send in a table name and a slice of structs
// and it will attempt a deployment.
func (p Picard) Deploy(data interface{}) error {
	tx, err := GetConnection().Begin()
	if err != nil {
		return err
	}

	p.Transaction = tx

	if err = p.upsert(data); err != nil {
		return err
	}

	return tx.Commit()
}

// Upsert takes data in the form of a slice of structs and performs a series of database
// operations that will sync the database with the state of that deployment payload
func (p Picard) upsert(data interface{}) error {

	// Verify that we've been passed valid input
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Slice {
		return errors.New("Can only upsert slices")
	}

	// Get our Lookup Options from the struct schema and tags
	lookups, childOptions, columnNames, tableName := getAdditionalOptionsFromSchema(t.Elem())
	if tableName == "" {
		return errors.New("No table name specified in struct metadata")
	}

	results, err := p.checkForExisting(data, tableName, lookups)
	if err != nil {
		return err
	}

	inserts, updates, _ /*deletes*/, err := p.generateChanges(data, results, lookups)
	if err != nil {
		return err
	}

	// Execute Delete Queries

	// Execute Update Queries
	if err := p.performUpdates(updates, tableName, columnNames); err != nil {
		return err
	}

	// Execute Insert Queries
	if err := p.performInserts(inserts, tableName, columnNames); err != nil {
		return err
	}

	combinedOperations := append(updates, inserts...)

	// Perform Child Upserts
	err = p.performChildUpserts(combinedOperations, childOptions)

	if err != nil {
		return err
	}

	return nil
}

func (p Picard) performUpdates(updates []DBChange, tableName string, columnNames []string) error {
	if len(updates) > 0 {

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		for _, update := range updates {
			changes := update.changes
			updateQuery := psql.Update(tableName)

			values := getColumnValues(columnNames, changes)

			for index, columnName := range columnNames {
				updateQuery = updateQuery.Set(columnName, values[index])
			}

			updateQuery = updateQuery.Where(squirrel.Eq{"organization_id": p.OrganizationID})
			updateQuery = updateQuery.Where(squirrel.Eq{"id": changes[p.DBRecordKey]})

			_, err := updateQuery.RunWith(p.Transaction).Exec()

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p Picard) performInserts(inserts []DBChange, tableName string, columnNames []string) error {
	if len(inserts) > 0 {

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		insertQuery := psql.Insert(tableName)
		insertQuery = insertQuery.Columns(columnNames...)

		for _, insert := range inserts {
			changes := insert.changes
			insertQuery = insertQuery.Values(getColumnValues(columnNames, changes)...)
		}

		insertQuery = insertQuery.Suffix("RETURNING \"" + p.DBRecordKey + "\"")

		rows, err := insertQuery.RunWith(p.Transaction).Query()
		if err != nil {
			return err
		}

		insertResults, err := getQueryResults(rows)
		if err != nil {
			return err
		}

		// Insert our new keys into the change objects
		for index, insert := range inserts {
			insert.changes[p.DBRecordKey] = insertResults[index][p.DBRecordKey]
		}
	}
	return nil
}

func (p Picard) getExistingObjectByID(tableName string, IDColumn string, IDValue uuid.UUID) (map[string]interface{}, error) {
	rows, err := squirrel.Select(fmt.Sprintf("%v.%v", tableName, IDColumn)).
		From(tableName).
		Where(squirrel.Eq{fmt.Sprintf("%v.%v", tableName, IDColumn): IDValue}).
		Where(squirrel.Eq{fmt.Sprintf("%v.organization_id", tableName): p.OrganizationID}).
		RunWith(p.Transaction).
		Query()

	if err != nil {
		return nil, err
	}
	results, err := getQueryResults(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		fmt.Println("Found no existing objects")
	}
	return results[0], nil
}

func (p Picard) checkForExisting(
	data interface{},
	tableName string,
	lookups []Lookup,
) (
	map[string]interface{},
	error,
) {

	query := p.getLookupQuery(data, tableName, lookups)

	rows, err := query.RunWith(p.Transaction).Query()

	if err != nil {
		return nil, err
	}

	return getLookupQueryResults(rows, tableName, lookups)
}

func (p Picard) getLookupQuery(data interface{}, tableName string, lookups []Lookup) *squirrel.SelectBuilder {
	query := squirrel.Select(fmt.Sprintf("%v.%v", tableName, p.DBRecordKey))
	query = query.From(tableName)
	wheres := []string{}
	whereValues := []string{}

	for _, lookup := range lookups {
		if lookup.JoinKey != "" && lookup.TableName != "" {
			query = query.Join(fmt.Sprintf("%[1]v on %[1]v.id = %[2]v", lookup.TableName, lookup.JoinKey))
		}
		tableToUse := tableName
		if lookup.TableName != "" {
			tableToUse = lookup.TableName
		}
		query = query.Column(fmt.Sprintf("%[1]v.%[2]v as %[1]v_%[2]v", tableToUse, lookup.MatchDBColumn))
		if lookup.Query {
			wheres = append(wheres, fmt.Sprintf("%v.%v", tableToUse, lookup.MatchDBColumn))
		}
	}

	s := reflect.ValueOf(data)
	for i := 0; i < s.Len(); i++ {
		whereValues = append(whereValues, getObjectKeyReflect(s.Index(i), lookups))
	}

	query = query.Where(strings.Join(wheres, " || '"+separator+"' || ")+" = ANY($1)", pq.Array(whereValues))
	query = query.Where(fmt.Sprintf("%v.organization_id = $2", tableName), p.OrganizationID)

	return &query
}

func (p Picard) performChildUpserts(changeObjects []DBChange, children []Child) error {

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
				foreignKeyValue := changeObject.changes[p.DBRecordKey]
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
func (p Picard) generateChanges(
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

		// TODO: Implement Delete Existing
		if p.DeleteExisting {

		}

	}

	return inserts, updates, deletes, nil
}

func (p Picard) processObject(
	metadataObject reflect.Value,
	databaseObject map[string]interface{},
) (DBChange, error) {
	returnObject := map[string]interface{}{}

	if databaseObject != nil {
		returnObject[p.DBRecordKey] = databaseObject[p.DBRecordKey]
	}

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

	returnObject["organization_id"] = p.OrganizationID
	returnObject["created_by_id"] = p.UserID
	returnObject["updated_by_id"] = p.UserID
	returnObject["created_at"] = time.Now()
	returnObject["updated_at"] = time.Now()

	// TODO: Implement Foreign Key Merges

	return DBChange{
		changes:       returnObject,
		originalValue: metadataObject,
	}, nil
}

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

func getObjectKey(objects map[string]interface{}, tableName string, lookups []Lookup) string {
	keyValue := []string{}
	for _, lookup := range lookups {
		tableToUse := tableName
		if lookup.TableName != "" {
			tableToUse = lookup.TableName
		}
		keyValue = append(keyValue, objects[fmt.Sprintf("%v_%v", tableToUse, lookup.MatchDBColumn)].(string))
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

func getAdditionalOptionsFromSchema(t reflect.Type) ([]Lookup, []Child, []string, string) {

	lookups := []Lookup{}
	columnNames := []string{}
	children := []Child{}

	var tableName string
	var structMetadata StructMetadata

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		kind := field.Type.Kind()

		picardTags := getStructTagsMap(field, "picard")
		_, isLookup := picardTags["lookup"]
		_, isChild := picardTags["child"]
		columnName, hasColumnName := picardTags["column"]
		_, hasTableName := picardTags["tablename"]

		if hasColumnName && kind != reflect.Map && kind != reflect.Slice {
			columnNames = append(columnNames, columnName)
		}

		if kind == reflect.Slice && isChild {
			children = append(children, Child{
				FieldName:  field.Name,
				FieldType:  field.Type,
				ForeignKey: picardTags["foreign_key"],
			})
		}

		if isLookup {
			lookups = append(lookups, Lookup{
				MatchDBColumn:       picardTags["column"],
				MatchObjectProperty: field.Name,
				Query:               true,
			})
		}

		if field.Type == reflect.TypeOf(structMetadata) && hasTableName {
			tableName = picardTags["tablename"]
		}
	}

	return lookups, children, columnNames, tableName
}

func getStructTagsMap(field reflect.StructField, tagType string) map[string]string {
	tagValue := field.Tag.Get(tagType)
	tags := strings.Split(tagValue, ",")
	tagsMap := map[string]string{}

	for _, v := range tags {
		tagSplit := strings.Split(v, "=")
		tagKey := tagSplit[0]
		tagValue := ""
		if (len(tagSplit)) == 2 {
			tagValue = tagSplit[1]
		}
		tagsMap[tagKey] = tagValue
	}

	return tagsMap
}

func getColumnValues(columnNames []string, data map[string]interface{}) []interface{} {
	columnValues := []interface{}{}
	for _, columnName := range columnNames {
		columnValues = append(columnValues, data[columnName])
	}
	return columnValues
}
