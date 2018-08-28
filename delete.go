package picard

import (
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

	whereClauses, _, err := porm.generateWhereClausesFromModel(modelValue, nil, tableMetadata, false)
	if err != nil {
		return 0, err
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
