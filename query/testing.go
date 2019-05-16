package query

import (
	"github.com/skuid/picard/metadata"
)

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
	ObjectID       string            `picard:"column=object_id"`
	Object         object            `json:"object" picard:"reference,column=object_id"`
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
