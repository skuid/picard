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

	metadata, err := tags.GetTableMetadata(model)
	if err != nil {
		return 0, err
	}

	hasAssociations, err := hasAssociations(model, metadata)
	if err != nil {
		return 0, err
	}

	pkField := metadata.GetPrimaryKeyFieldName()
	pkColumn := metadata.GetPrimaryKeyColumnName()

	tbl, err := query.Build(porm.multitenancyValue, model, nil, nil, nil, metadata)

	if err != nil {
		return 0, err
	}

	dSQL := tbl.DeleteSQL()

	lookupPks := make([]interface{}, 0)
	if hasAssociations {
		results, err := porm.FilterModel(FilterRequest{
			FilterModel:  model,
			SelectFields: []string{pkField},
		})
		if err != nil {
			return 0, err
		}

		for _, result := range results {
			val := getValueFromLookupString(reflect.ValueOf(result), pkField)
			if val.IsValid() {
				lookupPks = append(lookupPks, val.Interface())
			}
		}
		dSQL = dSQL.Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", tbl.Alias, pkColumn): lookupPks,
			},
		)
	}

	if porm.transaction == nil {
		tx, err := GetConnection().Begin()
		if err != nil {
			return 0, err
		}

		porm.transaction = tx
		defer porm.Commit()
	}

	results, err := dSQL.RunWith(porm.transaction).Exec()
	if err != nil {
		porm.Rollback()
		return 0, err
	}

	return results.RowsAffected()
}

func hasAssociations(model interface{}, metadata *tags.TableMetadata) (bool, error) {
	val, err := stringutil.GetStructValue(model)
	if err != nil {
		return false, err
	}

	if val.Kind() != reflect.Struct {
		return false, fmt.Errorf("Model must be a struct in order to get associations. It was a %v instead", val.Kind())
	}

	for _, fkField := range metadata.GetForeignKeys() {
		relatedName := fkField.RelatedFieldName

		if relatedName != "" {
			relatedVal := val.FieldByName(relatedName)
			if !reflectutil.IsZeroValue(relatedVal) {
				return true, nil
			}
		}
	}
	return false, nil
}
