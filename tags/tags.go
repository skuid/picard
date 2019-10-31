/*
Package tags generates table metadata by reading picard struct tag annotations
*/
package tags

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/skuid/picard/metadata"
	qp "github.com/skuid/picard/queryparts"
)

const picardTagKey = "picard"

/* Association represents a data model relationship in the form of hasOne, hasMany, belongsTo between parent and child structs.

Including Associations in FilterRequests will eager load the model relationship results in a single query with JOINs.

Name refers to the name of the struct field that will hold the filter results for the relationship.
For belongsTo relationships, this is the `related` tag value on a `foreign_key`` field on the struct

Example:

	type ChildModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=child"`
		ID			string				`picard:"primary_key,column=id"`
		ParentID 	string				`picard:"foreign_key,required,related=Parent,column=parent_id
		Parent		ParentModel
	}

	p.FilterModel(picard.FilterRequest{
		FilterModel: ChildModel{},
		Associations: []tags.Association{
			{
				Name: "Parent",
			},
		},
	})

For hasOne or hasMany relationships, this is the field with the `child` tag

Example:

	type ParentModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=parent"`
		ID			string				`picard:"primary_key,column=id"`
		Children	[]ChildModel		`picard:"child,foreign_key=ParentID"`
	}

	p.FilterModel(picard.FilterRequest{
		FilterModel: ParentModel{},
		Associations: []tags.Association{
			{
				Name: "Children",
			},
		},
	})


Each association may have nested relationships, like so:

	type ParentModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=parent"`
		ID			string				`picard:"primary_key,column=id"`
		Children	[]ChildModel		`picard:"child,foreign_key=ParentID"`
	}

	type ChildModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=child"`
		ID			string				`picard:"primary_key,column=id"`
		Children	[]GrandChildModel	`picard:"child,foreign_key=ParentID"`
		ParentID 	string				`picard:"foreign_key,required,related=Parent,column=parent_id"``
		Parent		ParentModel
	}

	type GrandChild struct {
		Metadata	metadata.Metadata	`picard:"tablename=grandchild"`
		ID			string				`picard:"primary_key,column=id"`
		ParentID 	string				`picard:"foreign_key,required,related=Parent,column=parent_id
		Parent		ChildModel
	}

	p.FilterModel(picard.FilterRequest{
		FilterModel: ParentModel{},
		Associations: []tags.Association{
		{
			Name: "Children",
			Associations: []tags.Association{
				{
					Name: "Children",
				},
			},
		},
	})


OrderBy lets you define the ordering of filter results by adding an ORDER BY clause with an OrderByRequest

Example:

	type ParentModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=parent"`
		ID			string				`picard:"primary_key,column=id"`
		Children	[]ChildModel		`picard:"child,foreign_key=ParentID"`
	}

	type ChildModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=child"`
		ID			string				`picard:"primary_key,column=id"`
		Name		string				`picard:"column=name"`
	}
	p.FilterModel(picard.FilterRequest{
		FilterModel: ParentModel{},
		Associations: []tags.Association{
				{
					Name: "Children",
					OrderBy: []qp.OrderByRequest{
					{
						Field:      "Name",
						Descending: true,
					},
				},
			},
	})

	// SELECT ... FROM parent AS t0 LEFT JOIN child AS t1 ON ...ORDER BY t1.name DESC

SelectFields lets you define the exact columns to query for. Without `SelectFields`, all the columns defined in the table will be included in the query.

From the example above for OrderBy:

	p.FilterModel(picard.FilterRequest{
		FilterModel: ParentModel{},
		Associations: []tags.Association{
				{
					Name: "Children",
					SelectFields: []string{
						"Name",
					}
				},
			},
		}
	})

	// SELECT ... t1.name FROM parent AS t0 LEFT JOIN child AS t1 ON ...

FieldFilters generates a `WHERE` clause grouping with either an `OR` grouping via `tags.OrFilterGroup` or an `AND` grouping via `tags.AndFilterGroup`. The `tags.FieldFilter`

	type ParentModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=parent"`
		ID			string				`picard:"primary_key,column=id"`
		Children	[]ChildModel		`picard:"child,foreign_key=ParentID"`
	}

	type ChildModel struct {
		Metadata	metadata.Metadata	`picard:"tablename=child"`
		ID			string				`picard:"primary_key,column=id"`
		FieldA		string				`picard:"column=field_a"`
		FieldB		string				`picard:"column=field_b"`
	}


import "github.com/skuid/picard/tags"

	p.FilterModel(picard.FilterRequest{
			FilterModel: ParentModel{},
			Associations: []tags.Association{
				{
					Name: "Children",
					FieldFilters: tags.OrFilterGroup{
						tags.FieldFilter{
							FieldName:   "FieldA",
							FilterValue: "foo",
						},
						tags.FieldFilter{
							FieldName:   "FieldB",
							FilterValue: "bar",
						},
					},
				}
			},
		}
	})

	// SELECT ... WHERE (t1.field_a = 'foo' OR t1.field_b = 'bar')

	p.FilterModel(picard.FilterRequest{
			FilterModel: ParentModel{},
			Associations: []tags.Association{
				{
					Name: "Children",
					FieldFilters: tags.AndFilterGroup{
						tags.FieldFilter{
							FieldName:   "FieldA",
							FilterValue: "foo",
						},
						tags.FieldFilter{
							FieldName:   "FieldB",
							FilterValue: "bar",
						},
					},
				}
			},
		}
	})

	// SELECT ... WHERE (t1.field_a = 'foo' AND t1.field_b = 'bar')
*/
type Association struct {
	Name         string
	Associations []Association
	OrderBy      []qp.OrderByRequest
	SelectFields []string
	FieldFilters Filterable
}

