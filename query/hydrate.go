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
func Hydrate(filterModel interface{}, aliasMap map[string]FieldDescriptor, rows *sql.Rows) ([]*reflect.Value, error) {
	modelVal, err := stringutil.GetStructValue(filterModel)
	if err != nil {
		return nil, err
	}

	// Get the models type and picard tags
	typ := modelVal.Type()
	meta := tags.TableMetadataFromType(typ)

	mappedCols, err := mapRows2Cols(meta, aliasMap, rows)
	if err != nil {
		return nil, err
	}

	hydrateds := make([]*reflect.Value, 0, len(mappedCols))
	for _, mapped := range mappedCols {
		hydrated, err := hydrate(typ, mapped, 0)

		if err != nil {
			return nil, err
		}
		hydrateds = append(hydrateds, hydrated)
	}

	return hydrateds, nil
}

func hydrate(typ reflect.Type, mapped map[string]map[string]interface{}, counter int) (*reflect.Value, error) {
	meta := tags.TableMetadataFromType(typ)

	model := reflect.Indirect(reflect.New(typ))

	alias := fmt.Sprintf("t%d", counter)

	mappedFields := mapped[fmt.Sprintf(aliasedField, alias, meta.GetTableName())]

	for _, field := range meta.GetFields() {
		fieldVal := mappedFields[field.GetColumnName()]
		setFieldValue(&model, field, fieldVal)
		if field.IsFK() {
			refTyp := field.GetRelatedType()
			fmt.Printf("refTyp: %v\n", refTyp)
			// Recursively hydrate this reference field
			refValHydrated, err := hydrate(refTyp, mapped, counter+1)
			if err != nil {
				return nil, err
			}

			model.FieldByName(field.GetRelatedName()).Set(*refValHydrated)
		}
	}

	hydratedModel := reflect.ValueOf(model.Addr().Interface()).Elem()
	return &hydratedModel, nil
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
			rval := reflect.Indirect(reflect.ValueOf(destinationValue))
			model.FieldByName(field.GetName()).Set(rval)
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
func mapRows2Cols(meta *tags.TableMetadata, aliasMap map[string]FieldDescriptor, rows *sql.Rows) ([]map[string]map[string]interface{}, error) {
	results := make([]map[string]map[string]interface{}, 0)

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

		result := make(map[string]map[string]interface{})

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		for i, colName := range cols {
			tmap := aliasMap[colName]
			aliasedTbl := fmt.Sprintf(aliasedField, tmap.Alias, tmap.Table)

			if result[aliasedTbl] == nil {
				result[aliasedTbl] = make(map[string]interface{})
			}

			val := columns[i]
			reflectValue := reflect.ValueOf(val)
			reflectTyp := reflectValue.Type()
			if reflectValue.IsValid() && reflectTyp == reflect.TypeOf([]byte(nil)) && reflectValue.Len() == 36 {
				result[aliasedTbl][tmap.Field] = string(val.([]uint8))
			} else {
				result[aliasedTbl][tmap.Field] = val
			}
		}

		results = append(results, result)
	}

	return results, nil
}
