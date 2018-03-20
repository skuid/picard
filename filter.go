package picard

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/Masterminds/squirrel"
)

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	filterModelValue, err := getStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, nil)

	if err != nil {
		return nil, err
	}

	results, err := p.doFilterSelect(filterModelValue.Type(), whereClauses, joinClauses)
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

		returnModels = append(returnModels, hydrateModel(filterModelType, tableMetadata, result).Interface())
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
	return model
}
