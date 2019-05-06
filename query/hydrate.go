package query

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/skuid/picard/stringutil"
	"github.com/skuid/picard/tags"
)

/*
Hydrate takes the rows and pops them into the correct struct, in the correct
order. This is usually called after you've built and executed the query model.
*/
func Hydrate(filterModel interface{}, aliasMap map[string]FieldDescriptor, rows *sql.Rows) (interface{}, error) {
	modelVal, err := stringutil.GetStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	// Get the models type and picard tags
	typ := modelVal.Type()
	meta := tags.TableMetadataFromType(typ)

	mapped, err := mapRows2Cols(meta, aliasMap, rows)
	if err != nil {
		return nil, err
	}
	return hydrate(typ, mapped, 0)
}

func hydrate(typ reflect.Type, mapped map[string]map[string]interface{}, counter int) (interface{}, error) {
	meta := tags.TableMetadataFromType(typ)

	model := reflect.Indirect(reflect.New(typ))

	// TODO: Fragile b/c the aliasing is based on the order we run across references.
	// This is a bit of coupling, where we could probably look up the alias somewhere
	// from the query.Table object
	alias := fmt.Sprintf("t%d", counter)

	mappedFields := mapped[fmt.Sprintf(aliasedField, alias, meta.GetTableName())]

	for _, field := range meta.GetFields() {
		fieldVal := mappedFields[field.GetColumnName()]
		if field.IsReference() {

			refTyp := field.GetFieldType()
			// Recursively hydrate this reference field
			refValHydrated, err := hydrate(refTyp, mapped, counter+1)
			if err != nil {
				return nil, err
			}

			modelRefVal, err := stringutil.GetStructValue(refValHydrated)
			if err != nil {
				return nil, err
			}

			model.FieldByName(field.GetName()).Set(modelRefVal)
		} else {
			setFieldValue(&model, field, fieldVal)
		}
	}

	hydratedModel := reflect.ValueOf(model.Addr().Interface()).Elem().Interface()
	return hydratedModel, nil
}

func setFieldValue(model *reflect.Value, field tags.FieldMetadata, value interface{}) {
	reflectedValue := reflect.ValueOf(value)

	if reflectedValue.IsValid() {
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
			model.FieldByName(field.GetName()).Set(reflect.ValueOf(value))
		}
	}
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

	"t0.customer" : {
		"id": 1234,
		"name": "Bob"
	},
	"t1.address": {
		"city": "Chattanooga"
	}


*/
func mapRows2Cols(meta *tags.TableMetadata, aliasMap map[string]FieldDescriptor, rows *sql.Rows) (map[string]map[string]interface{}, error) {
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
			aliasedTbl := fmt.Sprintf(aliasedField, tmap.Alias, tmap.Table)

			if results[aliasedTbl] == nil {
				results[aliasedTbl] = make(map[string]interface{})
			}

			val := columns[i]
			reflectValue := reflect.ValueOf(val)
			reflectTyp := reflectValue.Type()
			if reflectValue.IsValid() && reflectTyp == reflect.TypeOf([]byte(nil)) && reflectValue.Len() == 36 {
				results[aliasedTbl][tmap.Field] = string(val.([]uint8))
			} else {
				results[aliasedTbl][tmap.Field] = val
			}
		}
	}

	return results, nil
}
