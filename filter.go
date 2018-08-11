package picard

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
)

func (p PersistenceORM) getFilterModelResults(filterModelValue reflect.Value, filterMetadata *tableMetadata) ([]interface{}, error) {
	var zeroFields []string
	whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, zeroFields, filterMetadata)
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

	filterModelType := filterModelValue.Type()
	filterMetadata := tableMetadataFromType(filterModelType)

	// First do the filter with out any of the associations
	results, err := p.getFilterModelResults(filterModelValue, filterMetadata)
	if err != nil {
		return nil, err
	}

	err = p.processAssociations(associations, filterModelValue, filterMetadata, results)
	if err != nil {
		return nil, err
	}

	concreteResults := []interface{}{}
	for _, result := range results {
		concreteResults = append(concreteResults, reflect.ValueOf(result).Elem().Interface())
	}

	return concreteResults, nil
}

func (p PersistenceORM) processAssociations(associations []string, filterModelValue reflect.Value, filterMetadata *tableMetadata, results []interface{}) error {
	for _, association := range associations {
		parts := strings.Split(association, ".")
		firstPart, theRestParts := parts[0], parts[1:]
		child := filterMetadata.getChildField(firstPart)
		foreignKey := filterMetadata.getForeignKeyFieldFromRelation(firstPart)
		if child != nil {
			childType := child.FieldType.Elem()
			childMetadata := tableMetadataFromType(childType)
			foreignKey := childMetadata.getForeignKeyField(child.ForeignKey)
			newFilter := reflect.New(childType)
			relatedField := newFilter.Elem().FieldByName(foreignKey.RelatedFieldName)
			relatedField.Set(filterModelValue)
			childResults, err := p.FilterModel(newFilter.Interface(), []string{strings.Join(theRestParts, ".")})
			if err != nil {
				return err
			}
			populateChildResults(results, childResults, child, filterMetadata)
		} else if foreignKey != nil {
			relationField := filterModelValue.FieldByName(foreignKey.RelatedFieldName)
			relationType := relationField.Type()
			parentRelationField := foreignKey.TableMetadata.getChildFieldFromForeignKey(foreignKey.FieldName, reflect.SliceOf(filterModelValue.Type()))
			newFilter := reflect.New(relationType)
			relatedField := newFilter.Elem().FieldByName(parentRelationField.FieldName)
			relatedField.Set(reflect.Append(relatedField, filterModelValue))
			foreignResults, err := p.FilterModel(newFilter.Interface(), []string{strings.Join(theRestParts, ".")})
			if err != nil {
				return err
			}
			populateForeignKeyResults(results, foreignResults, foreignKey)
		}
	}
	return nil
}

func (p PersistenceORM) doFilterSelect(filterModelType reflect.Type, whereClauses []squirrel.Sqlizer, joinClauses []string) ([]interface{}, error) {
	var returnModels []interface{}
	db := GetConnection()

	tableMetadata := tableMetadataFromType(filterModelType)
	columnNames := tableMetadata.getColumnNames()
	tableName := tableMetadata.tableName

	query := createQueryFromParts(tableName, columnNames, joinClauses, whereClauses)

	query = query.RunWith(db)

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

func populateChildResults(results []interface{}, childResults []interface{}, child *Child, filterMetadata *tableMetadata) {
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
				if child.FieldKind == reflect.Slice {
					parentChildRelField.Set(reflect.Append(parentChildRelField, childValue))
				} else if child.FieldKind == reflect.Map {
					if parentChildRelField.IsNil() {
						parentChildRelField.Set(reflect.MakeMap(child.FieldType))
					}
					keyMappingValue := getValueFromLookupString(childValue, child.KeyMapping)
					parentChildRelField.SetMapIndex(keyMappingValue, childValue)
				}
				break
			}
		}
	}
}

func populateForeignKeyResults(mainResults []interface{}, foreignResults []interface{}, foreignKey *ForeignKey) {
	for _, mainResult := range mainResults {
		mainValue := reflect.ValueOf(mainResult)
		foreignKeyValue := mainValue.Elem().FieldByName(foreignKey.FieldName)
		for _, foreignResult := range foreignResults {
			foreignValue := reflect.ValueOf(foreignResult)
			primaryKeyValue := foreignValue.FieldByName(foreignKey.TableMetadata.getPrimaryKeyFieldName())
			if foreignKeyValue.Interface() == primaryKeyValue.Interface() {
				relField := mainValue.Elem().FieldByName(foreignKey.RelatedFieldName)
				relField.Set(foreignValue)
				break
			}
		}
	}
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
