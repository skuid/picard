package ds

import (
	"time"

	"github.com/skuid/picard"
	"github.com/skuid/warden/pkg/mapvalue"
)

// EntityNew glues together datasources, organizations, field mappings, and conditions.
type EntityNew struct {
	Metadata       picard.Metadata      `json:"-" picard:"tablename=data_source_object"`
	ID             string               `json:"id" picard:"primary_key,column=id"`
	OrganizationID string               `picard:"multitenancy_key,column=organization_id"`
	DataSourceID   string               `json:"data_source_id" picard:"foreign_key,required,lookup,related=DataSource,column=data_source_id"`
	Name           string               `json:"name" metadata-json:"name" picard:"lookup,column=name" validate:"required"`
	Schema         string               `json:"schema" metadata-json:"schema" picard:"column=schema"`
	Label          string               `json:"label" metadata-json:"label" picard:"column=label"`
	LabelPlural    string               `json:"label_plural" metadata-json:"labelPlural" picard:"column=label_plural"`
	DataSource     DataSourceNew        `json:"-" validate:"-"`
	Fields         []EntityFieldNew     `json:"fields" metadata-json:"fields" picard:"child,foreign_key=EntityID,delete_orphans"`
	Conditions     []EntityConditionNew `json:"conditions" metadata-json:"conditions" picard:"child,foreign_key=EntityID,delete_orphans"`
	Createable     bool                 `json:"createable"`
	Queryable      bool                 `json:"queryable"`
	Updateable     bool                 `json:"updateable"`
	Deleteable     bool                 `json:"deleteable"`
	CreatedByID    string               `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID    string               `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate    time.Time            `picard:"column=created_at,audit=created_at"`
	UpdatedDate    time.Time            `picard:"column=updated_at,audit=updated_at"`
	Permissions    []EntityPermission   `json:"-" picard:"child,foreign_key=EntityID"`
}

// GetEntityFilterFromKey returns a EntityNew object suitable for filtering
// in picard based on a key. To support multiple versions of clients, this key
// is sometimes the uuid (for old versions) and sometimes the data source name.
func GetEntityFilterFromKey(dsKey string, entityKey string) EntityNew {
	// This is just temporary until we can get all clients on or past version 0.2.2
	if mapvalue.IsValidUUID(dsKey) {
		return EntityNew{
			ID:           entityKey,
			DataSourceID: dsKey,
		}
	}

	return EntityNew{
		ID: entityKey,
		DataSource: DataSourceNew{
			Name: dsKey,
		},
	}
}

// GetEntityFilterFromKeyByName returns a EntityNew object suitable for filtering
// in picard based on a key. To support multiple versions of clients, this key
// is sometimes the uuid (for old versions) and sometimes the data source name.
func GetEntityFilterFromKeyByName(dsKey string, entityName string) EntityNew {
	// This is just temporary until we can get all clients on or past version 0.2.2
	if mapvalue.IsValidUUID(dsKey) {
		return EntityNew{
			Name:         entityName,
			DataSourceID: dsKey,
		}
	}

	return EntityNew{
		Name: entityName,
		DataSource: DataSourceNew{
			Name: dsKey,
		},
	}
}

// HasChildEntities checks for child relations on any field of this entity
//
// Returns:
// - true if any children exist on any field
// - false if no children exist on any field
func (e *EntityNew) HasChildEntities() bool {
	for _, field := range e.Fields {
		if len(field.ChildRelations) > 0 {
			return true
		}
	}
	return false
}

// RemoveUnimportedChildEntities filters child related on fields of this entity
// to remove any child relations that do not exist in the provided entity slice.
func (e *EntityNew) RemoveUnimportedChildEntities(importedEntities []EntityNew) {
	// Convert the list of DSOs to a list of names.
	var importedEntityNames []string
	for _, element := range importedEntities {
		importedEntityNames = append(importedEntityNames, element.Name)
	}

	// Alter each field to only include the children that are available
	for fieldIndex, field := range e.Fields {
		newChildren := []EntityRelation{}
		for _, childEntity := range field.ChildRelations {
			for _, objectName := range importedEntityNames {
				if objectName == childEntity.Object {
					newChildren = append(newChildren, childEntity)
				}
			}
		}
		// Replace the former relations
		field.ChildRelations = newChildren
		// Replace the field on the parent
		e.Fields[fieldIndex] = field
	}
}

