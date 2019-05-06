package tags

import (
	"errors"
	"reflect"
	"strings"

	"github.com/skuid/picard/metadata"
)

const picardTagKey = "picard"

// Association structure
type Association struct {
	Name         string
	Associations []Association
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
	isReference       bool
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

// IsReference function
func (fm FieldMetadata) IsReference() bool {
	return fm.isReference
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
	return tm.foreignKeys
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
	if t.Kind() != reflect.Slice {
		return nil, errors.New("Can only upsert slices")
	}

	tableMetadata := TableMetadataFromType(t.Elem())

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
		_, isReference := tagsMap["reference"]
		_, isEncrypted := tagsMap["encrypted"]
		_, isJSONB := tagsMap["jsonb"]
		auditType, _ := tagsMap["audit"]

		if field.Type == reflect.TypeOf(metadata) {
			if hasTableName {
				tableMetadata.tableName = tagsMap["tablename"]
			}
		}

		if hasColumnName {

			tableMetadata.fields[field.Name] = FieldMetadata{
				name:              field.Name,
				isEncrypted:       isEncrypted,
				isJSONB:           isJSONB,
				isMultitenancyKey: isMultitenancyKey,
				isPrimaryKey:      isPrimaryKey,
				isReference:       isReference,
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

// GetStructTagsMap function
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
