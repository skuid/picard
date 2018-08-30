package picard

import (
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
	multitenancyKeyColumnName := tableMetadata.getMultitenancyKeyColumnName()

	whereClauses, joinClauses, err := porm.generateWhereClausesFromModel(modelValue, nil, tableMetadata)
	if err != nil {
		return 0, err
	}

	if len(joinClauses) > 0 {

		// whereFields = append(whereFields, squirrel.Expr(lookup.MatchDBColumn+" IN ("+sql+")", args...))
		existsStr := ""
		for _, joinClause := range joinClauses {
			subTableName := ""
			if err != nil {
				return 0, err
			}
			joinStr := strings.Split(joinClause, " ")
			if len(joinStr) > 0 {
				subTableName = joinStr[0]
			}
		}
		for _, whereClause := range whereClauses {
			query, _, err := whereClause.ToSql()
			if err != nil {
				return _, err
			}
			eqStr := strings.Split('.')
			tableAlias := ""
			if len(eqStr) > 0 {
				tableAlias = eqStr[0]
			}
			// check
			if tableAlias != tableMetadata {
				// we got a join
			}
		}*/


		// If we have join clauses, we'll have to fetch the ids and then delete them.
		fetchResults, err := porm.FilterModel(model)
		if err != nil {
			return 0, err
		}

		deleteKeys := []string{}

		for _, fetchResult := range fetchResults {
			resultValue := reflect.ValueOf(fetchResult)
			resultID := resultValue.FieldByName(tableMetadata.getPrimaryKeyFieldName())
			deleteKeys = append(deleteKeys, resultID.String())
		}

		whereClauses = []squirrel.Sqlizer{
			squirrel.Eq{
				tableMetadata.getPrimaryKeyColumnName(): deleteKeys,
			},
		}

		if multitenancyKeyColumnName != "" {
			whereClauses = append(whereClauses, squirrel.Eq{multitenancyKeyColumnName: porm.multitenancyValue})
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
		PlaceholderFormat(squirrel.Dollar).
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
