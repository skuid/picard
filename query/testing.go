package query

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/skuid/picard/metadata"
)

func fmtSQL(sql string) string {
	str := strings.Replace(heredoc.Doc(sql), "\n", " ", -1)
	str = strings.Replace(str, "\t", "", -1)
	return strings.Trim(str, " ")
}

func fmtSQLRegex(sql string) string {
	str := fmtSQL(sql)
	str = strings.Replace(str, ".", "\\.", -1)
	str = strings.Replace(str, "$", "\\$", -1)
	return str
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