/* FieldFilter defines an arbitrary filter on a FilterRequest


Specify the fields that should be added in a AndFilterGroup or a OrFilterGroup WHERE clause grouping.

Example:

	import "github.com/skuid/picard/tags"

	tags.FieldFilter{
		FieldName:   "FieldB",
		FilterValue: "bar",
	},

SQL translation in WHERE clause grouping:

	t0.field_B = "bar"
*/
type FieldFilter struct {
	FieldName   string
	FilterValue interface{}
}

// Apply applies the filter
func (ff FieldFilter) Apply(table *qp.Table, metadata *TableMetadata) squirrel.Sqlizer {
	// Return early if no fieldname was provided in our filter
	if ff.FieldName == "" {
		return squirrel.Eq{}
	}
	fieldMetadata := metadata.GetField(ff.FieldName)
	columnName := fieldMetadata.GetColumnName()
	return squirrel.Eq{fmt.Sprintf(qp.AliasedField, table.Alias, columnName): ff.FilterValue}
}

// OrFilterGroup applies a group of filters using ors
type OrFilterGroup []Filterable

// Apply applies the filter
func (ofg OrFilterGroup) Apply(table *qp.Table, metadata *TableMetadata) squirrel.Sqlizer {
	ors := squirrel.Or{}
	for _, filter := range ofg {
		ors = append(ors, filter.Apply(table, metadata))
	}
	return ors
}

// AndFilterGroup applies a group of filters using ands
type AndFilterGroup []Filterable

// Apply applies the filter
func (afg AndFilterGroup) Apply(table *qp.Table, metadata *TableMetadata) squirrel.Sqlizer {
	ands := squirrel.And{}
	for _, filter := range afg {
		ands = append(ands, filter.Apply(table, metadata))
	}
	return ands
}

// Filterable interface allows filters to be specified in Filter Requests
type Filterable interface {
	Apply(*qp.Table, *TableMetadata) squirrel.Sqlizer
}

// Lookup structure
type Lookup struct {
	TableName           string
	MatchDBColumn       string
	MatchObjectProperty string
	JoinKey             string
	Value               interface{}
	SubQuery            []Lookup
	SubQueryForeignKey  string
	SubQueryMetadata    *TableMetadata
}

// Child structure
type Child struct {
	FieldName        string
	FieldType        reflect.Type
	FieldKind        reflect.Kind
	ForeignKey       string
	KeyMapping       string
	ValueMappings    map[string]string
	GroupingCriteria map[string]string
	DeleteOrphans    bool
}

