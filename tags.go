package picard

import (
	"errors"
	"reflect"
	"strings"
)

const picardTagKey = "picard"

type fieldMetadata struct {
	name              string
	isPrimaryKey      bool
	isMultitenancyKey bool
	isJSONB           bool
	isEncrypted       bool
	columnName        string
	audit             string
	fieldType         reflect.Type
}

func (fm fieldMetadata) includeInUpdate() bool {
	return !fm.isPrimaryKey && !fm.isMultitenancyKey && fm.audit != "created_at" && fm.audit != "created_by"
}

type tableMetadata struct {
	tableName            string
	primaryKeyField      string
	multitenancyKeyField string
	fields               map[string]fieldMetadata
	fieldOrder           []string
	lookups              []Lookup
	foreignKeys          []ForeignKey
	children             []Child
}

func (tm tableMetadata) getTableName() string {
	return tm.tableName
}

func (tm tableMetadata) getColumnNames() []string {
	columnNames := []string{}
	for _, field := range tm.getFields() {
		columnNames = append(columnNames, field.columnName)
	}
	return columnNames
}

func (tm tableMetadata) getColumnNamesWithoutPrimaryKey() []string {
	columnNames := []string{}
	for _, field := range tm.getFields() {
		if !field.isPrimaryKey {
			columnNames = append(columnNames, field.columnName)
		}
	}
	return columnNames
}

func (tm tableMetadata) getColumnNamesForUpdate() []string {
	columnNames := []string{}
	for _, field := range tm.getFields() {
		if !field.includeInUpdate() {
			continue
		}
		columnNames = append(columnNames, field.columnName)
	}
	return columnNames
}

func (tm tableMetadata) getEncryptedColumns() []string {
	columnNames := []string{}
	for _, field := range tm.getFields() {
		if field.isEncrypted {
			columnNames = append(columnNames, field.columnName)
		}
	}
	return columnNames
}

func (tm tableMetadata) getJSONBColumns() []string {
	columnNames := []string{}
	for _, field := range tm.getFields() {
		if field.isJSONB {
			columnNames = append(columnNames, field.columnName)
		}
	}
	return columnNames
}

func (tm tableMetadata) getPrimaryKeyMetadata() *fieldMetadata {
	metadata, ok := tm.fields[tm.primaryKeyField]
	if ok {
		return &metadata
	}
	return nil
}

func (tm tableMetadata) getMultitenancyKeyMetadata() *fieldMetadata {
	metadata, ok := tm.fields[tm.multitenancyKeyField]
	if ok {
		return &metadata
	}
	return nil
}

func (tm tableMetadata) getPrimaryKeyColumnName() string {
	metadata := tm.getPrimaryKeyMetadata()
	if metadata != nil {
		return metadata.columnName
	}
	return ""
}

func (tm tableMetadata) getPrimaryKeyFieldName() string {
	metadata := tm.getPrimaryKeyMetadata()
	if metadata != nil {
		return metadata.name
	}
	return ""
}

func (tm tableMetadata) getMultitenancyKeyColumnName() string {
	metadata := tm.getMultitenancyKeyMetadata()
	if metadata != nil {
		return metadata.columnName
	}
	return ""
}

// Returns the fields in the order they appear in the struct
func (tm tableMetadata) getFields() []fieldMetadata {
	fields := []fieldMetadata{}
	for _, key := range tm.fieldOrder {
		fields = append(fields, tm.fields[key])
	}
	return fields
}

func getTableMetadata(data interface{}) (*tableMetadata, error) {
	// Verify that we've been passed valid input
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Slice {
		return nil, errors.New("Can only upsert slices")
	}

	tableMetadata := tableMetadataFromType(t.Elem())

	if tableMetadata.tableName == "" {
		return nil, errors.New("No table name specified in struct metadata")
	}
	return tableMetadata, nil
}

