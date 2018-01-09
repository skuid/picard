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
	lookups               []Lookup
	children              []Child
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

func picardTagsFromType(t reflect.Type) picardTags {
	var structMetadata StructMetadata
	var (
		tableName             string
		primaryKeyColumn      string
		multitenancyKeyColumn string
		dataColumns           []string
		lookups               []Lookup
		children              []Child
	)

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

		switch {

		case field.Type == reflect.TypeOf(structMetadata) && hasTableName:
			tableName = tagsMap["tablename"]

		case isPK && hasColumnName:
			primaryKeyColumn = columnName

		case isMultitenancyKey && hasColumnName:
			multitenancyKeyColumn = columnName
			dataColumns = append(dataColumns, columnName)

		case isChild && kind == reflect.Slice:
			children = append(children, Child{
				FieldName:  field.Name,
				FieldType:  field.Type,
				ForeignKey: tagsMap["foreign_key"],
			})

		case isLookup:
			lookups = append(lookups, Lookup{
				MatchDBColumn:       tagsMap["column"],
				MatchObjectProperty: field.Name,
				Query:               true,
			})
			if hasColumnName {
				dataColumns = append(dataColumns, columnName)
			}

		case hasColumnName && !isPK && !isChild:
			dataColumns = append(dataColumns, columnName)

		default:
			// No known picard tags on this field
		}
	}

	return picardTags{
		tableName:             tableName,
		primaryKeyColumn:      primaryKeyColumn,
		multitenancyKeyColumn: multitenancyKeyColumn,
		dataColumns:           dataColumns,
		lookups:               lookups,
		children:              children,
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