// ForeignKey structure
type ForeignKey struct {
	TableMetadata    *TableMetadata
	FieldName        string
	KeyColumn        string
	RelatedFieldName string
	Required         bool
	NeedsLookup      bool
	LookupResults    map[string]interface{}
	LookupsUsed      []Lookup
	KeyMapField      string
}

// FieldMetadata structure
type FieldMetadata struct {
	name              string
	isPrimaryKey      bool
	isMultitenancyKey bool
	isJSONB           bool
	isEncrypted       bool
	isFK              bool
	relatedField      reflect.StructField
	columnName        string
	audit             string
	fieldType         reflect.Type
}

// IncludeInUpdate function
func (fm FieldMetadata) IncludeInUpdate() bool {
	return !fm.isPrimaryKey && !fm.isMultitenancyKey && fm.audit != "created_at" && fm.audit != "created_by"
}

// GetAudit function
func (fm FieldMetadata) GetAudit() string {
	return fm.audit
}

// IsJSONB function
func (fm FieldMetadata) IsJSONB() bool {
	return fm.isJSONB
}

// IsPrimaryKey function
func (fm FieldMetadata) IsPrimaryKey() bool {
	return fm.isPrimaryKey
}

// IsFK function
func (fm FieldMetadata) IsFK() bool {
	return fm.isFK
}

// IsMultitenancyKey function
func (fm FieldMetadata) IsMultitenancyKey() bool {
	return fm.isMultitenancyKey
}

// GetName function
func (fm FieldMetadata) GetName() string {
	return fm.name
}

// GetColumnName function
func (fm FieldMetadata) GetColumnName() string {
	return fm.columnName
}

// GetFieldType function
func (fm FieldMetadata) GetFieldType() reflect.Type {
	return fm.fieldType
}

// GetRelatedType function
func (fm FieldMetadata) GetRelatedType() reflect.Type {
	return fm.relatedField.Type
}

// GetRelatedName function
func (fm FieldMetadata) GetRelatedName() string {
	return fm.relatedField.Name
}

// IsEncrypted function
func (fm FieldMetadata) IsEncrypted() bool {
	return fm.isEncrypted
}

// TableMetadata structure
type TableMetadata struct {
	tableName            string
	primaryKeyField      string
	multitenancyKeyField string
	fields               map[string]FieldMetadata
	fieldOrder           []string
	lookups              []Lookup
	foreignKeys          []ForeignKey
	children             []Child
}

// GetChildren function
func (tm TableMetadata) GetChildren() []Child {
	return tm.children
}

// GetLookups function
func (tm TableMetadata) GetLookups() []Lookup {
	return tm.lookups
}

// GetForeignKeys function
func (tm TableMetadata) GetForeignKeys() []ForeignKey {
	// Clone the foreign keys, don't just return a reference
	// We wouldn't want code elsewhere to mutate it
	keys := []ForeignKey{}
	for _, key := range tm.foreignKeys {
		keys = append(keys, key)
	}
	return keys
}

// GetTableName gets the name of the table
func (tm TableMetadata) GetTableName() string {
	return tm.tableName
}

// GetColumnNames gets the column names
func (tm TableMetadata) GetColumnNames() []string {
	columnNames := []string{}
	for _, field := range tm.GetFields() {
		columnNames = append(columnNames, field.columnName)
	}
	return columnNames
}

// GetColumnNamesWithoutPrimaryKey gets the columm names, but excludes the primary key
func (tm TableMetadata) GetColumnNamesWithoutPrimaryKey() []string {
	columnNames := []string{}
	for _, field := range tm.GetFields() {
		if !field.isPrimaryKey {
			columnNames = append(columnNames, field.columnName)
		}
	}
	return columnNames
}

// GetColumnNamesForUpdate gets the columm names, but excludes certain fields
func (tm TableMetadata) GetColumnNamesForUpdate() []string {
	columnNames := []string{}
	for _, field := range tm.GetFields() {
		if !field.IncludeInUpdate() {
			continue
		}
		columnNames = append(columnNames, field.columnName)
	}
	return columnNames
}