func tableMetadataFromType(t reflect.Type) *tableMetadata {
	var metadata Metadata
	tableMetadata := tableMetadata{
		fields: map[string]fieldMetadata{},
	}
	children := []Child{}
	lookups := []Lookup{}
	foreignKeys := []ForeignKey{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		kind := field.Type.Kind()

		tagsMap := getStructTagsMap(field, picardTagKey)
		_, hasTableName := tagsMap["tablename"]
		_, isPrimaryKey := tagsMap["primary_key"]
		_, isMultitenancyKey := tagsMap["multitenancy_key"]
		columnName, hasColumnName := tagsMap["column"]
		_, isLookup := tagsMap["lookup"]
		_, isChild := tagsMap["child"]
		_, isRequired := tagsMap["required"]
		_, isForeignKey := tagsMap["foreign_key"]
		_, isEncrypted := tagsMap["encrypted"]
		_, isJSONB := tagsMap["jsonb"]
		auditType, _ := tagsMap["audit"]

		if field.Type == reflect.TypeOf(metadata) && hasTableName {
			tableMetadata.tableName = tagsMap["tablename"]
		}

		if hasColumnName {

			tableMetadata.fields[field.Name] = fieldMetadata{
				name:              field.Name,
				isEncrypted:       isEncrypted,
				isJSONB:           isJSONB,
				isMultitenancyKey: isMultitenancyKey,
				isPrimaryKey:      isPrimaryKey,
				columnName:        columnName,
				audit:             auditType,
				fieldType:         field.Type,
			}

			tableMetadata.fieldOrder = append(tableMetadata.fieldOrder, field.Name)

			if isMultitenancyKey {
				tableMetadata.multitenancyKeyField = field.Name
			}
			if isPrimaryKey {
				tableMetadata.primaryKeyField = field.Name
			}
		}

		if isChild && (kind == reflect.Slice || kind == reflect.Map) {
			keyMappings := []string{}
			valueMappingMap := map[string]string{}
			keyMappingString := tagsMap["key_mappings"]
			valueMappingString := tagsMap["value_mappings"]

			if keyMappingString != "" {
				keyMappings = strings.Split(keyMappingString, "&")
			}

			if valueMappingString != "" {
				valueMappingArray := strings.Split(valueMappingString, "&")
				for _, valueMap := range valueMappingArray {
					valueMapSplit := strings.Split(valueMap, "->")
					valueMappingMap[valueMapSplit[0]] = valueMapSplit[1]
				}
			}

			children = append(children, Child{
				FieldName:     field.Name,
				FieldType:     field.Type,
				FieldKind:     kind,
				ForeignKey:    tagsMap["foreign_key"],
				KeyMappings:   keyMappings,
				ValueMappings: valueMappingMap,
			})
		}

		if isLookup && !isForeignKey {
			lookups = append(lookups, Lookup{
				MatchDBColumn:       tagsMap["column"],
				MatchObjectProperty: field.Name,
				Query:               true,
			})
		}

		if isForeignKey {

			relatedField, hasRelatedField := t.FieldByName(tagsMap["related"])

			if hasRelatedField {
				tableMetadata := tableMetadataFromType(relatedField.Type)
				foreignKeys = append(foreignKeys, ForeignKey{
					TableMetadata:    tableMetadata,
					FieldName:        field.Name,
					KeyColumn:        tagsMap["column"],
					RelatedFieldName: relatedField.Name,
					Required:         isRequired,
					NeedsLookup:      isLookup,
					KeyMapField:      tagsMap["key_map"],
				})
			}
		}

		tableMetadata.children = children
		tableMetadata.lookups = lookups
		tableMetadata.foreignKeys = foreignKeys
	}

	return &tableMetadata
}

func getStructTagsMap(field reflect.StructField, tagType string) map[string]string {
	tagValue := field.Tag.Get(tagType)
	if tagValue == "" {
		return nil
	}

	tags := strings.Split(tagValue, ",")
	tagsMap := map[string]string{}

	for _, v := range tags {
		tagSplit := strings.Split(v, "=")
		tagKey := tagSplit[0]
		tagValue := ""
		if (len(tagSplit)) == 2 {
			tagValue = tagSplit[1]
		}
		tagsMap[tagKey] = tagValue
	}

	return tagsMap
}
