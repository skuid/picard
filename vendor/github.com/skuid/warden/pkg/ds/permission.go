package ds

import (
	"time"

	"github.com/skuid/picard"
)

// Profile structure
type Profile struct {
	Name          string        `json:"name" metadata-json:"name"`
	PermissionSet PermissionSet `json:"permissionSet" metadata-json:"permissionSet"`
}

// GetDataSourcePermissions returns the data source permisisons for this profile and also makes sure
// that each DataSourcePermission instance has the correct data populated on it
func (p *Profile) GetDataSourcePermissions() []DataSourcePermission {
	p.PermissionSet.Name = p.Name
	return p.PermissionSet.GetDataSourcePermissions()
}

// PermissionSet structure
type PermissionSet struct {
	Name                  string                           `json:"name"`
	DataSourcePermissions map[string]*DataSourcePermission `json:"dataSourcePermissions" metadata-json:"dataSourcePermissions" picard:"child,key_mapping=DataSource.Name"`
}

// GetDataSourcePermissions returns the data source permisisons for this permission set and also makes sure
// that each DataSourcePermission instance has the correct data populated on it
func (ps *PermissionSet) GetDataSourcePermissions() []DataSourcePermission {
	dsPermissions := make([]DataSourcePermission, len(ps.DataSourcePermissions))
	// Loop over the Data Source Object Permissions and Import them
	var i int
	for dsName, dsPermission := range ps.DataSourcePermissions {
		dsPermission.DataSource.Name = dsName
		dsPermission.PermissionSetID = ps.Name
		dsPermissions[i] = *dsPermission
		i++
	}
	return dsPermissions
}

// DataSourcePermission structure
type DataSourcePermission struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=data_source_permission"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `json:"organization_id" picard:"multitenancy_key,column=organization_id"`

	CreatedAt       time.Time     `json:"created_at" picard:"column=created_at,audit=created_at"`
	UpdatedAt       time.Time     `json:"updated_at" picard:"column=updated_at,audit=updated_at"`
	DataSourceID    string        `json:"data_source_id" picard:"foreign_key,lookup,required,related=DataSource,column=data_source_id"`
	DataSource      DataSourceNew `json:"-" validate:"-"`
	UpdatedByID     string        `json:"updated_by_id" picard:"column=updated_by_id,audit=updated_by"`
	CreatedByID     string        `json:"created_by_id" picard:"column=created_by_id,audit=created_by"`
	PermissionSetID string        `json:"permission_set_id" picard:"lookup,column=permission_set_id"`

	EntityPermissions map[string]EntityPermission `json:"dataSourceObjectPermissions" metadata-json:"dataSourceObjectPermissions" picard:"child,grouping_criteria=Entity.DataSourceID->DataSourceID&PermissionSetID->PermissionSetID,key_mapping=Entity.Name,value_mappings=DataSource.Name->Entity.DataSource.Name&PermissionSetID->PermissionSetID"`
}

// EntityPermission structure
type EntityPermission struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=data_source_object_permission"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `json:"organization_id" picard:"multitenancy_key,column=organization_id"`

	CreatedAt       time.Time `json:"created_at" picard:"column=created_at,audit=created_at"`
	UpdatedAt       time.Time `json:"updated_at" picard:"column=updated_at,audit=updated_at"`
	EntityID        string    `json:"data_source_object_id" picard:"foreign_key,lookup,required,related=Entity,column=data_source_object_id"`
	Entity          EntityNew `json:"-" validate:"-"`
	UpdatedByID     string    `json:"updated_by_id" picard:"column=updated_by_id,audit=updated_by"`
	CreatedByID     string    `json:"created_by_id" picard:"column=created_by_id,audit=created_by"`
	PermissionSetID string    `json:"permission_set_id" picard:"lookup,column=permission_set_id"`

	Createable bool `json:"createable" metadata-json:"createable" picard:"column=createable"`
	Queryable  bool `json:"queryable" metadata-json:"queryable" picard:"column=queryable"`
	Updateable bool `json:"updateable" metadata-json:"updateable" picard:"column=updateable"`
	Deleteable bool `json:"deleteable" metadata-json:"deleteable" picard:"column=deleteable"`

	FieldPermissions     map[string]FieldPermission     `json:"dataSourceFieldPermissions" metadata-json:"dataSourceFieldPermissions,omitempty" picard:"child,grouping_criteria=EntityField.EntityID->EntityID&PermissionSetID->PermissionSetID,key_mapping=EntityField.Name,value_mappings=Entity.DataSource.Name->EntityField.Entity.DataSource.Name&Entity.Name->EntityField.Entity.Name&PermissionSetID->PermissionSetID"`
	ConditionPermissions map[string]ConditionPermission `json:"dataSourceConditionPermissions" metadata-json:"dataSourceConditionPermissions,omitempty" picard:"child,grouping_criteria=EntityCondition.EntityID->EntityID&PermissionSetID->PermissionSetID,key_mapping=EntityCondition.Name,value_mappings=Entity.DataSource.Name->EntityCondition.Entity.DataSource.Name&Entity.Name->EntityCondition.Entity.Name&PermissionSetID->PermissionSetID"`
}

// FieldPermission structure
type FieldPermission struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=data_source_field_permission"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `json:"organization_id" picard:"multitenancy_key,column=organization_id"`

	CreatedAt       time.Time      `json:"created_at" picard:"column=created_at,audit=created_at"`
	UpdatedAt       time.Time      `json:"updated_at" picard:"column=updated_at,audit=updated_at"`
	EntityFieldID   string         `json:"data_source_field_id" picard:"foreign_key,lookup,required,related=EntityField,column=data_source_field_id"`
	EntityField     EntityFieldNew `json:"-" validate:"-"`
	UpdatedByID     string         `json:"updated_by_id" picard:"column=updated_by_id,audit=updated_by"`
	CreatedByID     string         `json:"created_by_id" picard:"column=created_by_id,audit=created_by"`
	PermissionSetID string         `json:"permission_set_id" picard:"lookup,column=permission_set_id"`

	Createable bool `json:"createable" metadata-json:"createable" picard:"column=createable"`
	Queryable  bool `json:"queryable" metadata-json:"queryable" picard:"column=queryable"`
	Updateable bool `json:"updateable" metadata-json:"updateable" picard:"column=updateable"`
	Deleteable bool `json:"deleteable" picard:"column=deleteable"`
}

// ConditionPermission structure
type ConditionPermission struct {
	Metadata       picard.Metadata `json:"-" picard:"tablename=data_source_condition_permission"`
	ID             string          `json:"id" picard:"primary_key,column=id"`
	OrganizationID string          `json:"organization_id" picard:"multitenancy_key,column=organization_id"`

	CreatedAt         time.Time          `json:"created_at" picard:"column=created_at,audit=created_at"`
	UpdatedAt         time.Time          `json:"updated_at" picard:"column=updated_at,audit=updated_at"`
	EntityConditionID string             `json:"data_source_condition_id" picard:"foreign_key,lookup,required,related=EntityCondition,column=data_source_condition_id"`
	EntityCondition   EntityConditionNew `json:"-" validate:"-"`
	UpdatedByID       string             `json:"updated_by_id" picard:"column=updated_by_id,audit=updated_by"`
	CreatedByID       string             `json:"created_by_id" picard:"column=created_by_id,audit=created_by"`
	PermissionSetID   string             `json:"permission_set_id" picard:"lookup,column=permission_set_id"`

	AlwaysOn bool `json:"alwaysOn" metadata-json:"alwaysOn" picard:"column=always_on"`
}
