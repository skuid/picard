package picard

import (
	"reflect"

	"github.com/Masterminds/squirrel"
)

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	filterModelValue, err := getStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	whereClauses := p.generateFilterWhereClauses(filterModelValue, nil)

	results, err := p.doFilterSelect(filterModelValue.Type(), whereClauses)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (p PersistenceORM) doFilterSelect(filterModelType reflect.Type, whereClauses []squirrel.Eq) ([]interface{}, error) {
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

func (p PersistenceORM) generateFilterWhereClauses(filterModelValue reflect.Value, zeroFields []string) []squirrel.Eq {
	var returnClauses []squirrel.Eq

	t := filterModelValue.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := filterModelValue.FieldByName(field.Name)

		picardTags := getStructTagsMap(field, "picard")
		column, hasColumn := picardTags["column"]
		_, isMultitenancyColumn := picardTags["multitenancy_key"]
		isZeroField := reflect.DeepEqual(fieldValue.Interface(), reflect.Zero(field.Type).Interface())

		isZeroColumn := false
		for _, zeroField := range zeroFields {
			if field.Name == zeroField {
				isZeroColumn = true
			}
		}

		switch {
		case hasColumn && isMultitenancyColumn:
			returnClauses = append(returnClauses, squirrel.Eq{column: p.multitenancyValue})
		case hasColumn && !isZeroField:
			returnClauses = append(returnClauses, squirrel.Eq{column: fieldValue.Interface()})
		case isZeroColumn:
			returnClauses = append(returnClauses, squirrel.Eq{column: reflect.Zero(field.Type).Interface()})
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
		if hasColumn {
			value, hasValue := values[column]
			if hasValue && reflect.ValueOf(value).IsValid() {
				model.FieldByName(field.Name).Set(reflect.ValueOf(value))
			}
		}
	}
	return model
}
