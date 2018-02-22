package picard

import (
	"reflect"
	"strings"
)

const picardTagKey = "picard"

type picardTags struct {
	tableName             string
	primaryKeyColumn      string
	primaryKeyFieldName   string
	multitenancyKeyColumn string
	dataColumns           []string
	encryptedColumns      []string
	jsonbColumns          []string
	lookups               []Lookup
	foreignKeys           []ForeignKey
	children              []Child
	fieldToColumnMap      map[string]string
}

func (pt picardTags) TableName() string {
	return pt.tableName
}
func (pt picardTags) PrimaryKeyColumnName() string {
	return pt.primaryKeyColumn
}
func (pt picardTags) PrimaryKeyFieldName() string {
	return pt.primaryKeyFieldName
}
func (pt picardTags) MultitenancyKeyColumnName() string {
	return pt.multitenancyKeyColumn
}
func (pt picardTags) DataColumnNames() []string {
	return pt.dataColumns
}
func (pt picardTags) EncryptedColumns() []string {
	return pt.encryptedColumns
}
func (pt picardTags) JSONBColumns() []string {
	return pt.jsonbColumns
}
func (pt picardTags) Lookups() []Lookup {
	return pt.lookups
}
func (pt picardTags) ForeignKeys() []ForeignKey {
	return pt.foreignKeys
}
func (pt picardTags) Children() []Child {
	return pt.children
}
func (pt picardTags) ColumnNames() []string {
	columnNames := pt.dataColumns
	if pt.primaryKeyColumn != "" {
		columnNames = append(columnNames, pt.primaryKeyColumn)
	}
	return columnNames
}

func (pt picardTags) getColumnFromFieldName(fieldName string) string {

	var columnName string
	columnName, hasColumn := pt.fieldToColumnMap[fieldName]
	if hasColumn {
		return columnName
	}
	return ""
}

func addColumn(fieldToColumnMap map[string]string, dataColumns *[]string, columnName string, fieldName string) {
	*dataColumns = append(*dataColumns, columnName)
	fieldToColumnMap[fieldName] = columnName
}

func picardTagsFromType(t reflect.Type) picardTags {
	var metadata Metadata
	var (
		tableName             string
		primaryKeyColumn      string
		primaryKeyFieldName   string
		multitenancyKeyColumn string
		dataColumns           []string
		encryptedColumns      []string
		jsonbColumns          []string
		lookups               []Lookup
		foreignKeys           []ForeignKey
		children              []Child
		fieldToColumnMap      map[string]string
	)

	fieldToColumnMap = map[string]string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		kind := field.Type.Kind()

		tagsMap := getStructTagsMap(field, picardTagKey)
		_, hasTableName := tagsMap["tablename"]
		_, isPK := tagsMap["primary_key"]
		_, isMultitenancyKey := tagsMap["multitenancy_key"]
		columnName, hasColumnName := tagsMap["column"]
		_, isLookup := tagsMap["lookup"]
		_, isChild := tagsMap["child"]
		_, isRequired := tagsMap["required"]
		_, isForeignKey := tagsMap["foreign_key"]
		_, isEncrypted := tagsMap["encrypted"]
		_, isJSONB := tagsMap["jsonb"]

		if field.Type == reflect.TypeOf(metadata) && hasTableName {
			tableName = tagsMap["tablename"]
		}

		if hasColumnName {
			if isMultitenancyKey {
				multitenancyKeyColumn = columnName
			}
			if isEncrypted {
				encryptedColumns = append(encryptedColumns, columnName)
			}
			if isJSONB {
				jsonbColumns = append(jsonbColumns, columnName)
			}
			if isPK {
				primaryKeyColumn = columnName
				primaryKeyFieldName = field.Name
			} else {
				addColumn(fieldToColumnMap, &dataColumns, columnName, field.Name)
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
				tags := picardTagsFromType(relatedField.Type)
				foreignKeys = append(foreignKeys, ForeignKey{
					ObjectInfo:       tags,
					FieldName:        field.Name,
					KeyColumn:        tagsMap["column"],
					RelatedFieldName: relatedField.Name,
					Required:         isRequired,
					NeedsLookup:      isLookup,
				})
			}
		}
	}

	return picardTags{
		tableName:             tableName,
		primaryKeyColumn:      primaryKeyColumn,
		primaryKeyFieldName:   primaryKeyFieldName,
		multitenancyKeyColumn: multitenancyKeyColumn,
		dataColumns:           dataColumns,
		encryptedColumns:      encryptedColumns,
		lookups:               lookups,
		foreignKeys:           foreignKeys,
		children:              children,
		fieldToColumnMap:      fieldToColumnMap,
		jsonbColumns:          jsonbColumns,
	}
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
