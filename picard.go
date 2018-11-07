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
	jsoniter "github.com/plusplusben/json-iterator-go"
	"github.com/skuid/picard/dbchange"
	"github.com/skuid/picard/decoding"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/reflectutil"
	validator "gopkg.in/go-playground/validator.v9"
)

const separator = "|"

// Association structure
type Association struct {
	Name         string
	Associations []Association
}

// Lookup structure
type Lookup struct {
	TableName           string
	MatchDBColumn       string
	MatchObjectProperty string
	JoinKey             string
	Value               interface{}
	SubQuery            []Lookup
	SubQueryForeignKey  string
	SubQueryMetadata    *tableMetadata
}

// Child structure
type Child struct {
	FieldName        string
	FieldType        reflect.Type
	FieldKind        reflect.Kind
	ForeignKey       string
	KeyMapping       string
	ValueMappings    map[string]string
	GroupingCriteria map[string]string
	DeleteOrphans    bool
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
	KeyMapField      string
}

// ORM interface describes the behavior API of any picard ORM
type ORM interface {
	FilterModel(interface{}) ([]interface{}, error)
	FilterModelAssociations(interface{}, []Association) ([]interface{}, error)
	SaveModel(model interface{}) error
	CreateModel(model interface{}) error
	DeleteModel(model interface{}) (int64, error)
	Deploy(data interface{}) error
	DeployMultiple(data []interface{}) error
}

// PersistenceORM provides the necessary configuration to perform an upsert of objects without IDs
// into a relational database using lookup fields to match and field name transformations.
type PersistenceORM struct {
	multitenancyValue string
	performedBy       string
	transaction       *sql.Tx
	batchSize         int
}

// New Creates a new Picard Object and handle defaults
func New(multitenancyValue string, performerID string) ORM {
	return PersistenceORM{
		multitenancyValue: multitenancyValue,
		performedBy:       performerID,
		batchSize:         100,
	}
}

// Decode decodes a reader using a specified decoder, but also writes metadata to picard StructMetadata
func Decode(body io.Reader, destination interface{}) error {
	bytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	err = GetDecoder(nil).Unmarshal(bytes, &destination)
	if err != nil {
		return err
	}
	return nil
}

func GetDecoder(config *decoding.Config) jsoniter.API {
	return decoding.GetDecoder(config)
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
	deploys := make([]interface{}, 1)
	deploys[0] = data
	return p.DeployMultiple(deploys)
}

