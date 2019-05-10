package query

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/skuid/picard/metadata"
)

func fmtSQL(sql string) string {
	str := strings.Replace(heredoc.Doc(sql), "\n", " ", -1)
	str = strings.Replace(str, "\t", "", -1)
	return strings.Trim(str, " ")
}

func FmtSQLRegex(sql string) string {
	str := fmtSQL(sql)
	str = strings.Replace(str, ".", "\\.", -1)
	str = strings.Replace(str, "$", "\\$", -1)
	str = strings.Replace(str, "(", "\\(", -1)
	str = strings.Replace(str, ")", "\\)", -1)
	return fmt.Sprintf("^%s", str)
}

type grandParentModel struct {
	Metadata       metadata.Metadata `picard:"tablename=grandparentmodel"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	Age            int               `json:"age" picard:"lookup,column=age"`
	Toys           []toyModel        `json:"toys" picard:"child,foreign_key=ParentID"`
	Children       []parentModel     `json:"children" picard:"child,foreign_key=ParentID"`
	Animals        []petModel        `json:"animals" picard:"child,foreign_key=ParentID"`
}
type parentModel struct {
	Metadata       metadata.Metadata     `picard:"tablename=parentmodel"`
	ID             string                `json:"id" picard:"primary_key,column=id"`
	OrganizationID string                `picard:"multitenancy_key,column=organization_id"`
	Name           string                `json:"name" picard:"lookup,column=name"`
	ParentID       string                `picard:"foreign_key,lookup,required,related=GrandParent,column=parent_id"`
	GrandParent    grandParentModel      `json:"parent" validate:"-"`
	Children       []childModel          `json:"children" picard:"child,foreign_key=ParentID"`
	Animals        []petModel            `json:"animals" picard:"child,foreign_key=ParentID"`
	ChildrenMap    map[string]childModel `picard:"child,foreign_key=ParentID,key_mapping=Name"`
}

type childModel struct {
	Metadata metadata.Metadata `picard:"tablename=childmodel"`

	ID             string      `json:"id" picard:"primary_key,column=id"`
	OrganizationID string      `picard:"multitenancy_key,column=organization_id"`
	Name           string      `json:"name" picard:"lookup,column=name"`
	ParentID       string      `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         parentModel `json:"parent" validate:"-"`
	Toys           []toyModel  `json:"children" picard:"child,foreign_key=ParentID"`
}

type toyModel struct {
	Metadata metadata.Metadata `picard:"tablename=toymodel"`

	ID             string     `json:"id" picard:"primary_key,column=id"`
	OrganizationID string     `picard:"multitenancy_key,column=organization_id"`
	Name           string     `json:"name" picard:"lookup,column=name"`
	ParentID       string     `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         childModel `json:"parent" validate:"-"`
}

type petModel struct {
	Metadata metadata.Metadata `picard:"tablename=petmodel"`

	ID             string      `json:"id" picard:"primary_key,column=id"`
	OrganizationID string      `picard:"multitenancy_key,column=organization_id"`
	Name           string      `json:"name" picard:"lookup,column=name"`
	ParentID       string      `picard:"foreign_key,lookup,required,related=Parent,column=parent_id"`
	Parent         parentModel `json:"parent" validate:"-"`
}

// Let's explore relationships that run the other way, where the parent has an
// FK to a child.
type object struct {
	Metadata       metadata.Metadata `picard:"tablename=object"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	Fields         []field           `json:"fields" picard:"child,foreign_key=ObjectID"`
}

type field struct {
	Metadata       metadata.Metadata `picard:"tablename=field"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	ObjectID       string            `picard:"foreign_key,lookup,required,related=object,column=object_id"`
	Object         object            `json:"object" validate:"-"`
	ReferenceTo    referenceTo       `json:"reference" picard:"reference,column=reference_id"`
}

type referenceTo struct {
	Metadata       metadata.Metadata `picard:"tablename=reference_to"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	RefFieldID     string            `picard:"column=reference_field_id"`
	RefField       refField          `json:"field" picard:"reference,column=reference_field_id"`
}

type refField struct {
	Metadata       metadata.Metadata `picard:"tablename=field"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	RefObject      refObject         `json:"object" picard:"reference,column=reference_object_id"`
}

type refObject struct {
	Metadata       metadata.Metadata `picard:"tablename=object"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
}
