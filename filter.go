package picard

import (
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/skuid/picard/query"
	"github.com/skuid/picard/reflectutil"
	"github.com/skuid/picard/stringutil"
	"github.com/skuid/picard/tags"
)

func (p PersistenceORM) getSingleFilterResults(filterModel interface{}, associations []tags.Association, tx sq.BaseRunner) ([]*reflect.Value, error) {
	tbl, err := query.Build(p.multitenancyValue, filterModel, associations)
	if err != nil {
		return nil, err
	}
	rows, err := tbl.BuildSQL().RunWith(tx).Query()
	if err != nil {
		return nil, err
	}
	aliasMap := tbl.FieldAliases()
	return query.Hydrate(filterModel, aliasMap, rows)
}

func (p PersistenceORM) getMultiFilterResults(modelVal reflect.Value, associations []tags.Association, tx sq.BaseRunner) ([]*reflect.Value, error) {
	mtVal := p.multitenancyValue
	if modelVal.Len() <= 0 {
		return []*reflect.Value{}, nil
	}

	ors := sq.Or{}
	var tbl *query.Table
	var filterModel interface{}

	for i := 0; i < modelVal.Len(); i++ {
		val := modelVal.Index(i)

		ftbl, err := query.Build(mtVal, val.Interface(), associations)
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

		for _, join := range ftbl.Joins {
			for _, where := range join.Table.Wheres {
				ands = append(ands, sq.Eq{
					fmt.Sprintf("%v.%v", join.Table.Alias, where.Field): where.Val,
				})
			}
		}

		ors = append(ors, ands)

	}

	tbl.Wheres = make([]query.Where, 0)

	for _, join := range tbl.Joins {
		join.Table.Wheres = make([]query.Where, 0)
	}

	sql := tbl.BuildSQL()
	sql = sql.Where(ors)
	rows, err := sql.RunWith(tx).Query()
	if err != nil {
		return nil, err
	}
	aliasMap := tbl.FieldAliases()
	return query.Hydrate(filterModel, aliasMap, rows)
}

func (p PersistenceORM) getFilterResults(filterModel interface{}, associations []tags.Association, tx sq.BaseRunner) ([]*reflect.Value, error) {
	modelVal := reflect.ValueOf(filterModel)
	modelKind := modelVal.Kind()
	if modelKind == reflect.Struct {
		return p.getSingleFilterResults(filterModel, associations, tx)
	} else if modelKind == reflect.Slice {
		return p.getMultiFilterResults(modelVal, associations, tx)
	} else if modelKind == reflect.Ptr {
		return p.getFilterResults(modelVal.Elem().Interface(), associations, tx)
	}
	return nil, fmt.Errorf("Filter must be a struct, a slice of structs, or a pointer to a struct or slice of structs")
}

// FilterModel returns models that match the provided struct, ignoring zero values.
func (p PersistenceORM) FilterModel(filterModel interface{}) ([]interface{}, error) {
	return p.FilterModelAssociations(filterModel, nil)
}

