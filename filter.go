package picard

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/Masterminds/squirrel"
)

type FilterAssociations func(reflect.Value, []interface{}) ([]interface{}, error)

func getPrimaryKeyValuesFromResults(primaryKey string, results []interface{}) ([]interface{}, error) {
	var primaryKeyValues []interface{}
	for _, result := range results {
		resultValue, err := getStructValue(result)
		if err != nil {
			return nil, err
		}
		primaryKeyValue := reflect.Indirect(resultValue).FieldByName(primaryKey).Interface()
		primaryKeyValues = append(primaryKeyValues, primaryKeyValue)
	}
	return primaryKeyValues, nil
}

func getForeignKeyValues(childMetadata *tableMetadata, pkField string, pkValues []interface{}) map[string][]interface{} {
	for _, foreignKey := range childMetadata.foreignKeys {
		if foreignKey.TableMetadata.primaryKeyField == pkField {
			return map[string][]interface{}{
				foreignKey.FieldName: pkValues,
			}
		}
	}
	return nil
}

// loadNextLoad creates a hydrated node representing an association with query results
func loadNextNode(head *oneToMany, doNextLoad func() (*oneToMany, bool, error)) (*oneToMany, error) {
	var list *oneToMany
	level := 0
	doNext := true
	for doNext != false {
		// load the next hydrated node
		node, isValidNext, err := doNextLoad()
		if err != nil {
			return nil, err
		}
		if level == 0 {
			list = node
		} else {
			list.Next = node
		}
		doNext = isValidNext
		level++
	}
	return list, nil
}

type loadModel func(modelValue reflect.Value, parentField string, childForeignKeyValues map[string][]interface{}) ([]interface{}, error)

// eagerLoad hydrates query results into association structs
func (p PersistenceORM) eagerLoad(filterModelValue reflect.Value, parentResults []interface{}, assocs associations) (associations, error) {

	var (
		childValue    reflect.Value
		childMetadata *tableMetadata
		loadedAssocs  associations
	)

	topLevelParents := parentResults
	topLevelModelValue := filterModelValue

	for _, a := range assocs {
		al := a.ModelLink
		nextNode := al
		nextLoad := func() (*oneToMany, bool, error) {
			next := reflect.ValueOf(nextNode).Elem()
			if !next.IsValid() {
				// no more valid nodes to traverse
				return nil, false, nil
			}

			parentModelType := filterModelValue.Type()
			parentMetadata := tableMetadataFromType(parentModelType)
			parentPKFieldName := parentMetadata.getPrimaryKeyFieldName()
			parentPKValues, err := getPrimaryKeyValuesFromResults(parentPKFieldName, parentResults)
			if err != nil {
				return nil, false, err
			}
			childValue, childMetadata, _ = parentMetadata.getChildFromParent(nextNode.Name, filterModelValue)

			if !childValue.IsValid() {
				// no more valid children to traverse
				return nil, false, nil
			}
			childForeignKeyValues := getForeignKeyValues(childMetadata, parentPKFieldName, parentPKValues)
			nodeResults, err := p.getFilterModelResults(childValue, parentPKFieldName, childForeignKeyValues)

			// update previous node with results
			nextNode.Data = nodeResults
			prevNode := nextNode

			if err != nil {
				return nil, false, err
			}

			if prevNode.Next == nil {
				// first level association (except the first a loop) means we need old parent results
				// for the next iteration
				parentResults = topLevelParents
				filterModelValue = topLevelModelValue
				return prevNode, false, nil
			} else {
				// nested associations mean we can use previous node results as parent results
				parentResults = nodeResults
				filterModelValue = childValue
			}
			// set next node for next iteration
			nextNode = nextNode.Next
			return prevNode, true, nil
		}

		hydratedModelLink, err := loadNextNode(nextNode, nextLoad)
		if err != nil {
			return nil, err
		}
		a.ModelLink = hydratedModelLink
		loadedAssocs = append(loadedAssocs, a)
	}
	return loadedAssocs, nil
}