// Entity glues together datasources, organizations, field mappings, and conditions.
type Entity struct {
	ID           string            `json:"id"`
	DataSourceID string            `json:"data_source_id"`
	Name         string            `json:"name"`
	Schema       string            `json:"schema"`
	Label        string            `json:"label"`
	LabelPlural  string            `json:"label_plural"`
	Fields       []EntityField     `json:"fields"`
	Conditions   []EntityCondition `json:"conditions"`
	Createable   bool              `json:"createable"`
	Queryable    bool              `json:"queryable"`
	Updateable   bool              `json:"updateable"`
	Deleteable   bool              `json:"deleteable"`
}

// ToEntityNew upcasts an Entity to EntityNew
func (e Entity) ToEntityNew() *EntityNew {
	efns := make([]EntityFieldNew, len(e.Fields))
	for i := 0; i < len(e.Fields); i++ {
		efns[i] = *e.Fields[i].toEntityFieldNew()
	}

	ecns := make([]EntityConditionNew, len(e.Conditions))
	for i := 0; i < len(e.Conditions); i++ {
		ecns[i] = *e.Conditions[i].toEntityConditionNew()
	}

	return &EntityNew{
		ID:           e.ID,
		DataSourceID: e.DataSourceID,
		Name:         e.Name,
		Schema:       e.Schema,
		Label:        e.Label,
		LabelPlural:  e.LabelPlural,
		Fields:       efns,
		Conditions:   ecns,
		Createable:   e.Createable,
		Queryable:    e.Queryable,
		Updateable:   e.Updateable,
		Deleteable:   e.Deleteable,
	}
}

func (ef EntityField) toEntityFieldNew() *EntityFieldNew {
	return &EntityFieldNew{
		Name:           ef.Name,
		EntityID:       ef.DataSourceObjectID,
		Label:          ef.Label,
		DisplayType:    ef.DisplayType,
		ReadOnly:       ef.ReadOnly,
		IsIDField:      ef.IsIDField,
		IsNameField:    ef.IsNameField,
		Createable:     ef.Createable,
		Queryable:      ef.Queryable,
		Updateable:     ef.Updateable,
		ReferenceTo:    ef.ReferenceTo,
		ChildRelations: ef.ChildRelations,
		Filterable:     ef.Filterable,
		Sortable:       ef.Sortable,
		Groupable:      ef.Groupable,
		Required:       ef.Required,
	}

}

func (ec EntityCondition) toEntityConditionNew() *EntityConditionNew {
	return &EntityConditionNew{
		Name:            ec.Name,
		Type:            ec.Type,
		Field:           ec.Field,
		Value:           ec.Value,
		AlwaysOn:        ec.AlwaysOn,
		ExecuteOnQuery:  ec.ExecuteOnQuery,
		ExecuteOnInsert: ec.ExecuteOnInsert,
		ExecuteOnUpdate: ec.ExecuteOnUpdate,
	}
}

// EntityFieldNew structure
type EntityFieldNew struct {
	Metadata        picard.Metadata       `json:"-" picard:"tablename=data_source_field"`
	ID              string                `json:"id" picard:"primary_key,column=id"`
	OrganizationID  string                `picard:"multitenancy_key,column=organization_id"`
	Name            string                `json:"name" metadata-json:"name" picard:"lookup,column=name" validate:"required"`
	DataSourceID    string                `json:"data_source_id"`
	EntityID        string                `json:"data_source_object_id" picard:"foreign_key,required,lookup,related=Entity,column=data_source_object_id"`
	Entity          EntityNew             `json:"-" validate:"-"`
	Label           string                `json:"label" metadata-json:"label" picard:"column=label"`
	DisplayType     string                `json:"display_type" metadata-json:"displayType" picard:"column=display_type" validate:"required"`
	PicklistEntries []EntityPicklistEntry `json:"picklistEntries" picard:"child,foreign_key=EntityFieldID"`
	ReadOnly        bool                  `json:"readonly" picard:"column=readonly"`
	IsIDField       bool                  `json:"is_id_field" metadata-json:"isIdField" picard:"column=is_id_field"`
	IsNameField     bool                  `json:"is_name_field" metadata-json:"isNameField" picard:"column=is_name_field"`
	Createable      bool                  `json:"createable"`
	Queryable       bool                  `json:"queryable"`
	Updateable      bool                  `json:"updateable"`
	ReferenceTo     []EntityReference     `json:"reference_to" metadata-json:"referenceTo" picard:"jsonb,column=reference_to"`
	ChildRelations  []EntityRelation      `json:"child_relations" metadata-json:"childRelations" picard:"jsonb,column=child_relations"`
	Filterable      bool                  `json:"filterable" metadata-json:"filterable" picard:"column=filterable"`
	Sortable        bool                  `json:"sortable" metadata-json:"sortable" picard:"column=sortable"`
	Groupable       bool                  `json:"groupable" metadata-json:"groupable" picard:"column=groupable"`
	Required        bool                  `json:"required" metadata-json:"required" picard:"column=required"`
	CreatedByID     string                `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID     string                `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate     time.Time             `picard:"column=created_at,audit=created_at"`
	UpdatedDate     time.Time             `picard:"column=updated_at,audit=updated_at"`
	Permissions     []FieldPermission     `json:"-" picard:"child,foreign_key=EntityFieldID"`
}

