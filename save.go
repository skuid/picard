package picard

import (
	"errors"
	"reflect"

	uuid "github.com/satori/go.uuid"
)

// SaveModel performs an upsert operation for the provided model.
func (p PersistenceORM) SaveModel(model interface{}) error {
	return p.persistModel(model, false)
}

// CreateModel performs an insert operation for the provided model.
func (p PersistenceORM) CreateModel(model interface{}) error {
	return p.persistModel(model, true)
}

func getColumnsToPersist(modelValue reflect.Value, picardTags picardTags) []string {
	columnNames := picardTags.DataColumnNames()
	primaryKeyColumnName := picardTags.PrimaryKeyColumnName()
	multitenancyKeyColumnName := picardTags.MultitenancyKeyColumnName()
	// Get Defined Fields if they exist
	modelMetadata := getMetadataFromPicardStruct(modelValue)

	// Check for nil here instead of the length of the slice.
	// The decode method in picard sets defined fields to an empty slice if it has been run.
	if modelMetadata.DefinedFields == nil {
		return columnNames
	}

	updateColumns := []string{}
	// Loop over columnNames
	for _, columnName := range columnNames {
		if columnName == primaryKeyColumnName || columnName == multitenancyKeyColumnName {
			updateColumns = append(updateColumns, columnName)
			continue
		}
		for _, fieldName := range modelMetadata.DefinedFields {

			definedColumnName := picardTags.getColumnFromFieldName(fieldName)
			if definedColumnName != "" && definedColumnName == columnName {
				updateColumns = append(updateColumns, columnName)
				break
			}
		}
	}
	return updateColumns
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

	primaryKeyValue := getPrimaryKey(modelValue)

	picardTags := picardTagsFromType(modelValue.Type())
	tableName := picardTags.TableName()
	primaryKeyColumnName := picardTags.PrimaryKeyColumnName()
	multitenancyKeyColumnName := picardTags.MultitenancyKeyColumnName()

	persistColumns := getColumnsToPersist(modelValue, picardTags)

	if primaryKeyValue == uuid.Nil || alwaysInsert {
		if primaryKeyValue != uuid.Nil && !stringSliceContainsKey(persistColumns, primaryKeyColumnName) {
			persistColumns = append(persistColumns, primaryKeyColumnName)
		}

		if err := p.insertModel(modelValue, tableName, persistColumns, primaryKeyColumnName); err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// Non-Empty UUID: the model needs to update.
		if err := p.updateModel(modelValue, tableName, persistColumns, multitenancyKeyColumnName, primaryKeyColumnName); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (p PersistenceORM) updateModel(modelValue reflect.Value, tableName string, columnNames []string, multitenancyKeyColumnName string, primaryKeyColumnName string) error {
	primaryKeyValue := getPrimaryKey(modelValue)
	existingObject, err := p.getExistingObjectByID(tableName, multitenancyKeyColumnName, primaryKeyColumnName, primaryKeyValue)
	if err != nil {
		return err
	}
	if existingObject == nil {
		return ModelNotFoundError
	}
	change, err := p.processObject(modelValue, existingObject, nil, false)
	if err != nil {
		return err
	}
	return p.performUpdates([]DBChange{change}, tableName, columnNames, multitenancyKeyColumnName, primaryKeyColumnName)
}

func (p PersistenceORM) insertModel(modelValue reflect.Value, tableName string, columnNames []string, primaryKeyColumnName string) error {
	change, err := p.processObject(modelValue, nil, nil, true)
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