// GetChildField function
func (tm TableMetadata) GetChildField(childName string) *Child {
	for _, child := range tm.children {
		if child.FieldName == childName {
			return &child
		}
	}
	return nil
}

// GetChildFieldFromForeignKey function
func (tm TableMetadata) GetChildFieldFromForeignKey(foreignKeyName string, foreignKeyType reflect.Type) *Child {
	for _, child := range tm.children {
		if child.ForeignKey == foreignKeyName && child.FieldType == foreignKeyType {
			return &child
		}
	}
	return nil
}

// GetForeignKeyField function
func (tm TableMetadata) GetForeignKeyField(foreignKeyName string) *ForeignKey {
	for _, foreignKey := range tm.foreignKeys {
		if foreignKey.FieldName == foreignKeyName {
			return &foreignKey
		}
	}
	return nil
}

// GetForeignKeyFieldFromRelation function
func (tm TableMetadata) GetForeignKeyFieldFromRelation(relationName string) *ForeignKey {
	for _, foreignKey := range tm.foreignKeys {
		if foreignKey.RelatedFieldName == relationName {
			return &foreignKey
		}
	}
	return nil
}

// GetEncryptedColumns function
func (tm TableMetadata) GetEncryptedColumns() []string {
	columnNames := []string{}
	for _, field := range tm.GetFields() {
		if field.isEncrypted {
			columnNames = append(columnNames, field.columnName)
		}
	}
	return columnNames
}

// GetJSONBColumns function
func (tm TableMetadata) GetJSONBColumns() []string {
	columnNames := []string{}
	for _, field := range tm.GetFields() {
		if field.isJSONB {
			columnNames = append(columnNames, field.columnName)
		}
	}
	return columnNames
}

// GetPrimaryKeyMetadata function
func (tm TableMetadata) GetPrimaryKeyMetadata() *FieldMetadata {
	metadata, ok := tm.fields[tm.primaryKeyField]
	if ok {
		return &metadata
	}
	return nil
}

// GetMultitenancyKeyMetadata function
func (tm TableMetadata) GetMultitenancyKeyMetadata() *FieldMetadata {
	metadata, ok := tm.fields[tm.multitenancyKeyField]
	if ok {
		return &metadata
	}
	return nil
}

// GetPrimaryKeyColumnName function
func (tm TableMetadata) GetPrimaryKeyColumnName() string {
	metadata := tm.GetPrimaryKeyMetadata()
	if metadata != nil {
		return metadata.columnName
	}
	return ""
}

// GetPrimaryKeyFieldName function
func (tm TableMetadata) GetPrimaryKeyFieldName() string {
	metadata := tm.GetPrimaryKeyMetadata()
	if metadata != nil {
		return metadata.name
	}
	return ""
}

// GetMultitenancyKeyColumnName function
func (tm TableMetadata) GetMultitenancyKeyColumnName() string {
	metadata := tm.GetMultitenancyKeyMetadata()
	if metadata != nil {
		return metadata.columnName
	}
	return ""
}

// GetFields returns the fields in the order they appear in the struct
func (tm TableMetadata) GetFields() []FieldMetadata {
	fields := []FieldMetadata{}
	for _, key := range tm.fieldOrder {
		fields = append(fields, tm.fields[key])
	}
	return fields
}

// GetField returns the fields in the order they appear in the struct
func (tm TableMetadata) GetField(fieldName string) FieldMetadata {
	return tm.fields[fieldName]
}

// GetTableMetadata function
func GetTableMetadata(data interface{}) (*TableMetadata, error) {
	// Verify that we've been passed valid input
	t := reflect.TypeOf(data)
	var tableMetadata *TableMetadata

	if t.Kind() == reflect.Slice {
		tableMetadata = TableMetadataFromType(t.Elem())
	} else if t.Kind() == reflect.Struct {
		tableMetadata = TableMetadataFromType(t)
	} else {
		return nil, errors.New("Can only get metadata structs or slices of structs")
	}

	if tableMetadata.tableName == "" {
		return nil, errors.New("No table name specified in struct metadata")
	}
	return tableMetadata, nil
}

