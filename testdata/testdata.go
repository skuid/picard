package testdata

import (
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/skuid/picard/metadata"
)

// These all explore relationships where a child table has an FK to the parent
// as a 1:M relationship
type GrandParentModel struct {
	Metadata       metadata.Metadata `picard:"tablename=grandparentmodel"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	Age            int               `json:"age" picard:"lookup,column=age"`
	Toys           []ToyModel        `json:"toys" picard:"child,foreign_key=ParentID"`
	Children       []ParentModel     `json:"children" picard:"child,foreign_key=ParentID"`
	Animals        []PetModel        `json:"animals" picard:"child,foreign_key=ParentID"`
}

type ParentModel struct {
	Metadata             metadata.Metadata     `picard:"tablename=parentmodel"`
	ID                   string                `json:"id" picard:"primary_key,column=id"`
	OrganizationID       string                `picard:"multitenancy_key,column=organization_id"`
	Name                 string                `json:"name" picard:"lookup,column=name"`
	ParentID             string                `picard:"foreign_key,lookup,required,related=GrandParent,column=parent_id"`
	GrandParent          GrandParentModel      `json:"parent" validate:"-"`
	OtherParentID        string                `picard:"foreign_key,related=GrandMother,column=other_parent_id"`
	GrandMother          GrandParentModel      `json:"other_parent" validate:"-"`
	Children             []ChildModel          `json:"children" picard:"child,foreign_key=ParentID"`
	Animals              []PetModel            `json:"animals" picard:"child,foreign_key=ParentID"`
	ChildrenMap          map[string]ChildModel `picard:"child,foreign_key=ParentID,key_mapping=Name"`
	ChildrenWithGrouping []ChildModel          `picard:"child,grouping_criteria=ParentID->ID"`
	ToysWithGrouping     []ToyModel            `picard:"child,grouping_criteria=Parent.ParentID->ID"`
}

type ChildModel struct {
	Metadata metadata.Metadata `picard:"tablename=childmodel"`

	ID             string      `json:"id" picard:"primary_key,column=id"`
	OrganizationID string      `picard:"multitenancy_key,column=organization_id"`
	Name           string      `json:"name" picard:"lookup,column=name"`
	ParentID       string      `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         ParentModel `json:"parent" validate:"-"`
	Toys           []ToyModel  `json:"children" picard:"child,foreign_key=ParentID"`
}

type PersonModel struct {
	Metadata       metadata.Metadata `picard:"tablename=personmodel"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
}

type SiblingJunctionModel struct {
	Metadata metadata.Metadata `picard:"tablename=siblingjunction"`

	ID             string      `json:"id" picard:"primary_key,column=id"`
	OrganizationID string      `picard:"multitenancy_key,column=organization_id"`
	ChildID        string      `json:"child_id" picard:"foreign_key,lookup,required,related=Child,column=child_id"`
	Child          PersonModel `json:"child" validate:"-"`
	SiblingID      string      `picard:"foreign_key,lookup,required,related=Sibling,column=sibling_id"`
	Sibling        PersonModel `json:"sibling" validate:"-"`
}

type ToyModel struct {
	Metadata metadata.Metadata `picard:"tablename=toymodel"`

	ID             string     `json:"id" picard:"primary_key,column=id"`
	OrganizationID string     `picard:"multitenancy_key,column=organization_id"`
	Name           string     `json:"name" picard:"lookup,column=name"`
	ParentID       string     `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         ChildModel `json:"parent" validate:"-"`
}

