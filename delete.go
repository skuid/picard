package picard

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
)

// DeleteModel will delete models that match the provided struct, ignoring zero values.
// Returns the number of rows affected or an error.
func (porm PersistenceORM) DeleteModel(model interface{}) (int64, error) {
	modelValue, err := getStructValue(model)
	if err != nil {
		return 0, err
	}

	tableMetadata := tableMetadataFromType(modelValue.Type())
	tableName := tableMetadata.getTableName()

	whereClauses, joinClauses, err := porm.generateWhereClausesFromModel(modelValue, nil, tableMetadata)
	if err != nil {
		return 0, err
	}

	// Joins not allowed so we need to build WHERE EXISTS clauses with a SELECT subquery for the join
	if len(joinClauses) > 0 {

		var subLookups []Lookup
		for _, joinClause := range joinClauses {
			joinStr := strings.Split(joinClause, " ")
			if len(joinStr) > 0 {
				joinKey := joinStr[len(joinStr)-1] // delete table's primary key column
				matchDBColumn := tableMetadata.getForeignKeyPrimaryKeyColumn(joinKey)
				subLookup := Lookup{
					TableName:     joinStr[0],
					JoinKey:       joinKey,
					MatchDBColumn: matchDBColumn,
				}
				subLookups = append(subLookups, subLookup)
			}
		}

		// separate inner subquery where clauses from outer delete where clauses
		deleteWheres, remainingWheres, err := filterWhereClausesByTableAlias(whereClauses, tableName)
		if err != nil {
			return 0, err
		}

		if len(deleteWheres) > 0 {
			whereClauses = deleteWheres
		}

		for _, sl := range subLookups {
			var (
				args      []interface{}
				subWheres []squirrel.Sqlizer
			)
			tableAliasCache := make(map[string]string)
			subTableAlias := getTableAlias(sl.TableName, sl.JoinKey, tableAliasCache)
			// separate inner subquery where clauses from outer delete where clauses
			subWheres, _, err := filterWhereClausesByTableAlias(remainingWheres, subTableAlias)
			if err != nil {
				return 0, err
			}
			// No subquery option for deletes in squirrel so we need to build the query string
			existsQuery := fmt.Sprintf("EXISTS (SELECT %v.%v FROM %v as %v", subTableAlias, sl.MatchDBColumn, sl.TableName, subTableAlias)
			// first "join" where clause
			existsQuery = existsQuery + fmt.Sprintf(" WHERE %v.%v = %v.%v", subTableAlias, sl.MatchDBColumn, tableName, sl.JoinKey)

			for _, subWhere := range subWheres {
				existsQuery = existsQuery + " AND"
				subWhereSql, subArgs, err := subWhere.ToSql()
				if err != nil {
					return 0, err
				}
				args = append(args, subArgs...)
				existsQuery = existsQuery + " " + subWhereSql
			}

			existsQuery = existsQuery + ")"
			existsExpr := squirrel.Expr(existsQuery, args...)
			whereClauses = append(whereClauses, existsExpr)
		}
	}

	if porm.transaction == nil {
		tx, err := GetConnection().Begin()
		if err != nil {
			return 0, err
		}

		porm.transaction = tx
	}

	deleteStatement := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Question).
		Delete(tableName).
		RunWith(porm.transaction)

	for _, where := range whereClauses {
		deleteStatement = deleteStatement.Where(where)
	}
	results, err := deleteStatement.Exec()
	if err != nil {
		return 0, err
	}

	if err = porm.transaction.Commit(); err != nil {
		return 0, err
	}

	return results.RowsAffected()
}

// filterWhereClausesByAlias takes a group of whereClauses and filters/separates them by the table alias
func filterWhereClausesByTableAlias(whereClauses []squirrel.Sqlizer, tableAlias string) ([]squirrel.Sqlizer, []squirrel.Sqlizer, error) {
	var (
		matchWheres   []squirrel.Sqlizer
		excludeWheres []squirrel.Sqlizer
	)
	whereTableAlias := ""
	for _, whereClause := range whereClauses {
		query, _, err := whereClause.ToSql()
		if err != nil {
			return nil, nil, err
		}
		eqStr := strings.Split(query, ".")
		if len(eqStr) > 0 {
			whereTableAlias = eqStr[0]
		}
		if whereTableAlias == tableAlias {
			matchWheres = append(matchWheres, whereClause)
		} else {
			excludeWheres = append(excludeWheres, whereClause)
		}
	}
	return matchWheres, excludeWheres, nil
}
