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

	whereClauses := p.generateWhereClausesFromModel(filterModelValue, nil)

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
