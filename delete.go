package picard

import (
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/skuid/picard/reflectutil"

	"github.com/skuid/picard/query"
	"github.com/skuid/picard/stringutil"
	"github.com/skuid/picard/tags"
)

// DeleteModel will delete models that match the provided struct, ignoring zero values.
// Returns the number of rows affected or an error.
func (porm PersistenceORM) DeleteModel(model interface{}) (int64, error) {

	associations, err := getAssociationsFromModel(model)
	if err != nil {
		return 0, err
	}

	tbl, err := query.Build(porm.multitenancyValue, model, nil)
	if err != nil {
		return 0, err
	}

	dSQL := tbl.DeleteSQL()

	lookupPks := make([]interface{}, 0)
	if len(associations) > 0 {
		_, pk := reflectutil.ReflectTableInfo(reflect.TypeOf(model))
		results, err := porm.FilterModelAssociations(model, associations)
		if err != nil {
			return 0, err
		}

		for _, result := range results {
			val, ok := reflectutil.GetPK(reflect.ValueOf(result))
			if ok {
				lookupPks = append(lookupPks, val.Interface())
			}
		}
		dSQL = dSQL.Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", tbl.Alias, pk): lookupPks,
			},
		)
	}

	if porm.transaction == nil {
		tx, err := GetConnection().Begin()
		if err != nil {
			return 0, err
		}

		porm.transaction = tx
	}

	results, err := dSQL.RunWith(porm.transaction).Exec()
	if err != nil {
		return 0, err
	}

	if err = porm.transaction.Commit(); err != nil {
		return 0, err
	}

	return results.RowsAffected()
}

func getAssociationsFromModel(model interface{}) ([]tags.Association, error) {

	val, err := stringutil.GetStructValue(model)

	if err != nil {
		return nil, err
	}

	return getAssociationsFromValue(val)
}

func getAssociationsFromValue(val reflect.Value) ([]tags.Association, error) {
	associations := make([]tags.Association, 0)

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Model must be a struct in order to get associations. It was a %v instead", val.Kind())
	}

	for i := 0; i < val.Type().NumField(); i++ {
		structField := val.Type().Field(i)
		ptags := tags.GetStructTagsMap(structField, "picard")

		_, isFK := ptags["foreign_key"]

		if isFK {
			if relatedName, ok := ptags["related"]; ok {

				relatedVal := val.FieldByName(relatedName)

				if !reflectutil.IsZeroValue(relatedVal) {
					fieldAssoc := tags.Association{
						Name: relatedName,
					}

					childAssocs, err := getAssociationsFromValue(relatedVal)
					if err != nil {
						return nil, err
					}
					fieldAssoc.Associations = append(fieldAssoc.Associations, childAssocs...)

					associations = append(associations, fieldAssoc)
				}
			}
		}
	}

	return associations, nil
}
