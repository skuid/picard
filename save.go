package picard

import (
	"errors"
	"reflect"

	"github.com/skuid/picard/dbchange"
	"github.com/skuid/picard/tags"
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

	if p.transaction == nil {
		tx, err := GetConnection().Begin()
		if err != nil {
			return err
		}
		p.transaction = tx
		defer p.Commit()
	}

	tableMetadata := tags.TableMetadataFromType(modelValue.Type())
	primaryKeyValue := modelValue.FieldByName(tableMetadata.GetPrimaryKeyFieldName()).Interface()

	if primaryKeyValue == nil || primaryKeyValue == "" || alwaysInsert {
		if err := p.insertModel(modelValue, tableMetadata, primaryKeyValue); err != nil {
			p.Rollback()
			return err
		}
	} else {
		// Non-Empty UUID: the model needs to update.
		if err := p.updateModel(modelValue, tableMetadata, primaryKeyValue); err != nil {
			p.Rollback()
			return err
		}
	}

	return nil
}

func (p PersistenceORM) updateModel(modelValue reflect.Value, tableMetadata *tags.TableMetadata, primaryKeyValue interface{}) error {
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
	return p.performUpdates([]dbchange.Change{change}, tableMetadata)
}

func (p PersistenceORM) insertModel(modelValue reflect.Value, tableMetadata *tags.TableMetadata, primaryKeyValue interface{}) error {
	change, err := p.processObject(modelValue, nil, nil, tableMetadata)
	if err != nil {
		return err
	}
	insertsHavePrimaryKey := !(primaryKeyValue == nil || primaryKeyValue == "")
	if err := p.performInserts([]dbchange.Change{change}, insertsHavePrimaryKey, tableMetadata); err != nil {
		return err
	}
	setPrimaryKeyFromInsertResult(modelValue, change, tableMetadata)
	return nil
}

func setPrimaryKeyFromInsertResult(v reflect.Value, change dbchange.Change, tableMetadata *tags.TableMetadata) {
	fieldName := tableMetadata.GetPrimaryKeyFieldName()
	columnName := tableMetadata.GetPrimaryKeyColumnName()
	if fieldName != "" {
		v.FieldByName(fieldName).Set(reflect.ValueOf(change.Changes[columnName]))
	}
}