func (p PersistenceORM) getFilterModelResults(filterModelValue reflect.Value, newParentField string, childForeignKeyValues map[string][]interface{}) ([]interface{}, error) {
	var zeroFields []string
	whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, zeroFields, childForeignKeyValues)
	if err != nil {
		return nil, err
	}

	filterResults, err := p.doFilterSelect(filterModelValue.Type(), whereClauses, joinClauses)
	if err != nil {
		return nil, err
	}
	return filterResults, nil
}

func (p PersistenceORM) filter(filterModelValue reflect.Value, assocs associations) ([]interface{}, error) {

	// First do the filter with out any of the associations
	results, err := p.getFilterModelResults(filterModelValue, "", nil)
	if err != nil {
		return nil, err
	}

	// eager load is one select query per child model type
	if len(assocs) > 0 {
		loadedAssocs, err := p.eagerLoad(filterModelValue, results, assocs)
		if err != nil {
			return nil, err
		}
		if len(loadedAssocs) > 0 {
			results, err = populateAssociations(loadedAssocs, results)
			if err != nil {
				return nil, err
			}
		}
	}

	concreteResults := []interface{}{}
	for _, result := range results {
		concreteResults = append(concreteResults, reflect.ValueOf(result).Elem().Interface())
	}

	return concreteResults, nil
}

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}, associations []string) ([]interface{}, error) {
	// root model results
	filterModelValue, err := getStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	// Handle the associations now
	as, err := getAssociations(associations, filterModelValue)
	if err != nil {
		return nil, err
	}

	results, err := p.filter(filterModelValue, as)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (p PersistenceORM) doFilterSelect(filterModelType reflect.Type, whereClauses []squirrel.Eq, joinClauses []string) ([]interface{}, error) {
	var returnModels []interface{}
	db := GetConnection()

	tableMetadata := tableMetadataFromType(filterModelType)
	columnNames := tableMetadata.getColumnNames()
	tableName := tableMetadata.tableName

	fullColumnNames := []string{}

	for _, columnName := range columnNames {
		fullColumnNames = append(fullColumnNames, tableName+"."+columnName)
	}

	// Do select query with provided where clauses and columns/tablename
	query := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select(fullColumnNames...).
		From(tableName).
		RunWith(db)

	for _, join := range joinClauses {
		query = query.Join(join)
	}

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

		// Decrypt any encrypted columns

		encryptedColumns := tableMetadata.getEncryptedColumns()
		for _, column := range encryptedColumns {
			value := result[column]

			if value == nil || value == "" {
				continue
			}

			valueAsString, ok := value.(string)
			if !ok {
				return nil, errors.New("can only decrypt values which are stored as base64 strings")
			}

			valueAsBytes, err := base64.StdEncoding.DecodeString(valueAsString)
			if err != nil {
				return nil, errors.New("base64 decoding of value failed")
			}

			decryptedValue, err := DecryptBytes(valueAsBytes)
			if err != nil {
				return nil, err
			}

			result[column] = decryptedValue
		}

		hydratedModel := hydrateModel(filterModelType, tableMetadata, result).Interface()
		returnModels = append(returnModels, hydratedModel)
	}

	return returnModels, nil
}

func hydrateModel(modelType reflect.Type, tableMetadata *tableMetadata, values map[string]interface{}) reflect.Value {
	model := reflect.Indirect(reflect.New(modelType))
	for _, field := range tableMetadata.fields {
		value, hasValue := values[field.columnName]
		reflectedValue := reflect.ValueOf(value)

		if hasValue && reflectedValue.IsValid() {
			if field.isJSONB {
				valueString, isString := value.(string)
				if !isString {
					valueString = string(value.([]byte))
				}
				destinationValue := reflect.New(field.fieldType).Interface()
				json.Unmarshal([]byte(valueString), destinationValue)
				value = reflect.Indirect(reflect.ValueOf(destinationValue)).Interface()
			}

			if reflectedValue.Type().ConvertibleTo(field.fieldType) {
				reflectedValue = reflectedValue.Convert(field.fieldType)
				value = reflectedValue.Interface()
			}
			model.FieldByName(field.name).Set(reflect.ValueOf(value))
		}
	}
	return model.Addr()
}
