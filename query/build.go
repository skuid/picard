package query

import (
	"database/sql"
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

	tbl, err := buildQuery(multitenancyVal, typ, &val, associations, 0)
	if err != nil {
		return nil, err
	}

	return tbl, nil
}

/*
FindChildren will call all child tables recursively and load them into the
appropriate place in the object's hierarchy
*/
func FindChildren(db *sql.DB, mtk string, val *reflect.Value, associations []tags.Association) error {
	typ := val.Type()

	for _, association := range associations {
		field := val.FieldByName(association.Name)
		if structField, ok := typ.FieldByName(association.Name); ok {
			ptags := tags.GetStructTagsMap(structField, "picard")

			if _, yes := ptags["child"]; yes {
				fk, ok := ptags["foreign_key"]

				if !ok {
					return fmt.Errorf("Missing 'foreign_key' tag on child %s", association.Name)
				}

				filterModel := reflect.Indirect(reflect.New(structField.Type.Elem()))

				pkval, ok := reflectutil.GetPK(*val)
				if !ok {
					return fmt.Errorf("Missing 'primary_key' tag on %v", val.Type().Name())
				}

				if fmf := filterModel.FieldByName(fk); fmf.CanSet() {
					fmf.Set(*pkval)
				} else {
					return fmt.Errorf("Error setting 'foreign_key' field '%s' on 'child' type %v", fk, filterModel.Type())
				}

				tbl, err := Build(mtk, filterModel.Interface(), association.Associations)
				if err != nil {
					return err
				}

				rows, err := tbl.BuildSQL().RunWith(db).Query()
				if err != nil {
					return err
				}

				aliasMap := tbl.FieldAliases()
				childResults, err := Hydrate(filterModel.Interface(), aliasMap, rows)
				if err != nil {
					return err
				}

				if field.Kind() == reflect.Slice {
					field.Set(reflect.MakeSlice(field.Type(), len(childResults), len(childResults)))

					for i, cr := range childResults {
						field.Index(i).Set(*cr)
						cf := field.Index(i)
						err = FindChildren(db, mtk, &cf, association.Associations)
						if err != nil {
							return err
						}
					}
				} else if field.Kind() == reflect.Map {
					field.Set(reflect.MakeMap(field.Type()))

					for _, cr := range childResults {
						keyField, ok := ptags["key_mapping"]
						if !ok {
							return fmt.Errorf("Missing 'key_mapping' in the picard tags for this child field")
						}

						key := cr.FieldByName(keyField)

						if !key.IsValid() {
							return fmt.Errorf("Key field '%s' in type %v does not hold a value on the results and can't be used as a map key", keyField, cr.Type())
						}

						field.SetMapIndex(key, *cr)
						err = FindChildren(db, mtk, cr, association.Associations)
						if err != nil {
							return err
						}
					}

				} else {
					return fmt.Errorf("Expected a slice or map for %s, but got kind: %v", association.Name, field.Kind())
				}

			}
		}
	}

	return nil
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
		// _, isPK := ptags["primary_key"]
		// _, isFK := ptags["foreign_key"]
		_, isReference := ptags["reference"]

		if !hasColumn {
			continue
		}

		switch {
		case isMultitenancyColumn:
			if !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
			tbl.AddWhere(column, multitenancyVal)
		case isReference:
			refTypName := field.Name

			if association, ok := getAssociation(associations, refTypName); ok {
				if !seen[column] {
					cols = append(cols, column)
					seen[column] = true
				}
				// Get type, load it as a model so we can build it out
				refTyp := field.Type

				refTbl, err := buildQuery(multitenancyVal, refTyp, &val, association.Associations, counter+1)
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
			if !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
			tbl.AddWhere(column, val.Interface())
		default:
			if !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
		}
	}

	tbl.AddColumns(cols)

	return tbl, nil

}
