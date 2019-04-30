package query

import (
	"database/sql"
	"encoding/json"
	"reflect"

	"github.com/skuid/picard/stringutil"
	"github.com/skuid/picard/tags"
)

/*
Hydrate takes the rows and pops them into the correct struct, in the correct
order. This is usually called after you've built and executed the query model.
*/
func Hydrate(model interface{}, aliasMap map[string]FieldDescriptor, rows *sql.Rows) ([]interface{}, error) {
	modelVal, err := stringutil.GetStructValue(model)
	if err != nil {
		return nil, err
	}

	// Get the models type and picard tags
	modelType := modelVal.Type()
	meta := tags.TableMetadataFromType(modelType)

	// Map the rows into a map that looks like this:
	// map[tableName][fieldName] = fieldValue
	// This is so that we can remap each table's fields
	// into the proper struct
	mapped, err := mapRows2Cols(aliasMap, rows)
	if err != nil {
		return nil, err
	}

	// Go through each table's fields and build them into a struct
	// representing that table's results
	// TODO: Merge child tables properly
	models := make([]interface{}, 0, len(mapped))
	for _ /*name*/, fields := range mapped {
		models = append(models, hydrateModel(modelType, meta, fields))
	}

	return models, nil
}

/*
mapRows2Cols takes the alias map and the returned sql rows and maps them onto
a map, keyed by table, with each value being a map of column name to value

For example, if the query looks like:

	SELECT t0.id AS "t0.id", t0.name AS "t0.name", t1.city as "t1.city"
	FROM customer as t0
	LEFT JOIN address as t1 ON t0.address_id = t1.id

and it returns:

	t0.id,	t0.name,	t1.city
	1234,	"Bob",		"Chattanooga"

This function would return something like:

	"customer" : {
		"id": 1234,
		"name": "Bob"
	},
	"address": {
		"city": "Chattanooga"
	}


*/
func mapRows2Cols(aliasMap map[string]FieldDescriptor, rows *sql.Rows) (map[string]map[string]interface{}, error) {
	results := make(map[string]map[string]interface{})

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {

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
		for i, colName := range cols {
			tmap := aliasMap[colName]

			if results[tmap.Table] == nil {
				results[tmap.Table] = make(map[string]interface{})
			}

			val := columns[i]
			reflectValue := reflect.ValueOf(val)
			if reflectValue.IsValid() && reflectValue.Type() == reflect.TypeOf([]byte(nil)) && reflectValue.Len() == 36 {
				results[tmap.Table][tmap.Field] = string(val.([]uint8))
			} else {
				results[tmap.Table][tmap.Field] = val
			}
		}
	}

	return results, nil
}

/*
hydrateModel takes the values for one record and turns it into a struct based
on the picard tags

It can use one row for one table and build that one struct
*/
func hydrateModel(modelType reflect.Type, meta *tags.TableMetadata, values map[string]interface{}) interface{} {

	model := reflect.Indirect(reflect.New(modelType))
	for _, field := range meta.GetFields() {
		value, hasValue := values[field.GetColumnName()]
		reflectedValue := reflect.ValueOf(value)

		if hasValue && reflectedValue.IsValid() {
			if field.IsJSONB() {
				valueString, isString := value.(string)
				if !isString {
					valueString = string(value.([]byte))
				}
				destinationValue := reflect.New(field.GetFieldType()).Interface()
				json.Unmarshal([]byte(valueString), destinationValue)
				value = reflect.Indirect(reflect.ValueOf(destinationValue)).Interface()
			}

			if reflectedValue.Type().ConvertibleTo(field.GetFieldType()) {
				reflectedValue = reflectedValue.Convert(field.GetFieldType())
				value = reflectedValue.Interface()
			}
			model.FieldByName(field.GetName()).Set(reflect.ValueOf(value))
		}
	}
	return reflect.ValueOf(model.Addr().Interface()).Elem().Interface()
}