// EntityField structure
type EntityField struct {
	Name               string            `json:"name"`
	DataSourceObjectID string            `json:"data_source_object_id"`
	Label              string            `json:"label"`
	DisplayType        string            `json:"display_type"`
	ReadOnly           bool              `json:"readonly"`
	IsIDField          bool              `json:"is_id_field"`
	IsNameField        bool              `json:"is_name_field"`
	Createable         bool              `json:"createable"`
	Queryable          bool              `json:"queryable"`
	Updateable         bool              `json:"updateable"`
	ReferenceTo        []EntityReference `json:"reference_to"`
	ChildRelations     []EntityRelation  `json:"child_relations"`
	Filterable         bool              `json:"filterable"`
	Sortable           bool              `json:"sortable"`
	Groupable          bool              `json:"groupable"`
	Required           bool              `json:"required"`
}

// EntityPicklistEntry structure
type EntityPicklistEntry struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=picklist_entry"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `picard:"multitenancy_key,column=organization_id"`
	Active         bool            `json:"active"`
	Value          string          `json:"value" picard:"column=value"`
	Label          string          `json:"label" picard:"column=label"`
	EntityFieldID  string          `json:"data_source_field_id" picard:"foreign_key,required,lookup,related=EntityField,column=data_source_field_id"`
	EntityField    EntityFieldNew  `json:"-" validate:"-"`
	CreatedByID    string          `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID    string          `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate    time.Time       `picard:"column=created_at,audit=created_at"`
	UpdatedDate    time.Time       `picard:"column=updated_at,audit=updated_at"`
	DataSourceID   string          `json:"data_source_id"`
}

// EntityConditionNew structure
type EntityConditionNew struct {
	Metadata        picard.Metadata       `json:"-" picard:"tablename=data_source_condition"`
	ID              string                `json:"id" picard:"primary_key,column=id"`
	OrganizationID  string                `picard:"multitenancy_key,column=organization_id"`
	Name            string                `json:"name" metadata-json:"name" picard:"lookup,column=name" validate:"required"`
	DataSourceID    string                `json:"data_source_id"`
	EntityID        string                `json:"data_source_object_id" picard:"foreign_key,required,lookup,related=Entity,column=data_source_object_id"`
	Entity          EntityNew             `json:"-" validate:"-"`
	Type            string                `json:"type" metadata-json:"type" picard:"column=type" validate:"required"`
	Field           string                `json:"field" metadata-json:"field" picard:"column=field"`
	Value           string                `json:"value" metadata-json:"value" picard:"column=value"`
	AlwaysOn        bool                  `json:"always_on"`
	ExecuteOnQuery  bool                  `json:"execute_on_query" metadata-json:"executeOnQuery" picard:"column=execute_on_query"`
	ExecuteOnInsert bool                  `json:"execute_on_insert" metadata-json:"executeOnInsert" picard:"column=execute_on_insert"`
	ExecuteOnUpdate bool                  `json:"execute_on_update" metadata-json:"executeOnUpdate" picard:"column=execute_on_update"`
	CreatedByID     string                `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID     string                `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate     time.Time             `picard:"column=created_at,audit=created_at"`
	UpdatedDate     time.Time             `picard:"column=updated_at,audit=updated_at"`
	Permissions     []ConditionPermission `json:"-" picard:"child,foreign_key=EntityConditionID"`
}

// EntityCondition structure
type EntityCondition struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Field           string `json:"field"`
	Value           string `json:"value"`
	AlwaysOn        bool   `json:"always_on"`
	ExecuteOnQuery  bool   `json:"execute_on_query"`
	ExecuteOnInsert bool   `json:"execute_on_insert"`
	ExecuteOnUpdate bool   `json:"execute_on_update"`
}

