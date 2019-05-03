package query

import (
	"errors"
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

	tbl, err := buildQuery(multitenancyVal, typ, &val, associations, 0)
	if err != nil {
		return nil, err
	}

	return tbl, nil
}

/*
reflectTableInfo will return the table name and primary key name from the type
*/
func reflectTableInfo(typ reflect.Type) (string, string) {
	var tblName string
	var primaryKey string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		typName := field.Type.Name()
		ptags := tags.GetStructTagsMap(field, "picard")
		if typName == "Metadata" {
			tblName = ptags["tablename"]
		}
		if _, isPK := ptags["primary_key"]; isPK {
			primaryKey = ptags["column"]
		}
	}

	return tblName, primaryKey
}

/*
getAssociation will look through the list of associations and will return one
if it matches the name
*/
func getAssociation(associations []tags.Association, name string) (tags.Association, bool) {
	var found tags.Association
	for _, association := range associations {
		if association.Name == name {
			return association, true
		}
	}
	return found, false
}

/*
buildQuery is called recursively to create a Table object, which can be used
to generate the SQL. It takes
- multitenancyVal: this will be used as a WHERE on every table queried, including joins.
- modelType: This is the reflected type of the struct used for this table's load. It
			is used to figure out which columns to select, joins to add, and wheres.
- modelVal: This is an instance of the struct, holding any lookup values
- assocations: List of associations to load. For references, this will add the
			join to the table at the correct level.
- counter: because record keeping and aliasing is hard, we have to keep track
			of which join we're currently looking at during the recursions.
*/
func buildQuery(
	multitenancyVal interface{},
	modelType reflect.Type,
	modelVal *reflect.Value,
	associations []tags.Association,
	counter int,
) (*Table, error) {
	// Inspect current reflected value, and add select/where clauses

	tableName, pkName := reflectTableInfo(modelType)

	tbl := NewIndexed(tableName, counter)

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
			refTypName := field.Name

			if association, ok := getAssociation(associations, refTypName); ok {
				cols = append(cols, column)
				// Get type, load it as a model so we can build it out
				refTyp := field.Type

				refTbl, err := buildQuery(multitenancyVal, refTyp, nil, association.Associations, counter+1)
				if err != nil {
					return nil, err
				}

				joinField := ptags["column"]

				// TODO: parent field should pull PK from referenced table
				tbl.AppendJoinTable(refTbl, pkName, joinField, "left")

			}

		case notZero:
			_, isEncrypted := ptags["encrypted"]
			if isEncrypted {
				return nil, errors.New("cannot perform queries with where clauses on encrypted fields")
			}
			cols = append(cols, column)
			tbl.AddWhere(column, val.Interface())
		default:
			cols = append(cols, column)
		}
	}

	tbl.AddColumns(cols)

	return tbl, nil

}