type PetModel struct {
	Metadata metadata.Metadata `picard:"tablename=petmodel"`

	ID             string      `json:"id" picard:"primary_key,column=id"`
	OrganizationID string      `picard:"multitenancy_key,column=organization_id"`
	Name           string      `json:"name" picard:"lookup,column=name"`
	ParentID       string      `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         ParentModel `json:"parent" validate:"-"`
}

// Config is a sample struct that would go in a jsonb field
type Config struct {
	ConfigA string
	ConfigB string
}

// ParentTestObject sample parent object for tests
type ParentTestObject struct {
	Metadata metadata.Metadata `picard:"tablename=parenttest"`

	ID             string       `json:"id" picard:"primary_key,column=id"`
	OrganizationID string       `picard:"multitenancy_key,column=organization_id"`
	Name           string       `json:"name" picard:"column=name"`
	Children       []TestObject `json:"children" picard:"child,foreign_key=ParentID"`
}

// TestObject sample parent object for tests
type TestObject struct {
	Metadata metadata.Metadata `picard:"tablename=testobject"`

	ID             string                     `json:"id" picard:"primary_key,column=id"`
	OrganizationID string                     `picard:"multitenancy_key,column=organization_id"`
	Name           string                     `json:"name" picard:"lookup,column=name" validate:"required"`
	NullableLookup string                     `json:"nullableLookup" picard:"lookup,column=nullable_lookup"`
	Type           string                     `json:"type" picard:"column=type"`
	IsActive       bool                       `json:"is_active" picard:"column=is_active"`
	Children       []ChildTestObject          `json:"children" picard:"child,foreign_key=ParentID"`
	ChildrenMap    map[string]ChildTestObject `json:"childrenmap" picard:"child,foreign_key=ParentID,key_mapping=Name,value_mappings=Type->OtherInfo"`
	ParentID       string                     `picard:"foreign_key,related=Parent,column=parent_id"`
	Parent         ParentTestObject           `validate:"-"`
	Config         Config                     `json:"config" picard:"jsonb,column=config"`
	CreatedByID    string                     `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID    string                     `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate    time.Time                  `picard:"column=created_at,audit=created_at"`
	UpdatedDate    time.Time                  `picard:"column=updated_at,audit=updated_at"`
}

// TestObject sample parent object for tests
type TestObjectWithOrphans struct {
	Metadata metadata.Metadata `picard:"tablename=testobject"`

	ID             string                     `json:"id" picard:"primary_key,column=id"`
	OrganizationID string                     `picard:"multitenancy_key,column=organization_id"`
	Name           string                     `json:"name" picard:"lookup,column=name" validate:"required"`
	NullableLookup string                     `json:"nullableLookup" picard:"lookup,column=nullable_lookup"`
	Type           string                     `json:"type" picard:"column=type"`
	IsActive       bool                       `json:"is_active" picard:"column=is_active"`
	Children       []ChildTestObject          `json:"children" picard:"child,foreign_key=ParentID,delete_orphans"`
	ChildrenMap    map[string]ChildTestObject `json:"childrenmap" picard:"child,foreign_key=ParentID,key_mapping=Name,value_mappings=Type->OtherInfo,delete_orphans"`
	ParentID       string                     `picard:"foreign_key,related=Parent,column=parent_id"`
	Parent         ParentTestObject           `validate:"-"`
	Config         Config                     `json:"config" picard:"jsonb,column=config"`
	CreatedByID    string                     `picard:"column=created_by_id,audit=created_by"`
	UpdatedByID    string                     `picard:"column=updated_by_id,audit=updated_by"`
	CreatedDate    time.Time                  `picard:"column=created_at,audit=created_at"`
	UpdatedDate    time.Time                  `picard:"column=updated_at,audit=updated_at"`
}

// ChildTestObject sample child object for tests
type ChildTestObject struct {
	Metadata metadata.Metadata `picard:"tablename=childtest"`

	ID               string     `json:"id" picard:"primary_key,column=id"`
	OrganizationID   string     `picard:"multitenancy_key,column=organization_id"`
	Name             string     `json:"name" picard:"lookup,column=name"`
	OtherInfo        string     `picard:"column=other_info"`
	ParentID         string     `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent           TestObject `json:"parent" validate:"-"`
	OptionalParentID string     `picard:"foreign_key,related=OptionalParent,column=optional_parent_id"`
	OptionalParent   TestObject `json:"optional_parent" validate:"-"`
}

// ChildTestObjectWithKeyMap sample child object for tests
type ChildTestObjectWithKeyMap struct {
	Metadata metadata.Metadata `picard:"tablename=childtest"`

	ID               string     `json:"id" picard:"primary_key,column=id"`
	OrganizationID   string     `picard:"multitenancy_key,column=organization_id"`
	Name             string     `json:"name" picard:"lookup,column=name"`
	OtherInfo        string     `picard:"column=other_info"`
	ParentID         string     `json:"parent" picard:"foreign_key,lookup,required,related=Parent,column=parent_id,key_map=Name"`
	Parent           TestObject `validate:"-"`
	OptionalParentID string     `picard:"foreign_key,related=OptionalParent,column=optional_parent_id"`
	OptionalParent   TestObject `json:"optional_parent" validate:"-"`
}

type TestParentSerializedObject struct {
	Metadata metadata.Metadata `picard:"tablename=parent_serialize"`

	ID               string                 `json:"id" picard:"primary_key,column=id"`
	SerializedThings []TestSerializedObject `json:"serialized_things" picard:"jsonb,column=serialized_things"`
}

// SerializedObject sample object to be stored in a JSONB column
type TestSerializedObject struct {
	Name               string `json:"name"`
	Active             bool   `json:"active"`
	NonSerializedField string `json:"-"`
}

func FmtSQL(sql string) string {
	str := strings.Replace(heredoc.Doc(sql), "\n", " ", -1)
	str = strings.Replace(str, "\t", "", -1)
	return strings.Trim(str, " ")
}

//FmtSQLRegex will covert a multiline/heredoc SQL statement into a REGEX version,
// which is useful for testing mock SQL calls. This allows the user to write out
// the SQL without worrying about tabs, newlines, and escaping characters like
// ., $, (, ). It also adds the ^ at the beginning.
func FmtSQLRegex(sql string) string {
	str := FmtSQL(sql)
	str = strings.Replace(str, ".", "\\.", -1)
	str = strings.Replace(str, "$", "\\$", -1)
	str = strings.Replace(str, "(", "\\(", -1)
	str = strings.Replace(str, ")", "\\)", -1)
	return fmt.Sprintf("^%s$", str)
}
