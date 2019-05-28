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
	Secret         string `json:"secret" picard:"encrypted,column=secret"`
	ObjectID    string      `picard:"foreign_key,lookup,required,related=Object,column=object_id"`
	Object      object      `json:"object" validate:"-"`
	ReferenceID string      `picard:"foreign_key,lookup,required,related=ReferenceTo,column=reference_id"`
	ReferenceTo referenceTo `json:"referenceTo" validate:"-"`
}

type referenceTo struct {
	Metadata       metadata.Metadata `picard:"tablename=reference_to"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	RefFieldID     string            `picard:"foreign_key,related=RefField,column=reference_field_id"`
	RefField       refField          `json:"refField" validate:"-"`
}

type refField struct {
	Metadata       metadata.Metadata `picard:"tablename=field"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
	RefObjectID    string            `picard:"foreign_key,related=RefObject,column=reference_object_id"`
	RefObject      refObject         `json:"object" validate:"-"`
}

type refObject struct {
	Metadata       metadata.Metadata `picard:"tablename=object"`
	ID             string            `json:"id" picard:"primary_key,column=id"`
	OrganizationID string            `picard:"multitenancy_key,column=organization_id"`
	Name           string            `json:"name" picard:"lookup,column=name"`
}