// TableMetadataFromType gets table metadata from a reflect type
func TableMetadataFromType(t reflect.Type) *TableMetadata {
	var metadata metadata.Metadata
	tableMetadata := TableMetadata{
		fields: map[string]FieldMetadata{},
	}
	children := []Child{}
	lookups := []Lookup{}
	foreignKeys := []ForeignKey{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		kind := field.Type.Kind()

		tagsMap := GetStructTagsMap(field, picardTagKey)
		_, hasTableName := tagsMap["tablename"]
		_, isPrimaryKey := tagsMap["primary_key"]
		_, isMultitenancyKey := tagsMap["multitenancy_key"]
		columnName, hasColumnName := tagsMap["column"]
		_, isLookup := tagsMap["lookup"]
		_, isChild := tagsMap["child"]
		_, isRequired := tagsMap["required"]
		_, isForeignKey := tagsMap["foreign_key"]
		// _, isReference := tagsMap["reference"]
		_, isEncrypted := tagsMap["encrypted"]
		_, isJSONB := tagsMap["jsonb"]
		auditType, _ := tagsMap["audit"]

		if field.Type == reflect.TypeOf(metadata) {
			if hasTableName {
				tableMetadata.tableName = tagsMap["tablename"]
			}
		}

		if hasColumnName {
			var relatedField reflect.StructField
			if isForeignKey {
				relatedField, _ = t.FieldByName(tagsMap["related"])
			}

			tableMetadata.fields[field.Name] = FieldMetadata{
				name:              field.Name,
				isEncrypted:       isEncrypted,
				isJSONB:           isJSONB,
				isMultitenancyKey: isMultitenancyKey,
				isPrimaryKey:      isPrimaryKey,
				isFK:              isForeignKey,
				relatedField:      relatedField,
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
			var keyMapping string
			var valueMappingMap map[string]string
			var groupingCriteriaMap map[string]string
			keyMappingString := tagsMap["key_mapping"]
			valueMappingString := tagsMap["value_mappings"]
			groupingCriteriaString := tagsMap["grouping_criteria"]
			_, deleteOrphans := tagsMap["delete_orphans"]

			if groupingCriteriaString != "" {
				groupingCriteriaMap = map[string]string{}
				groupingCriteriaArray := strings.Split(groupingCriteriaString, "&")
				for _, groupingCriteria := range groupingCriteriaArray {
					groupingCriteriaSplit := strings.Split(groupingCriteria, "->")
					groupingCriteriaMap[groupingCriteriaSplit[0]] = groupingCriteriaSplit[1]
				}
			}

			if keyMappingString != "" {
				keyMapping = keyMappingString
			}

			if valueMappingString != "" {
				valueMappingMap = map[string]string{}
				valueMappingArray := strings.Split(valueMappingString, "&")
				for _, valueMap := range valueMappingArray {
					valueMapSplit := strings.Split(valueMap, "->")
					valueMappingMap[valueMapSplit[0]] = valueMapSplit[1]
				}
			}

			children = append(children, Child{
				FieldName:        field.Name,
				FieldType:        field.Type,
				FieldKind:        kind,
				ForeignKey:       tagsMap["foreign_key"],
				KeyMapping:       keyMapping,
				ValueMappings:    valueMappingMap,
				GroupingCriteria: groupingCriteriaMap,
				DeleteOrphans:    deleteOrphans,
			})

		}

		if isLookup && !isForeignKey {
			lookups = append(lookups, Lookup{
				MatchDBColumn:       tagsMap["column"],
				MatchObjectProperty: field.Name,
			})
		}

		if isForeignKey {

			relatedField, hasRelatedField := t.FieldByName(tagsMap["related"])

			if hasRelatedField {
				tableMetadata := TableMetadataFromType(relatedField.Type)
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

// GetStructTagsMap generates a map of struct tag to values
// Example
// 	input: testKeyOne=test_value_one,testKeyTwo=test_value_two
// 	output: map[string]string{"testKeyOne": "test_value_one", "testKeyTwo": "test_value_two"
func GetStructTagsMap(field reflect.StructField, tagType string) map[string]string {
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