// FilterModelAssociations returns models that match the provide struct and also
// return the requested associated models
func (p PersistenceORM) FilterModelAssociations(filterModel interface{}, associations []tags.Association) ([]interface{}, error) {

	results, err := p.getFilterResults(filterModel, associations, GetConnection())
	if err != nil {
		return nil, err
	}

	filterModelType, err := stringutil.GetFilterType(filterModel)
	if err != nil {
		return nil, err
	}

	filterMetadata := tags.TableMetadataFromType(filterModelType)

	for _, association := range associations {
		child := filterMetadata.GetChildField(association.Name)
		if child != nil {
			childType := child.FieldType.Elem()
			childMetadata := tags.TableMetadataFromType(childType)
			foreignKey := childMetadata.GetForeignKeyField(child.ForeignKey)
			newFilterList := reflect.Indirect(reflect.New(reflect.SliceOf(childType)))
			if foreignKey != nil {
				for _, result := range results {
					newFilter := reflect.Indirect(reflect.New(childType))
					pkval, ok := reflectutil.GetPK(*result)
					if !ok {
						return nil, fmt.Errorf("Missing 'primary_key' tag on type '%v'", result.Type().Name())
					}

					if fmf := newFilter.FieldByName(foreignKey.FieldName); fmf.CanSet() {
						fmf.Set(*pkval)
					} else {
						return nil, fmt.Errorf("'foreign_key' field '%s' on 'child' type '%v' is not settable", foreignKey.FieldName, newFilter.Type())
					}
					newFilterList = reflect.Append(newFilterList, newFilter)
				}
			} else if child.GroupingCriteria != nil {
				// By default, we take the primary key from the parent and add it as a filter condition on the
				// foreign key field from the child. However, this adds special funcitonality that maps a set
				// of values on the parent to a set of fields on the child. This mapping is specified in the
				// grouping_criteria metadata.
				for _, result := range results {
					newFilter := reflect.Indirect(reflect.New(childType))
					for childMatchKey, parentMatchKey := range child.GroupingCriteria {
						parentValue := getValueFromLookupString(*result, parentMatchKey)
						if !parentValue.IsValid() {
							return nil, fmt.Errorf("Missing 'grouping_criteria' value on type '%v'", result.Type().Name())
						}

						childValue := getValueFromLookupString(newFilter, childMatchKey)
						if fmf := childValue; fmf.CanSet() {
							fmf.Set(parentValue)
						} else {
							return nil, fmt.Errorf("'grouping_criteria' field '%s' on 'child' type '%v' is not settable", childMatchKey, newFilter.Type())
						}
					}

					newFilterList = reflect.Append(newFilterList, newFilter)
				}
			} else {
				return nil, fmt.Errorf("Missing 'foreign_key' tag or 'grouping_criteria' on child '%s' of type '%v'", association.Name, childType.Name())
			}

			childResults, err := p.FilterModelAssociations(newFilterList.Interface(), association.Associations)
			if err != nil {
				return nil, err
			}
			populateChildResults(results, childResults, child, filterMetadata)
		} else {
			return nil, fmt.Errorf("No child association named '%s' found", association.Name)
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
func (p PersistenceORM) FilterModels(models interface{}, tx sq.BaseRunner) ([]interface{}, error) {
	results, err := p.getFilterResults(models, nil, tx)
	if err != nil {
		return nil, err
	}

	ir := make([]interface{}, 0, len(results))
	for _, r := range results {
		ir = append(ir, r.Interface())
	}

	return ir, nil
}

func populateChildResults(results []*reflect.Value, childResults []interface{}, child *tags.Child, filterMetadata *tags.TableMetadata) {
	var parentGroupingCriteria []string
	var childGroupingCriteria []string
	if child.GroupingCriteria != nil {
		parentGroupingCriteria = []string{}
		childGroupingCriteria = []string{}
		for childMatchKey, parentMatchKey := range child.GroupingCriteria {
			childGroupingCriteria = append(childGroupingCriteria, childMatchKey)
			parentGroupingCriteria = append(parentGroupingCriteria, parentMatchKey)
		}
	}

	// Attach the results
	for _, childResult := range childResults {
		childValue := reflect.ValueOf(childResult)
		var childMatchValues []reflect.Value
		// Child Match Value
		if childGroupingCriteria != nil {
			for _, childMatchKey := range childGroupingCriteria {
				matchValue := getValueFromLookupString(childValue, childMatchKey)
				childMatchValues = append(childMatchValues, matchValue)
			}
		} else {
			// Just use the foreign key as a default grouping criteria key
			childMatchValues = append(childMatchValues, childValue.FieldByName(child.ForeignKey))
		}

		// Find the parent and attach
		for _, parentResult := range results {
			parentValue := *parentResult
			var parentMatchValues []reflect.Value

			// Parent Match Value
			if parentGroupingCriteria != nil {
				for _, parentMatchKey := range parentGroupingCriteria {
					matchValue := getValueFromLookupString(parentValue, parentMatchKey)
					parentMatchValues = append(parentMatchValues, matchValue)
				}
			} else {
				// Just use the primary key as a default grouping criteria match
				parentMatchValues = append(parentMatchValues, parentValue.FieldByName(filterMetadata.GetPrimaryKeyFieldName()))
			}
			if parentMatchesChild(childMatchValues, parentMatchValues) {
				parentChildRelField := parentValue.FieldByName(child.FieldName)
				if child.FieldKind == reflect.Slice {
					parentChildRelField.Set(reflect.Append(parentChildRelField, childValue))
				} else if child.FieldKind == reflect.Map {
					if parentChildRelField.IsNil() {
						parentChildRelField.Set(reflect.MakeMap(child.FieldType))
					}
					keyMappingValue := getValueFromLookupString(childValue, child.KeyMapping)
					parentChildRelField.SetMapIndex(keyMappingValue, childValue)
				}
				break
			}
		}
	}
}

func parentMatchesChild(childMatchValues []reflect.Value, parentMatchValues []reflect.Value) bool {
	if len(childMatchValues) != len(parentMatchValues) || len(childMatchValues) == 0 {
		return false
	}
	for i, childMatchValue := range childMatchValues {
		if !(childMatchValue.CanInterface() && parentMatchValues[i].CanInterface()) {
			return false
		}
		if childMatchValue.Interface() != parentMatchValues[i].Interface() {
			return false
		}
	}
	return true
}
