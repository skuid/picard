package picard

import (
	"database/sql"
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/skuid/picard/query"
	"github.com/skuid/picard/tags"
)

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	return p.FilterModelAssociations(filterModel, nil)
}

// FilterModelAssociations returns models that match the provide struct and also
// return the requested associated models
func (p PersistenceORM) FilterModelAssociations(filterModel interface{}, associations []tags.Association) ([]interface{}, error) {
	// root model results
	tbl, err := query.Build(p.multitenancyValue, filterModel, associations)
	if err != nil {
		return nil, err
	}

	rows, err := tbl.BuildSQL().RunWith(GetConnection()).Query()
	if err != nil {
		return nil, err
	}

	aliasMap := tbl.FieldAliases()
	results, err := query.Hydrate(filterModel, aliasMap, rows)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		err := query.FindChildren(GetConnection(), p.multitenancyValue, result, associations)

		if err != nil {
			return nil, err
		}
	}

	ir := make([]interface{}, 0, len(results))
	for _, r := range results {
		ir = append(ir, r.Interface())
	}

	return ir, nil
}

// FilterModels returns models that match the provided struct,
// multiple models can be provided
func (p PersistenceORM) FilterModels(models interface{}, tx *sql.Tx) ([]interface{}, error) {
	mtVal := p.multitenancyValue

	modelVals := reflect.ValueOf(models)

	if modelVals.Len() <= 0 {
		return []interface{}{}, nil
	}

	ors := sq.Or{}
	var tbl *query.Table
	var filterModel interface{}

	for i := 0; i < modelVals.Len(); i++ {
		val := modelVals.Index(i)

		ftbl, err := query.Build(mtVal, val.Interface(), nil)
		if err != nil {
			return nil, err
		}

		if tbl == nil {
			tbl = ftbl
			filterModel = val.Interface()
		}

		ands := sq.And{}

		for _, where := range ftbl.Wheres {
			ands = append(ands, sq.Eq{
				fmt.Sprintf("%v.%v", tbl.Alias, where.Field): where.Val,
			})
		}

		ors = append(ors, ands)

	}

	tbl.Wheres = make([]query.Where, 0)

	sql := tbl.BuildSQL()
	sql = sql.Where(ors)
	rows, err := sql.RunWith(tx).Query()
	if err != nil {
		return nil, err
	}

	aliasMap := tbl.FieldAliases()
	results, err := query.Hydrate(filterModel, aliasMap, rows)
	if err != nil {
		return nil, err
	}

	ir := make([]interface{}, 0, len(results))
	for _, r := range results {
		ir = append(ir, r.Interface())
	}

	return ir, nil
}
