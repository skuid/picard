package picard

import (
	"reflect"
)

func getMetadataFromValue(t reflect.Value) StructMetadata {
	var structMetadata StructMetadata

	for i := 0; i < t.Type().NumField(); i++ {
		field := t.Type().Field(i)
		if field.Type == reflect.TypeOf(structMetadata) {
			structMetadata = t.FieldByName(field.Name).Interface().(StructMetadata)
			break
		}
	}

	return structMetadata
}
