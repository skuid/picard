/*
Package dbchange abstrats SQL upsert operations into structs
*/
package dbchange

import (
	"reflect"

	"github.com/skuid/picard/tags"
)

// Type is an enum for the type of change being made. Insert, Update or Delete
type Type int

const (
	// Insert Type
	Insert Type = iota
	// Update Type
	Update
	// Delete Type
	Delete
)

// Change structure
type Change struct {
	Changes       map[string]interface{}
	OriginalValue reflect.Value
	Key           string
	Type          Type
}

// ChangeSet structure
type ChangeSet struct {
	Inserts               []Change
	Updates               []Change
	Deletes               []Change
	InsertsHavePrimaryKey bool
	LookupsUsed           []tags.Lookup
}
