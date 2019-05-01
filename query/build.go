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
Build takes the filter model and returns a query object. It takes the
multitenancy value, current reflected value, and any tags
*/
func Build(multitenancyVal, model interface{}, associations []tags.Association) (*Table, error) {

	val, err := stringutil.GetStructValue(model)
	if err != nil {
		return nil, err
	}

	typ := val.Type()

	assocs := make(map[string]bool)
	for _, assoc := range associations {
		assocs[assoc.Name] = true
	}

	tbl, err := buildQuery(multitenancyVal, typ, &val, assocs)
	if err != nil {
		return nil, err
	}

	return tbl, nil
}

func reflectTableName(typ reflect.Type) string {
	var tblName string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		typName := field.Type.Name()
		if typName == "Metadata" {
			ptags := tags.GetStructTagsMap(field, "picard")
			return ptags["tablename"]
		}
	}

	return tblName
}

func buildQuery(
	multitenancyVal interface{},
	modelType reflect.Type,
	modelVal *reflect.Value,
	assocs map[string]bool,
) (*Table, error) {
	// Inspect current reflected value, and add select/where clauses

	tableName := reflectTableName(modelType)

	tbl := New(tableName)

	cols := make([]string, 0, modelType.NumField())

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		notZero := false
		var val reflect.Value
		if modelVal != nil {
			val = modelVal.FieldByName(field.Name)
			notZero = !reflectutil.IsZeroValue(val)
		}
		ptags := tags.GetStructTagsMap(field, "picard")
		column, hasColumn := ptags["column"]
		_, isMultitenancyColumn := ptags["multitenancy_key"]
		_, isPK := ptags["primary_key"]
		_, isFK := ptags["foreign_key"]
		_, isReference := ptags["reference"]

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
		case isReference:
			// did you ask for me? If so, let's load a join!
			cols = append(cols, column)
			refTypName := field.Name

			if assocs[refTypName] {
				fmt.Println("in a reference, what's going on here?")
				// Get type, load it as a model so we can build it out
				refTyp := field.Type

				refTbl, err := buildQuery(multitenancyVal, refTyp, nil, nil)
				if err != nil {
					return nil, err
				}

				joinField := ptags["column"]

				joinTbl := tbl.AppendJoin(refTbl.Name, "id", joinField, "left")
				joinTbl.AddColumns(refTbl.columns)
				for _, where := range refTbl.Wheres {
					joinTbl.AddWhere(where.Field, where.Val)
				}

			}

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
