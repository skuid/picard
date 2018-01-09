package picard

import (
	"errors"
	"reflect"

	uuid "github.com/satori/go.uuid"
)

// SaveModel performs an upsert operation for the provided model.
func (p Picard) SaveModel(model interface{}) error {
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

	primaryKeyValue := getPrimaryKey(modelValue)

	picardTags := picardTagsFromType(modelValue.Type())
	columnNames := picardTags.DataColumnNames()
	tableName := picardTags.TableName()
	primaryKeyColumnName := picardTags.PrimaryKeyColumnName()
	multitenancyKeyColumnName := picardTags.MultitenancyKeyColumnName()

	if primaryKeyValue == uuid.Nil {
		// Empty UUID: the model needs to insert.
		if err := p.insertModel(modelValue, tableName, columnNames, primaryKeyColumnName); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Non-Empty UUID: the model needs to update.
		if err := p.updateModel(modelValue, tableName, columnNames, multitenancyKeyColumnName, primaryKeyColumnName); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (p Picard) updateModel(modelValue reflect.Value, tableName string, columnNames []string, multitenancyKeyColumnName string, primaryKeyColumnName string) error {
	primaryKeyValue := getPrimaryKey(modelValue)
	existingObject, err := p.getExistingObjectByID(tableName, multitenancyKeyColumnName, primaryKeyColumnName, primaryKeyValue)
	if err != nil {
		return err
	}
	change, err := p.processObject(modelValue, existingObject)
	if err != nil {
		return err
	}
	err = p.performUpdates([]DBChange{change}, tableName, columnNames, multitenancyKeyColumnName, primaryKeyColumnName)
	if err != nil {
		return err
	}
	return nil
}

func (p Picard) insertModel(modelValue reflect.Value, tableName string, columnNames []string, primaryKeyColumnName string) error {
	change, err := p.processObject(modelValue, nil)
	if err != nil {
		return err
	}
	if err := p.performInserts([]DBChange{change}, tableName, columnNames, primaryKeyColumnName); err != nil {
		return err
	}
	setPrimaryKeyFromInsertResult(modelValue, change)
	return nil
}

func getPrimaryKey(v reflect.Value) uuid.UUID {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		picardFieldTags := getStructTagsMap(field, "picard")

		_, isPrimaryKey := picardFieldTags["primary_key"]
		if isPrimaryKey {
			primaryKeyUUID := v.FieldByName(field.Name)
			// Ignoring error here because ID should always be uuid
			id, _ := uuid.FromString(primaryKeyUUID.Interface().(string))
			return id
		}

	}
	return uuid.Nil
}

func setPrimaryKeyFromInsertResult(v reflect.Value, change DBChange) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		picardFieldTags := getStructTagsMap(field, "picard")

		_, isPrimaryKey := picardFieldTags["primary_key"]
		column, hasColumn := picardFieldTags["column"]
		if isPrimaryKey && hasColumn {
			v.FieldByName(field.Name).Set(reflect.ValueOf(change.changes[column]))
		}

	}
}
