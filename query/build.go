package query

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/skuid/picard/reflectutil"
	"github.com/skuid/picard/stringutil"
	"github.com/skuid/picard/tags"
)

/*
Build takes the filter model and returns a query object
Take the multitenancy value, current reflected value, and any tags
*/
func Build(multitenancyVal, model interface{}, associations []tags.Association) (*Table, error) {
	modelVal, err := stringutil.GetStructValue(model)
	if err != nil {
		return nil, err
	}

	modelType := modelVal.Type()
	meta := tags.TableMetadataFromType(modelType)

	return buildQuery(multitenancyVal, modelVal, modelType, meta)
}

func buildQuery(multitenancyVal interface{}, modelVal reflect.Value, modelType reflect.Type, meta *tags.TableMetadata) (*Table, error) {
	// Inspect current reflected value, and add select/where clauses

	tableName := meta.GetTableName()
	// pk := meta.GetPrimaryKeyColumnName()

	tbl := New(tableName)

	cols := make([]string, 0, modelType.NumField())

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		val := modelVal.FieldByName(field.Name)
		ptags := tags.GetStructTagsMap(field, "picard")
		column, hasColumn := ptags["column"]
		_, isMultitenancyColumn := ptags["multitenancy_key"]
		_, isPK := ptags["primary_key"]
		_, isFK := ptags["foreign_key"]

		kind := val.Kind()

		notZero := !reflectutil.IsZeroValue(val)

		fmt.Printf("got kind: %v", kind)

		if !hasColumn {
			continue
		}

		switch {
		case isMultitenancyColumn:
			cols = append(cols, column)
			tbl.AddWhere(column, multitenancyVal)
		case isFK:
			fallthrough
		case isPK:
			cols = append(cols, column)
		case notZero:
			_, isEncrypted := ptags["encrypted"]
			if isEncrypted {
				return nil, errors.New("cannot perform queries with where clauses on encrypted fields")
			}
			cols = append(cols, column)
			tbl.AddWhere(column, val.Interface())
		}
	}

	tbl.AddColumns(cols)

	return tbl, nil

}
