package picard

import (
	"reflect"

	"github.com/Masterminds/squirrel"
	"github.com/skuid/picard/tags"
)

// DeleteModel will delete models that match the provided struct, ignoring zero values.
// Returns the number of rows affected or an error.
func (porm PersistenceORM) DeleteModel(model interface{}) (int64, error) {
	modelValue, err := getStructValue(model)
	if err != nil {
		return 0, err
	}

	tableMetadata := tags.TableMetadataFromType(modelValue.Type())
	tableName := tableMetadata.GetTableName()
	multitenancyKeyColumnName := tableMetadata.GetMultitenancyKeyColumnName()

	whereClauses, joinClauses, err := porm.generateWhereClausesFromModel(modelValue, nil, tableMetadata)
	if err != nil {
		return 0, err
	}

	if len(joinClauses) > 0 {
		// If we have join clauses, we'll have to fetch the ids and then delete them.
		fetchResults, err := porm.FilterModel(model)
		if err != nil {
			return 0, err
		}

		deleteKeys := []string{}

		for _, fetchResult := range fetchResults {
			resultValue := reflect.ValueOf(fetchResult)
			resultID := resultValue.FieldByName(tableMetadata.GetPrimaryKeyFieldName())
			deleteKeys = append(deleteKeys, resultID.String())
		}

		whereClauses = []squirrel.Sqlizer{
			squirrel.Eq{
				tableMetadata.GetPrimaryKeyColumnName(): deleteKeys,
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
