package reflectutil

import (
	"reflect"

	"github.com/skuid/picard/tags"
)

// IsZeroValue returns true if the value provided is the zero value for its type
func IsZeroValue(v reflect.Value) bool {
	if v.CanInterface() {
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
	return false
}

// GetPK returns the primary key for a struct
func GetPK(val reflect.Value) (*reflect.Value, bool) {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		ptags := tags.GetStructTagsMap(field, "picard")
		if _, isPK := ptags["primary_key"]; isPK {
			fv := val.FieldByName(field.Name)
			return &fv, true
		}
	}

	return nil, false
}

/*
ReflectTableInfo will return the table name and primary key name from the type
*/
func ReflectTableInfo(typ reflect.Type) (string, string) {
	var tblName string
	var primaryKey string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		typName := field.Type.Name()
		ptags := tags.GetStructTagsMap(field, "picard")
		if typName == "Metadata" {
			tblName = ptags["tablename"]
		}
		if _, isPK := ptags["primary_key"]; isPK {
			primaryKey = ptags["column"]
		}
	}

	return tblName, primaryKey
}
