package picard

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/skuid/picard/query"
	"github.com/skuid/picard/tags"
)

const (
	coalesceWhere string = "%s = ANY(?)"
	coalesceSep   string = " || '|' || "
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

func (p PersistenceORM) FilterLookups(models interface{}) (interface{}, error) {

	if reflect.TypeOf(models).Kind() != reflect.Slice {
		return nil, fmt.Errorf("models for lookups must be a slice, but found kind '%v'", reflect.TypeOf(models).Kind())
	}

	vals := reflect.ValueOf(models)

	var lTbl *query.Table
	var model interface{}
	var name string

	lookupVals := make([]string, 0, vals.Len())

	if vals.Len() <= 0 {
		return nil, nil
	}

	for i := 0; i < vals.Len(); i++ {
		val := vals.Index(i)
		if name != "" {
			if val.Type().Name() != name {
				return nil, fmt.Errorf(
					"FilterLookups expects the models slice to have elements that are all of the same type. Expected '%s' but found '%s' at index '%d'",
					name, val.Type().Name(), i,
				)
			}
		} else {
			name = val.Type().Name()
		}

		tbl, err := query.BuildLookups(p.multitenancyValue, val.Interface())
		if err != nil {
			return nil, err
		}

		if lookup := tbl.LookupVals(); lookup != "" {
			lookupVals = append(lookupVals, lookup)
		}

		if lTbl == nil {
			lTbl = tbl
			model = val.Interface()
		}
	}

	bld := lTbl.BuildSQL()

	lookups := lTbl.Lookups()
	if len(lookups) > 0 {
		bld = bld.Where(fmt.Sprintf(coalesceWhere, strings.Join(lookups, coalesceSep)), pq.Array(lookupVals))
	}

	sql, args, err := bld.ToSql()
	if err != nil {
		return nil, err
	}

	fmt.Printf("QUERY: %s\nARGS:%s\n", sql, args)

	lTbl.BuildSQL()

	rows, err := bld.RunWith(GetConnection()).Query()
	if err != nil {
		return nil, err
	}

	aliasMap := lTbl.FieldAliases()
	results, err := query.Hydrate(model, aliasMap, rows)
	if err != nil {
		return nil, err
	}

	return results, nil
}
