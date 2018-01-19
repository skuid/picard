package picard

import (
	"reflect"
	"strings"
)

const picardTagKey = "picard"

type picardTags struct {
	tableName             string
	primaryKeyColumn      string
	multitenancyKeyColumn string
	dataColumns           []string
	encryptedColumns      []string
	lookups               []Lookup
	children              []Child
	fieldToColumnMap      map[string]string
}

func (pt picardTags) TableName() string {
	return pt.tableName
}
func (pt picardTags) PrimaryKeyColumnName() string {
	return pt.primaryKeyColumn
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
func (pt picardTags) Lookups() []Lookup {
	return pt.lookups
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
		multitenancyKeyColumn string
		dataColumns           []string
		encryptedColumns      []string
		lookups               []Lookup
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
		_, isEncrypted := tagsMap["encrypted"]

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
			if isPK {
				primaryKeyColumn = columnName
			} else {
				addColumn(fieldToColumnMap, &dataColumns, columnName, field.Name)
			}
		}

		if isChild && kind == reflect.Slice {
			children = append(children, Child{
				FieldName:  field.Name,
				FieldType:  field.Type,
				ForeignKey: tagsMap["foreign_key"],
			})
		}

		if isLookup {
			lookups = append(lookups, Lookup{
				MatchDBColumn:       tagsMap["column"],
				MatchObjectProperty: field.Name,
				Query:               true,
			})
		}
	}

	return picardTags{
		tableName:             tableName,
		primaryKeyColumn:      primaryKeyColumn,
		multitenancyKeyColumn: multitenancyKeyColumn,
		dataColumns:           dataColumns,
		encryptedColumns:      encryptedColumns,
		lookups:               lookups,
		children:              children,
		fieldToColumnMap:      fieldToColumnMap,
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
