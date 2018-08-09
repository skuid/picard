package picard

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
)

func (p PersistenceORM) getFilterModelResults(filterModelValue reflect.Value) ([]interface{}, error) {
	var zeroFields []string
	whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, zeroFields)
	if err != nil {
		return nil, err
	}

	filterResults, err := p.doFilterSelect(filterModelValue.Type(), whereClauses, joinClauses)
	if err != nil {
		return nil, err
	}
	return filterResults, nil
}

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}, associations []string) ([]interface{}, error) {
	// root model results
	filterModelValue, err := getStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	// First do the filter with out any of the associations
	results, err := p.getFilterModelResults(filterModelValue)
	if err != nil {
		return nil, err
	}

	if len(associations) > 0 {
		for _, association := range associations {
			parts := strings.Split(association, ".")
			firstPart, theRestParts := parts[0], parts[1:]
			filterModelType := filterModelValue.Type()
			filterMetadata := tableMetadataFromType(filterModelType)
			child := filterMetadata.getChildField(firstPart)
			if child != nil {
				childType := child.FieldType.Elem()
				childMetadata := tableMetadataFromType(childType)
				foreignKey := childMetadata.getForeignKeyField(child.ForeignKey)
				newFilter := reflect.New(childType)
				relatedField := newFilter.Elem().FieldByName(foreignKey.RelatedFieldName)
				relatedField.Set(filterModelValue)
				childResults, err := p.FilterModel(newFilter.Interface(), []string{strings.Join(theRestParts, ".")})
				if err != nil {
					return nil, err
				}
				// Attach the results
				for _, childResult := range childResults {
					childValue := reflect.ValueOf(childResult)
					foreignKeyValue := childValue.FieldByName(child.ForeignKey)
					// Find the parent and attach
					for _, parentResult := range results {
						parentValue := reflect.ValueOf(parentResult)
						primaryKeyValue := parentValue.Elem().FieldByName(filterMetadata.getPrimaryKeyFieldName())
						if foreignKeyValue.Interface() == primaryKeyValue.Interface() {
							parentChildRelField := parentValue.Elem().FieldByName(child.FieldName)
							parentChildRelField.Set(reflect.Append(parentChildRelField, childValue))
							break
						}
					}
				}
			}
		}
	}

	concreteResults := []interface{}{}
	for _, result := range results {
		concreteResults = append(concreteResults, reflect.ValueOf(result).Elem().Interface())
	}

	return concreteResults, nil
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
