package picard

import (
	"encoding/base64"
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

	tx, err := GetConnection().Begin()
	if err != nil {
		return nil, err
	}

	p.transaction = tx

	picardTags := picardTagsFromType(filterModelType)
	columnNames := picardTags.ColumnNames()
	tableName := picardTags.TableName()

	// Do select query with provided where clauses and columns/tablename
	query := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select(columnNames...).
		From(tableName).
		RunWith(p.transaction)

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

		encryptedColumns := picardTags.EncryptedColumns()
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

		returnModels = append(returnModels, hydrateModel(filterModelType, result).Interface())
	}

	return returnModels, nil
}

func hydrateModel(modelType reflect.Type, values map[string]interface{}) reflect.Value {
	model := reflect.Indirect(reflect.New(modelType))
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		if hasColumn {
			value, hasValue := values[column]
			reflectedValue := reflect.ValueOf(value)

			if hasValue && reflect.ValueOf(value).IsValid() {

				if reflectedValue.Type().ConvertibleTo(field.Type) {
					reflectedValue = reflectedValue.Convert(field.Type)
					value = reflectedValue.Interface()
				}
				model.FieldByName(field.Name).Set(reflect.ValueOf(value))
			}
		}
	}
	return model
}