// DeployMultiple allows for doing multiple deployments in the same transaction
func (p PersistenceORM) DeployMultiple(data []interface{}) error {
	tx, err := GetConnection().Begin()
	if err != nil {
		return err
	}

	p.transaction = tx

	for _, dataItem := range data {
		if err = p.upsert(dataItem, nil); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (p PersistenceORM) upsert(data interface{}, deleteFilters interface{}) error {
	tableMetadata, err := getTableMetadata(data)
	if err != nil {
		return err
	}
	dataValue := reflect.ValueOf(data)
	dataCount := dataValue.Len()
	if dataCount > 0 {
		for i := 0; i < dataCount; i += p.batchSize {
			end := i + p.batchSize
			if end > dataCount {
				end = dataCount
			}
			err := p.upsertBatch(dataValue.Slice(i, end).Interface(), deleteFilters, tableMetadata)
			if err != nil {
				return err
			}
		}
	} else {
		// We need to do an empty batch for certain types of deployments like delete existing
		err := p.upsertBatch(data, deleteFilters, tableMetadata)
		if err != nil {
			return err
		}
	}
	return nil
}

// Upsert takes data in the form of a slice of structs and performs a series of database
// operations that will sync the database with the state of that deployment payload
func (p PersistenceORM) upsertBatch(data interface{}, deleteFilters interface{}, tableMetadata *tableMetadata) error {

	changeSet, err := p.generateChanges(data, deleteFilters, tableMetadata)
	if err != nil {
		return err
	}

	// Execute Delete Queries
	if err := p.performDeletes(changeSet.Deletes, tableMetadata); err != nil {
		return err
	}

	// Execute Update Queries
	if err := p.performUpdates(changeSet.Updates, tableMetadata); err != nil {
		return err
	}

	// Execute Insert Queries
	if err := p.performInserts(changeSet.Inserts, changeSet.InsertsHavePrimaryKey, tableMetadata); err != nil {
		return err
	}

	combinedOperations := append(changeSet.Updates, changeSet.Inserts...)

	// Perform Child Upserts
	err = p.performChildUpserts(combinedOperations, tableMetadata)

	if err != nil {
		return err
	}

	return nil
}

func (p PersistenceORM) performDeletes(deletes []dbchange.Change, tableMetadata *tableMetadata) error {
	if len(deletes) > 0 {
		tableName := tableMetadata.tableName
		primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
		multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
		keys := []string{}
		for _, delete := range deletes {
			changes := delete.Changes
			keys = append(keys, changes[primaryKeyColumnName].(string))
		}

		deleteQuery := psql.Delete(tableName)
		deleteQuery = deleteQuery.Where(squirrel.Eq{primaryKeyColumnName: keys})

		if multitenancyKeyColumnName != "" {
			deleteQuery = deleteQuery.Where(squirrel.Eq{multitenancyKeyColumnName: p.multitenancyValue})
		}
		_, err := deleteQuery.RunWith(p.transaction).Exec()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p PersistenceORM) performUpdates(updates []dbchange.Change, tableMetadata *tableMetadata) error {
	if len(updates) > 0 {

		tableName := tableMetadata.tableName

		columnNames := tableMetadata.getColumnNamesWithoutPrimaryKey()

		primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
		multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()

		psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

		for _, update := range updates {
			changes := update.Changes
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

func (p PersistenceORM) performInserts(inserts []dbchange.Change, insertsHavePrimaryKey bool, tableMetadata *tableMetadata) error {
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
			changes := insert.Changes
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
			insert.Changes[primaryKeyColumnName] = insertResults[index][primaryKeyColumnName]
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
	foreignKey *ForeignKey,
) (
	map[string]interface{},
	[]Lookup,
	error,
) {
	tableName := tableMetadata.tableName
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()
	tableAliasCache := map[string]string{}
	lookupsToUse := getLookupsForDeploy(data, tableMetadata, foreignKey, tableAliasCache)
	lookupObjectKeys := getLookupObjectKeys(data, lookupsToUse, foreignKey)

	if len(lookupObjectKeys) == 0 || len(lookupsToUse) == 0 {
		return map[string]interface{}{}, lookupsToUse, nil
	}

	query := squirrel.Select(fmt.Sprintf("%v.%v", tableName, primaryKeyColumnName))
	query = query.From(tableName)

	columns, joins, whereFields := getQueryParts(tableMetadata, lookupsToUse, tableAliasCache)

	for _, join := range joins {
		query = query.Join(join)
	}

	query = query.Columns(columns...)

	if len(whereFields) > 0 {
		wheres := []string{}
		for _, whereField := range whereFields {
			eq, ok := whereField.(squirrel.Eq)
			if ok {
				for whereFieldKey := range eq {
					wheres = append(wheres, fmt.Sprintf("COALESCE(%v::\"varchar\",'')", whereFieldKey))
				}
			}
		}
		query = query.Where(strings.Join(wheres, " || '"+separator+"' || ")+" = ANY(?)", pq.Array(lookupObjectKeys))
	}

	if multitenancyKeyColumnName != "" {
		query = query.Where(fmt.Sprintf("%v.%v = ?", tableName, multitenancyKeyColumnName), p.multitenancyValue)
	}

	rows, err := query.PlaceholderFormat(squirrel.Dollar).RunWith(p.transaction).Query()
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

func getLookupsForDeploy(data interface{}, tableMetadata *tableMetadata, foreignKey *ForeignKey, tableAliasCache map[string]string) []Lookup {
	lookupsToUse := []Lookup{}
	tableName := tableMetadata.tableName
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	primaryKeyFieldName := tableMetadata.getPrimaryKeyFieldName()
	lookups := tableMetadata.lookups

	// Create a new slice of all the foreign keys for this type
	foreignKeysToCheck := tableMetadata.foreignKeys[:]

	hasValidPK := false
	// Determine which lookups are necessary based on whether keys exist in the data
	s := reflect.ValueOf(data)

	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)

		if foreignKey != nil {
			keyMapField := foreignKey.KeyMapField
			keyValue := item.FieldByName(foreignKey.FieldName)
			item = item.FieldByName(foreignKey.RelatedFieldName)
			if keyMapField != "" {
				relatedKeyField := item.FieldByName(keyMapField)
				relatedKeyField.Set(keyValue)
			}
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
				},
			}, lookupsToUse...)
		}

		// Iterate over foreign keys to check in reverse so we can remove items without consequence
		for i := len(foreignKeysToCheck) - 1; i >= 0; i-- {
			foreignKeyToCheck := foreignKeysToCheck[i]
			fkValue := item.FieldByName(foreignKeyToCheck.FieldName)
			if fkValue.IsValid() && fkValue.String() != "" && foreignKeyToCheck.NeedsLookup {
				// Remove this foreign key from the list of foreign keys to check
				if i == 0 {
					foreignKeysToCheck = foreignKeysToCheck[:0]
				} else {
					foreignKeysToCheck = append(foreignKeysToCheck[:i], foreignKeysToCheck[i-1:]...)
				}

				lookupsToUse = append(lookupsToUse, Lookup{
					MatchDBColumn:       foreignKeyToCheck.KeyColumn,
					MatchObjectProperty: foreignKeyToCheck.FieldName,
				})
			}
		}
	}

	if !hasValidPK {
		lookupsToUse = append(lookups, lookupsToUse...)
	}

	lookupsToUse = append(lookupsToUse, getLookupsFromForeignKeys(foreignKeysToCheck, "", "", tableAliasCache)...)

	return lookupsToUse
}

func getLookupObjectKeys(data interface{}, lookupsToUse []Lookup, foreignKey *ForeignKey) []string {
	keys := []string{}
	s := reflect.ValueOf(data)
	emptyKeyLength := len(separator) * len(lookupsToUse)
	for i := 0; i < s.Len(); i++ {
		item := s.Index(i)

		if foreignKey != nil {
			item = item.FieldByName(foreignKey.RelatedFieldName)
		}
		isZeroField := reflect.DeepEqual(item.Interface(), reflect.Zero(item.Type()).Interface())
		if isZeroField {
			continue
		}
		objectKey := getObjectKeyReflect(item, lookupsToUse)
		// If none of the lookups have values, don't do the lookup
		if len(objectKey) == emptyKeyLength-1 {
			continue
		}
		// Determine the where values that we need for this lookup
		keys = append(keys, objectKey)
	}
	return keys
}

func getQueryParts(tableMetadata *tableMetadata, lookupsToUse []Lookup, tableAliasCache map[string]string) ([]string, []string, []squirrel.Sqlizer) {
	tableName := tableMetadata.getTableName()
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	joinMap := map[string]bool{}
	columns := []string{}
	joins := []string{}
	whereFields := []squirrel.Sqlizer{}

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
		if lookup.SubQuery != nil {
			_, joinParts, whereParts := getQueryParts(lookup.SubQueryMetadata, lookup.SubQuery, tableAliasCache)
			subQueryFKField := lookup.SubQueryMetadata.getField(lookup.SubQueryForeignKey)
			subquery := createQueryFromParts(lookup.SubQueryMetadata.getTableName(), []string{subQueryFKField.columnName}, joinParts, whereParts)
			// Use the question mark placeholder format so that no replacements are made.
			sql, args, _ := subquery.ToSql()
			whereFields = append(whereFields, squirrel.Expr(lookup.MatchDBColumn+" IN ("+sql+")", args...))
		} else {
			whereFields = append(whereFields, squirrel.Eq{fmt.Sprintf("%v.%v", tableAlias, lookup.MatchDBColumn): lookup.Value})
		}
	}
	return columns, joins, whereFields
}

func createQueryFromParts(tableName string, columnNames []string, joinClauses []string, whereClauses []squirrel.Sqlizer) squirrel.SelectBuilder {
	fullColumnNames := []string{}

	for _, columnName := range columnNames {
		fullColumnNames = append(fullColumnNames, tableName+"."+columnName)
	}

	// Do select query with provided where clauses and columns/tablename
	query := squirrel.StatementBuilder.
		Select(fullColumnNames...).
		From(tableName)

	for _, join := range joinClauses {
		query = query.Join(join)
	}

	for _, where := range whereClauses {
		query = query.Where(where)
	}
	return query
}

func (p PersistenceORM) performChildUpserts(changeObjects []dbchange.Change, tableMetadata *tableMetadata) error {

	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()

	for _, child := range tableMetadata.children {

		var data reflect.Value
		var deleteFiltersValue reflect.Value
		var deleteFilters interface{}
		index := 0

		if child.FieldKind == reflect.Slice {
			// Creates a new Slice of the same type of elements that were stored in the slice of data.
			data = reflect.New(child.FieldType).Elem()
			deleteFiltersValue = reflect.New(child.FieldType).Elem()
		} else if child.FieldKind == reflect.Map {
			// Creates a new Slice of the type of elements that were stored in the map of data.
			data = reflect.New(reflect.SliceOf(child.FieldType.Elem())).Elem()
			deleteFiltersValue = reflect.New(reflect.SliceOf(child.FieldType.Elem())).Elem()
		}

		for _, changeObject := range changeObjects {
			// Add the id of the parent to any foreign keys on the child
			originalValue := changeObject.OriginalValue
			childValue := originalValue.FieldByName(child.FieldName)
			foreignKeyValue := changeObject.Changes[primaryKeyColumnName]

			if child.DeleteOrphans && !childValue.IsNil() && changeObject.Type == dbchange.Update {
				// If we're doing deletes
				filter := reflect.New(child.FieldType.Elem()).Elem()
				foreignKeyField := filter.FieldByName(child.ForeignKey)
				foreignKeyField.SetString(foreignKeyValue.(string))
				deleteFiltersValue = reflect.Append(deleteFiltersValue, filter)
			}

			if childValue.Kind() == reflect.Slice {
				for i := 0; i < childValue.Len(); i++ {
					value := childValue.Index(i)
					if child.ForeignKey != "" {
						keyField := getValueFromLookupString(value, child.ForeignKey)
						keyField.SetString(foreignKeyValue.(string))
					}
					data = reflect.Append(data, value)
					index = index + 1
				}
			} else if childValue.Kind() == reflect.Map {
				mapKeys := childValue.MapKeys()
				for _, key := range mapKeys {
					value := childValue.MapIndex(key)
					data = reflect.Append(data, value)
					addressibleData := data.Index(index)
					index = index + 1
					if child.ForeignKey != "" {
						valueToChange := getValueFromLookupString(addressibleData, child.ForeignKey)
						valueToChange.SetString(foreignKeyValue.(string))
					}
					if child.KeyMapping != "" {
						valueToChange := getValueFromLookupString(addressibleData, child.KeyMapping)
						valueToChange.SetString(key.String())
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

		if deleteFiltersValue.Len() == 0 {
			deleteFilters = nil
		} else {
			deleteFilters = deleteFiltersValue.Interface()
		}

		err := p.upsert(data.Interface(), deleteFilters)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateChanges takes results from performing lookup and foreign lookup
// queries and creates a set of inserts, updates, and deletes to be
// performed on the database.
func (p PersistenceORM) generateChanges(
	data interface{},
	deleteFilters interface{},
	tableMetadata *tableMetadata,
) (
	*dbchange.ChangeSet,
	error,
) {
	foreignKeys := tableMetadata.foreignKeys
	insertsHavePrimaryKey := false
	primaryKeyColumnName := tableMetadata.getPrimaryKeyColumnName()
	lookupResults, lookups, err := p.checkForExisting(data, tableMetadata, nil)
	if err != nil {
		return nil, err
	}

	for index := range foreignKeys {
		foreignKey := &foreignKeys[index]
		foreignResults, foreignLookupsUsed, err := p.checkForExisting(data, foreignKey.TableMetadata, foreignKey)
		if err != nil {
			return nil, err
		}
		foreignKey.LookupResults = foreignResults
		foreignKey.LookupsUsed = foreignLookupsUsed
	}

	inserts := []dbchange.Change{}
	updates := []dbchange.Change{}
	deletes := []dbchange.Change{}

	s := reflect.ValueOf(data)

	for i := 0; i < s.Len(); i++ {
		value := s.Index(i)
		objectKey := getObjectKeyReflect(value, lookups)

		object := lookupResults[objectKey]

		var existingObj map[string]interface{}

		if object != nil {
			existingObj = object.(map[string]interface{})
		}
		// TODO: Implement Delete Conditions
		shouldDelete := false

		if shouldDelete {
			if existingObj != nil {
				deletes = append(deletes, dbchange.Change{
					Changes: existingObj,
					Type:    dbchange.Delete,
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

		if dbChange.Changes == nil {
			continue
		}

		dbChange.Key = objectKey

		if existingObj != nil {
			dbChange.Type = dbchange.Update
			updates = append(updates, dbChange)

		} else {
			dbChange.Type = dbchange.Insert
			if !insertsHavePrimaryKey && dbChange.Changes[primaryKeyColumnName] != nil {
				insertsHavePrimaryKey = true
			}
			inserts = append(inserts, dbChange)
		}
	}

	if deleteFilters != nil {
		deleteResults, err := p.FilterModels(deleteFilters, p.transaction)
		if err != nil {
			return nil, err
		}

		updateKeyMap := map[string]bool{}
		for _, update := range updates {
			updateKeyMap[update.Key] = true
		}

		for _, result := range deleteResults {
			resultValue := reflect.ValueOf(result)
			compoundObjectKey := getObjectKeyReflect(resultValue, lookups)
			if _, ok := updateKeyMap[compoundObjectKey]; !ok {
				deletes = append(deletes, dbchange.Change{
					Changes: map[string]interface{}{
						tableMetadata.getPrimaryKeyColumnName(): getObjectProperty(resultValue, tableMetadata.getPrimaryKeyFieldName()),
					},
					Type: dbchange.Delete,
				})
			}
		}
	}

	return &dbchange.ChangeSet{
		Inserts:               inserts,
		Updates:               updates,
		Deletes:               deletes,
		InsertsHavePrimaryKey: insertsHavePrimaryKey,
	}, nil
}

func serializeJSONBColumns(columns []string, returnObject map[string]interface{}) error {
	for _, column := range columns {
		value := returnObject[column]

		// No value to process
		if value == nil || value == "" {
			continue
		}

		serializedValue, err := serializeJSONBColumn(value)
		if err != nil {
			return err
		}

		returnObject[column] = serializedValue
	}
	return nil
}

func serializeJSONBColumn(value interface{}) (interface{}, error) {
	// No value to process
	if value == nil || value == "" {
		return value, nil
	}
	return json.Marshal(value)
}

func isFieldDefinedOnStruct(modelMetadata metadata.Metadata, fieldName string, data reflect.Value) bool {
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
	// Finally, check to see if we have a non-zero value in the struct for this field
	// If so, it doesn't matter if it's in our defined list, it's defined
	fieldValue := data.FieldByName(fieldName)
	if !reflectutil.IsZeroValue(fieldValue) {
		return true
	}
	return false
}

func (p PersistenceORM) processObject(
	metadataObject reflect.Value,
	databaseObject map[string]interface{},
	foreignKeys []ForeignKey,
	tableMetadata *tableMetadata,
) (dbchange.Change, error) {
	returnObject := map[string]interface{}{}

	isUpdate := databaseObject != nil

	// Get Defined Fields if they exist
	modelMetadata := metadata.GetMetadataFromPicardStruct(metadataObject)

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
			if !isFieldDefinedOnStruct(modelMetadata, field.name, metadataObject) {
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
				return dbchange.Change{}, errors.New("can only encrypt values that can be converted to bytes")
			}
			valueAsBytes = assertedBytes
		}

		// Do encryption over bytes
		encryptedValue, err := EncryptBytes(valueAsBytes)
		if err != nil {
			return dbchange.Change{}, err
		}

		// Base64 encode to get standard character set
		encoded := base64.StdEncoding.EncodeToString(encryptedValue)

		// Replace the original value with the newly encrypted value.
		returnObject[column] = encoded
	}

	// Process JSONB columns that need to be serialized prior to storage
	serializeJSONBColumns(tableMetadata.getJSONBColumns(), returnObject)

	for _, foreignKey := range foreignKeys {
		fkValue, keyIsDefined := returnObject[foreignKey.KeyColumn]
		if keyIsDefined && fkValue != "" && foreignKey.KeyMapField == "" {
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
				return dbchange.Change{}, errors.New("Missing Required Foreign Key Lookup")
			}
		}
	}

	if !isUpdate {
		if err := validator.New().Struct(metadataObject.Interface()); err != nil {
			return dbchange.Change{}, err
		}
	}

	return dbchange.Change{
		Changes:       returnObject,
		OriginalValue: metadataObject,
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
		columnValue, hasValue := data[columnName]
		if hasValue {
			columnValues = append(columnValues, columnValue)
		} else {
			columnValues = append(columnValues, squirrel.Expr("DEFAULT"))
		}
	}
	return columnValues
}

func (p PersistenceORM) getFilterLookups(filterModelValue reflect.Value, zeroFields []string, parentTableMetadata *tableMetadata, baseJoinKey string, joinKey string, tableAliasCache map[string]string) ([]Lookup, error) {
	filterModelType := filterModelValue.Type()
	tableName := parentTableMetadata.getTableName()
	primaryKeyColumnName := parentTableMetadata.getPrimaryKeyColumnName()
	lookups := []Lookup{}

	fullJoinKey := getJoinKey(baseJoinKey, joinKey)

	for i := 0; i < filterModelType.NumField(); i++ {
		field := filterModelType.Field(i)
		fieldValue := filterModelValue.FieldByName(field.Name)

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		_, isMultitenancyColumn := picardTags["multitenancy_key"]
		_, isChild := picardTags["child"]

		isZeroField := reflectutil.IsZeroValue(fieldValue)

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
				Value:               fieldValue.Interface(),
			})

		case isZeroColumn:
			lookups = append(lookups, Lookup{
				MatchDBColumn:       column,
				MatchObjectProperty: field.Name,
				TableName:           tableName,
				JoinKey:             fullJoinKey,
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
		case kind == reflect.Slice && isChild && !isZeroField:
			foreignKeyField := picardTags["foreign_key"]

			for i := 0; i < fieldValue.Len(); i++ {
				item := fieldValue.Index(i)
				relatedPicardTags := tableMetadataFromType(item.Type())
				subQueryLookups, err := p.getFilterLookups(item, []string{}, relatedPicardTags, "", "", tableAliasCache)
				if err != nil {
					return nil, err
				}
				lookups = append(lookups, Lookup{
					MatchDBColumn:      primaryKeyColumnName,
					TableName:          tableName,
					SubQuery:           subQueryLookups,
					SubQueryForeignKey: foreignKeyField,
					SubQueryMetadata:   relatedPicardTags,
				})
			}
		}
	}
	return lookups, nil
}

func (p PersistenceORM) generateWhereClausesFromModel(filterModelValue reflect.Value, zeroFields []string, tableMetadata *tableMetadata) ([]squirrel.Sqlizer, []string, error) {

	tableAliasCache := map[string]string{}

	lookups, err := p.getFilterLookups(filterModelValue, zeroFields, tableMetadata, "", "", tableAliasCache)
	if err != nil {
		return nil, nil, err
	}
	_, joins, whereFields := getQueryParts(tableMetadata, lookups, tableAliasCache)

	return whereFields, joins, nil

}
