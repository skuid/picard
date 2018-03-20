package picard

import (
	"errors"
	"reflect"
)

// SaveModel performs an upsert operation for the provided model.
func (p PersistenceORM) SaveModel(model interface{}) error {
	return p.persistModel(model, false)
}

// CreateModel performs an insert operation for the provided model.
func (p PersistenceORM) CreateModel(model interface{}) error {
	return p.persistModel(model, true)
}

// persistModel performs an upsert operation for the provided model.
func (p PersistenceORM) persistModel(model interface{}, alwaysInsert bool) error {
	// This makes modelValue a reflect.Value of model whether model is a pointer or not.
	modelValue := reflect.Indirect(reflect.ValueOf(model))
	if modelValue.Kind() != reflect.Struct {
		return errors.New("Models must be structs")
	}
	tx, err := GetConnection().Begin()
	if err != nil {
		return err
	}

	p.transaction = tx

	tableMetadata := tableMetadataFromType(modelValue.Type())
	primaryKeyValue := modelValue.FieldByName(tableMetadata.primaryKeyField).Interface()

	if primaryKeyValue == nil || primaryKeyValue == "" || alwaysInsert {
		if err := p.insertModel(modelValue, tableMetadata, primaryKeyValue); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Non-Empty UUID: the model needs to update.
		if err := p.updateModel(modelValue, tableMetadata, primaryKeyValue); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (p PersistenceORM) updateModel(modelValue reflect.Value, tableMetadata *tableMetadata, primaryKeyValue interface{}) error {
	existingObject, err := p.getExistingObjectByID(tableMetadata, primaryKeyValue)
	if err != nil {
		return err
	}
	if existingObject == nil {
		return ModelNotFoundError
	}
	change, err := p.processObject(modelValue, existingObject, nil, tableMetadata)
	if err != nil {
		return err
	}
	return p.performUpdates([]DBChange{change}, tableMetadata)
}

func (p PersistenceORM) insertModel(modelValue reflect.Value, tableMetadata *tableMetadata, primaryKeyValue interface{}) error {
	change, err := p.processObject(modelValue, nil, nil, tableMetadata)
	if err != nil {
		return err
	}
	insertsHavePrimaryKey := !(primaryKeyValue == nil || primaryKeyValue == "")
	if err := p.performInserts([]DBChange{change}, insertsHavePrimaryKey, tableMetadata); err != nil {
		return err
	}
	setPrimaryKeyFromInsertResult(modelValue, change, tableMetadata)
	return nil
}

func setPrimaryKeyFromInsertResult(v reflect.Value, change DBChange, tableMetadata *tableMetadata) {
	fieldName := tableMetadata.getPrimaryKeyFieldName()
	columnName := tableMetadata.getPrimaryKeyColumnName()
	if fieldName != "" {
		v.FieldByName(fieldName).Set(reflect.ValueOf(change.changes[columnName]))
	}
}