// EntityReference structure
type EntityReference struct {
	Object   string `json:"object" metadata-json:"object"`
	KeyField string `json:"keyfield" metadata-json:"keyfield"`
}

// EntityRelation structure
type EntityRelation struct {
	Object           string `json:"object" metadata-json:"object"`
	KeyField         string `json:"keyfield" metadata-json:"keyfield"`
	RelationshipName string `json:"relationshipName" metadata-json:"relationshipName"`
}

// NewEntityFromMetadata constructs a new Entity Struct from a metadata payload
func NewEntityFromMetadata(metadata map[string]interface{}) EntityNew {
	return EntityNew{
		Name:        mapvalue.String(metadata, "objectName"),
		Label:       mapvalue.String(metadata, "label"),
		LabelPlural: mapvalue.String(metadata, "labelPlural"),
		Schema:      mapvalue.String(metadata, "schemaName"),
		Fields:      NewEntityFieldsFromMetadata(metadata),
	}
}

// NewEntityFieldsFromMetadata takes the incoming metadata map and turns it into
// an array of EntityFields.
func NewEntityFieldsFromMetadata(metadata map[string]interface{}) []EntityFieldNew {
	entityFieldsMetadata := mapvalue.MapSlice(metadata, "fields")
	idFields := mapvalue.StringSlice(metadata, "idFields")
	nameFields := mapvalue.StringSlice(metadata, "nameFields")
	fields := []EntityFieldNew{}

	for _, entityField := range entityFieldsMetadata {
		fieldID := mapvalue.String(entityField, "id")
		childRelations := mapvalue.MapSlice(entityField, "childRelations")
		referenceTo := mapvalue.MapSlice(entityField, "referenceTo")
		picklistEntries := mapvalue.MapSlice(entityField, "picklistEntries")
		fields = append(fields, EntityFieldNew{
			Name:            fieldID,
			Label:           mapvalue.String(entityField, "label"),
			DisplayType:     mapvalue.String(entityField, "displaytype"),
			Filterable:      mapvalue.Bool(entityField, "filterable"),
			IsIDField:       mapvalue.StringSliceContainsKey(idFields, fieldID),
			IsNameField:     mapvalue.StringSliceContainsKey(nameFields, fieldID),
			ChildRelations:  NewChildRelationsFromMetadata(childRelations),
			ReferenceTo:     NewReferenceFromMetadata(referenceTo),
			PicklistEntries: NewPicklistEntriesFromMetadata(picklistEntries),
		})
	}
	return fields
}

// NewPicklistEntriesFromMetadata takes incoming metadata and converts it into
// an array of picklist entries
func NewPicklistEntriesFromMetadata(entryList []map[string]interface{}) []EntityPicklistEntry {
	picklistEntryList := make([]EntityPicklistEntry, len(entryList))
	for index, entry := range entryList {
		picklistEntryList[index] = EntityPicklistEntry{
			Active: mapvalue.Bool(entry, "active"),
			Value:  mapvalue.String(entry, "value"),
			Label:  mapvalue.String(entry, "label"),
		}
	}
	return picklistEntryList
}

// NewChildRelationsFromMetadata takes incoming metadata and converts it into
// an array of EntityRelations
func NewChildRelationsFromMetadata(childMetadataList []map[string]interface{}) []EntityRelation {
	childRelationList := make([]EntityRelation, len(childMetadataList))
	for index, childMetadata := range childMetadataList {
		childRelationList[index] = EntityRelation{
			Object:           mapvalue.String(childMetadata, "objectName"),
			KeyField:         mapvalue.String(childMetadata, "keyField"),
			RelationshipName: mapvalue.String(childMetadata, "relationshipName"),
		}
	}
	return childRelationList
}

// NewReferenceFromMetadata takes incoming metadata and converts it into
// an array of EntityReferences
func NewReferenceFromMetadata(childMetadataList []map[string]interface{}) []EntityReference {
	childReferenceList := make([]EntityReference, len(childMetadataList))
	for index, childMetadata := range childMetadataList {
		childReferenceList[index] = EntityReference{
			Object:   mapvalue.String(childMetadata, "objectName"),
			KeyField: mapvalue.String(childMetadata, "keyField"),
		}
	}
	return childReferenceList
}
