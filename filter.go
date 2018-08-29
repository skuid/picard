package picard

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/Masterminds/squirrel"
)

func (p PersistenceORM) getFilterModelResults(filterModelValue reflect.Value, filterMetadata *tableMetadata) ([]interface{}, error) {
	var zeroFields []string
	whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, zeroFields, filterMetadata)
	if err != nil {
		return nil, err
	}

	filterResults, err := p.doFilterSelect(filterModelValue.Type(), whereClauses, joinClauses, nil)
	if err != nil {
		return nil, err
	}
	return filterResults, nil
}

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	return p.FilterModelAssociations(filterModel, nil)
}

// FilterModels returns models that match the provided struct, multiple models can be provided
func (p PersistenceORM) FilterModels(filterModels interface{}, transaction *sql.Tx) ([]interface{}, error) {
	s := reflect.ValueOf(filterModels)
	filterMetadata := tableMetadataFromType(s.Type().Elem())
	ors := squirrel.Or{}

	for i := 0; i < s.Len(); i++ {
		filterModelValue := s.Index(i)
		whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, nil, filterMetadata)
		if err != nil {
			return nil, err
		}

		if len(joinClauses) > 0 {
			return nil, errors.New("Cannot filter on related data for multi-filters")
		}

		var ands squirrel.And
		ands = whereClauses
		ors = append(ors, ands)

	}
	filterResults, err := p.doFilterSelect(s.Type().Elem(), ors, nil, transaction)
	if err != nil {
		return nil, err
	}

	concreteResults := []interface{}{}
	for _, result := range filterResults {
		concreteResults = append(concreteResults, reflect.ValueOf(result).Elem().Interface())
	}

	return concreteResults, nil
}

// FilterModelAssociations returns models that match the provide struct and also
// return the requested associated models
func (p PersistenceORM) FilterModelAssociations(filterModel interface{}, associations []Association) ([]interface{}, error) {
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

func (p PersistenceORM) processAssociations(associations []Association, filterModelValue reflect.Value, filterMetadata *tableMetadata, results []interface{}) error {
	for _, association := range associations {
		child := filterMetadata.getChildField(association.Name)
		foreignKey := filterMetadata.getForeignKeyFieldFromRelation(association.Name)
		if child != nil {
			childType := child.FieldType.Elem()
			childMetadata := tableMetadataFromType(childType)
			foreignKey := childMetadata.getForeignKeyField(child.ForeignKey)
			newFilter := reflect.New(childType)
			if foreignKey != nil {
				relatedField := newFilter.Elem().FieldByName(foreignKey.RelatedFieldName)
				relatedField.Set(filterModelValue)
			}
			childResults, err := p.FilterModelAssociations(newFilter.Interface(), association.Associations)
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
			foreignResults, err := p.FilterModelAssociations(newFilter.Interface(), association.Associations)
			if err != nil {
				return err
			}
			populateForeignKeyResults(results, foreignResults, foreignKey)
		}
	}
	return nil
}

func (p PersistenceORM) doFilterSelect(filterModelType reflect.Type, whereClauses []squirrel.Sqlizer, joinClauses []string, transaction *sql.Tx) ([]interface{}, error) {
	var returnModels []interface{}
	var db squirrel.BaseRunner
	if transaction == nil {
		db = GetConnection()
	} else {
		db = transaction
	}

	tableMetadata := tableMetadataFromType(filterModelType)
	columnNames := tableMetadata.getColumnNames()
	tableName := tableMetadata.tableName

	query := createQueryFromParts(tableName, columnNames, joinClauses, whereClauses)

	query = query.PlaceholderFormat(squirrel.Dollar).RunWith(db)

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
	var parentGroupingCriteria []string
	var childGroupingCriteria []string
	if child.GroupingCriteria != nil {
		parentGroupingCriteria = []string{}
		childGroupingCriteria = []string{}
		for childMatchKey, parentMatchKey := range child.GroupingCriteria {
			childGroupingCriteria = append(childGroupingCriteria, childMatchKey)
			parentGroupingCriteria = append(parentGroupingCriteria, parentMatchKey)
		}
	}

	// Attach the results
	for _, childResult := range childResults {
		childValue := reflect.ValueOf(childResult)
		var childMatchValues []reflect.Value
		// Child Match Value
		if childGroupingCriteria != nil {
			for _, childMatchKey := range childGroupingCriteria {
				matchValue := getValueFromLookupString(childValue, childMatchKey)
				childMatchValues = append(childMatchValues, matchValue)
			}
		} else {

			// Just use the foreign key as a default grouping criteria key
			childMatchValues = append(childMatchValues, childValue.FieldByName(child.ForeignKey))
		}

		// Find the parent and attach
		for _, parentResult := range results {
			parentValue := reflect.ValueOf(parentResult)
			var parentMatchValues []reflect.Value

			// Parent Match Value
			if parentGroupingCriteria != nil {
				for _, parentMatchKey := range parentGroupingCriteria {
					matchValue := getValueFromLookupString(parentValue.Elem(), parentMatchKey)
					parentMatchValues = append(parentMatchValues, matchValue)
				}
			} else {
				// Just use the primary key as a default grouping criteria match
				parentMatchValues = append(parentMatchValues, parentValue.Elem().FieldByName(filterMetadata.getPrimaryKeyFieldName()))
			}
			if parentMatchesChild(childMatchValues, parentMatchValues) {
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

func parentMatchesChild(childMatchValues []reflect.Value, parentMatchValues []reflect.Value) bool {
	if len(childMatchValues) != len(parentMatchValues) || len(childMatchValues) == 0 {
		return false
	}
	for i, childMatchValue := range childMatchValues {
		if !(childMatchValue.CanInterface() && parentMatchValues[i].CanInterface()) {
			return false
		}
		if childMatchValue.Interface() != parentMatchValues[i].Interface() {
			return false
		}
	}
	return true
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
