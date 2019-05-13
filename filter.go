package picard

import (
	"github.com/skuid/picard/query"
	"github.com/skuid/picard/tags"
)

// // FilterModels returns models that match the provided struct,
// // multiple models can be provided
// func (p PersistenceORM) FilterModels(filterModels interface{}, transaction *sql.Tx) ([]interface{}, error) {
// 	//TODO: Is this used anywhere? I need more context on the purpose of this call, and then to
// 	// run it through the new code.

// 	// looks like it is used for deletes? Wut?
// 	s := reflect.ValueOf(filterModels)
// 	filterMetadata := tags.TableMetadataFromType(s.Type().Elem())
// 	ors := squirrel.Or{}

// 	for i := 0; i < s.Len(); i++ {
// 		filterModelValue := s.Index(i)
// 		whereClauses, joinClauses, err := p.generateWhereClausesFromModel(filterModelValue, nil, filterMetadata)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if len(joinClauses) > 0 {
// 			return nil, errors.New("Cannot filter on related data for multi-filters")
// 		}

// 		var ands squirrel.And
// 		ands = whereClauses
// 		ors = append(ors, ands)
// 	}
// 	finalWhere := squirrel.And{
// 		ors,
// 	}
// 	filterResults, err := p.doFilterSelect(s.Type().Elem(), finalWhere, nil, transaction)
// 	if err != nil {
// 		return nil, err
// 	}

// 	concreteResults := []interface{}{}
// 	for _, result := range filterResults {
// 		concreteResults = append(concreteResults, reflect.ValueOf(result).Elem().Interface())
// 	}

// 	return concreteResults, nil
// }

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	return p.FilterModelAssociations(filterModel, nil)
}

// FilterModelAssociations returns models that match the provide struct and also
// return the requested associated models
func (p PersistenceORM) FilterModelAssociations(filterModel interface{}, associations []tags.Association) ([]interface{}, error) {
	// root model results
	tbl, err := query.Build(p.multitenancyValue, filterModel, associations)
	if err != nil {
		return nil, err
	}

	rows, err := tbl.BuildSQL().RunWith(GetConnection()).Query()
	if err != nil {
		return nil, err
	}

	aliasMap := tbl.FieldAliases()
	results, err := query.Hydrate(filterModel, aliasMap, rows)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		err := query.FindChildren(GetConnection(), p.multitenancyValue, result, associations)

		if err != nil {
			return nil, err
		}
	}

	ir := make([]interface{}, 0, len(results))
	for _, r := range results {
		ir = append(ir, r.Interface())
	}

	return ir, nil
}

// func (p PersistenceORM) doFilterSelect(filterModelType reflect.Type, whereClauses []squirrel.Sqlizer, joinClauses []string, transaction *sql.Tx) ([]interface{}, error) {
// 	var returnModels []interface{}
// 	var db squirrel.BaseRunner
// 	if transaction == nil {
// 		db = GetConnection()
// 	} else {
// 		db = transaction
// 	}

// 	tableMetadata := tags.TableMetadataFromType(filterModelType)
// 	columnNames := tableMetadata.GetColumnNames()
// 	tableName := tableMetadata.GetTableName()

// 	query := createQueryFromParts(tableName, columnNames, joinClauses, whereClauses)

// 	query = query.PlaceholderFormat(squirrel.Dollar).RunWith(db)

// 	rows, err := query.Query()
// 	if err != nil {
// 		return nil, err
// 	}

// 	results, err := getQueryResults(rows)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, result := range results {

// 		// Decrypt any encrypted columns

// 		encryptedColumns := tableMetadata.GetEncryptedColumns()
// 		for _, column := range encryptedColumns {
// 			value := result[column]

// 			if value == nil || value == "" {
// 				continue
// 			}

// 			valueAsString, ok := value.(string)
// 			if !ok {
// 				return nil, errors.New("can only decrypt values which are stored as base64 strings")
// 			}

// 			valueAsBytes, err := base64.StdEncoding.DecodeString(valueAsString)
// 			if err != nil {
// 				return nil, errors.New("base64 decoding of value failed")
// 			}

// 			decryptedValue, err := crypto.DecryptBytes(valueAsBytes)
// 			if err != nil {
// 				return nil, err
// 			}

// 			result[column] = decryptedValue
// 		}

// 		hydratedModel := hydrateModel(filterModelType, tableMetadata, result).Interface()
// 		returnModels = append(returnModels, hydratedModel)
// 	}

// 	return returnModels, nil
// }

// func hydrateModel(modelType reflect.Type, tableMetadata *tags.TableMetadata, values map[string]interface{}) reflect.Value {
// 	model := reflect.Indirect(reflect.New(modelType))
// 	for _, field := range tableMetadata.GetFields() {
// 		value, hasValue := values[field.GetColumnName()]
// 		reflectedValue := reflect.ValueOf(value)

// 		if hasValue && reflectedValue.IsValid() {
// 			if field.IsJSONB() {
// 				valueString, isString := value.(string)
// 				if !isString {
// 					valueString = string(value.([]byte))
// 				}
// 				destinationValue := reflect.New(field.GetFieldType()).Interface()
// 				json.Unmarshal([]byte(valueString), destinationValue)
// 				value = reflect.Indirect(reflect.ValueOf(destinationValue)).Interface()
// 			}

// 			if reflectedValue.Type().ConvertibleTo(field.GetFieldType()) {
// 				reflectedValue = reflectedValue.Convert(field.GetFieldType())
// 				value = reflectedValue.Interface()
// 			}
// 			model.FieldByName(field.GetName()).Set(reflect.ValueOf(value))
// 		}
// 	}
// 	return model.Addr()
// }
