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
func Build(multitenancyVal, model interface{}, filters tags.Filterable, associations []tags.Association, selectFields []string, filterMetadata *tags.TableMetadata) (*qp.Table, error) {

	val, err := stringutil.GetStructValue(model)
	if err != nil {
		return nil, err
	}

	typ := val.Type()

	counter := 0

	tbl, err := buildQuery(multitenancyVal, typ, &val, filters, associations, selectFields, false, "", filterMetadata, &counter)
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
- filters: Additional filters to add to this query. This allows for more complex conditions
			than a simple modelFilter can provide.
- associations: List of associations to load. For references, this will add the
			join to the table at the correct level.
- selectFields: List of fields to add to the select clause of the query. If this is null,
			add all fields with columns specified to the query.
- onlyJoin: If the association wasn't asked for, but there is a value in the related structure, just join but don't
			add the fields to the select.
- counter: because record keeping and aliasing is hard, we have to keep track
			of which join we're currently looking at during the recursions.
- filterMetadata: Metadata about struct that was passed in in modelVal
*/
func buildQuery(
	multitenancyVal interface{},
	modelType reflect.Type,
	modelVal *reflect.Value,
	filters tags.Filterable,
	associations []tags.Association,
	selectFields []string,
	onlyJoin bool,
	refPath string,
	filterMetadata *tags.TableMetadata,
	counter *int,
) (*qp.Table, error) {
	// Inspect current reflected value, and add select/where clauses

	pkName := filterMetadata.GetPrimaryKeyColumnName()
	tableName := filterMetadata.GetTableName()

	tbl := NewAliased(tableName, stringutil.GenerateTableAlias(counter), refPath)

	cols := make([]string, 0, modelType.NumField())
	seen := make(map[string]bool)

	for _, field := range filterMetadata.GetFields() {
		notZero := false
		var val reflect.Value
		fieldName := field.GetName()
		if modelVal != nil {
			val = modelVal.FieldByName(fieldName)
			notZero = !reflectutil.IsZeroValue(val)
		}
		column := field.GetColumnName()
		isMultitenancyColumn := field.IsMultitenancyKey()
		isFk := field.IsFK()
		isPrimaryKey := field.IsPrimaryKey()

		addCol := true

		if selectFields != nil && !stringutil.StringSliceContainsKey(selectFields, fieldName) {
			addCol = false
		}

		if onlyJoin && !isPrimaryKey {
			addCol = false
		}

		switch {
		case isMultitenancyColumn:
			if addCol && !seen[column] {
				cols = append(cols, column)
				seen[column] = true
			}
			tbl.AddMultitenancyWhere(column, multitenancyVal)
		case isFk:
			relatedName := field.GetRelatedName()
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
				refMetadata := filterMetadata.GetForeignKeyField(fieldName).TableMetadata

				fkRefPath := fieldName
				if refPath != "" {
					fkRefPath = refPath + "." + fieldName
				}

				refTbl, err := buildQuery(multitenancyVal, refTyp, &relatedVal, association.FieldFilters, association.Associations, association.SelectFields, childOnlyJoin, fkRefPath, refMetadata, counter)
				if err != nil {
					return nil, err
				}

				joinField := column

				direction := "left"
				if childOnlyJoin {
					direction = ""
				}
				tbl.AppendJoinTable(refTbl, pkName, joinField, direction)
			}

		case notZero:
			if field.IsEncrypted() {
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

	if filters != nil && modelVal != nil {
		tbl.AddWhereGroup(filters.Apply(tbl, filterMetadata))
	}

	return tbl, nil

}
