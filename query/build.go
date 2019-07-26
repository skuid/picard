package query

import (
	"errors"
	"reflect"

	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/reflectutil"
	"github.com/skuid/picard/stringutil"
	"github.com/skuid/picard/tags"
)

/*
Build takes the filter model and returns a query object. It takes the
multitenancy value, current reflected value, and any tags
*/
func Build(multitenancyVal, model interface{}, fields qp.SelectFilter, associations []tags.Association) (*qp.Table, error) {

	val, err := stringutil.GetStructValue(model)
	if err != nil {
		return nil, err
	}

	typ := val.Type()

	tbl, err := buildQuery(multitenancyVal, typ, &val, fields, associations, false, 0)
	if err != nil {
		return nil, err
	}

	return tbl, nil
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
- associations: List of associations to load. For references, this will add the
			join to the table at the correct level.
- counter: because record keeping and aliasing is hard, we have to keep track
			of which join we're currently looking at during the recursions.
*/
func buildQuery(
	multitenancyVal interface{},
	modelType reflect.Type,
	modelVal *reflect.Value,
	selectFilter qp.SelectFilter,
	associations []tags.Association,
	onlyJoin bool,
	counter int,
) (*qp.Table, error) {
	// Inspect current reflected value, and add select/where clauses

	tableName, pkName := reflectutil.ReflectTableInfo(modelType)

	tbl := NewIndexed(tableName, counter)

	cols := make([]string, 0, modelType.NumField())
	seen := make(map[string]bool)

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
		_, isFk := ptags["foreign_key"]
		_, isPrimaryKey := ptags["primary_key"]

		addCol := true

		if onlyJoin && !isPrimaryKey {
			addCol = false
		}

		if !hasColumn {
			continue
		}

		switch {
		case isMultitenancyColumn:
			if addCol && !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
			tbl.AddMultitenancyWhere(column, multitenancyVal)
		case isFk:
			relatedName := ptags["related"]
			relatedVal := modelVal.FieldByName(relatedName)

			association, ok := getAssociation(associations, relatedName)

			if addCol && !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}

			if notZero {
				tbl.AddWhere(column, val.Interface())
			}

			// If the association wasn't asked for, but there is a value in the related structure, just join but don't
			// add the fields to the select.
			childOnlyJoin := !ok && !reflectutil.IsZeroValue(relatedVal)

			if ok || childOnlyJoin {
				// Get type, load it as a model so we can build it out
				refTyp := relatedVal.Type()

				refTbl, err := buildQuery(multitenancyVal, refTyp, &relatedVal, qp.SelectFilter{}, association.Associations, childOnlyJoin, counter+1)
				if err != nil {
					return nil, err
				}

				joinField := ptags["column"]

				direction := "left"
				if childOnlyJoin {
					direction = ""
				}
				tbl.AppendJoinTable(refTbl, pkName, joinField, direction)
			}

		case notZero:
			_, isEncrypted := ptags["encrypted"]
			if isEncrypted {
				return nil, errors.New("cannot perform queries with where clauses on encrypted fields")
			}
			if !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
			tbl.AddWhere(column, val.Interface())
		default:
			if addCol && !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
		}
	}

	tbl.AddColumns(cols)

	if tableName == selectFilter.TableName && len(selectFilter.Values) > 0 {
		tbl.AddWhereIn(selectFilter.FieldName, selectFilter.Values)
	}

	return tbl, nil

}
