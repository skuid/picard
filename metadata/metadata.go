package metadata

import (
	"reflect"
)

// Metadata is a field type that can be easily detected by picard.
// Used as an embedded type on a model struct, and certain metadata can be added as struct tags.
// Currently supported tags:
//   tablename
type Metadata struct {
	DefinedFields []string
}

func AddDefinedField(metadataValue reflect.Value, fieldName string) {
	// Put Defined values into the Defined nested struct
	if metadataValue.IsValid() {
		definedFields := metadataValue.FieldByName("DefinedFields")
		definedFields.Set(reflect.Append(definedFields, reflect.ValueOf(fieldName)))
	}
	return
}

func InitializeDefinedFields(metadataValue reflect.Value) {
	if metadataValue.IsValid() {
		definedFields := metadataValue.FieldByName("DefinedFields")
		definedFields.Set(reflect.ValueOf([]string{}))
	}
	return
}

func GetMetadataValue(picardStruct reflect.Value) reflect.Value {
	var metadataValue reflect.Value
	var metadata Metadata
	for i := 0; i < picardStruct.Type().NumField(); i++ {
		field := picardStruct.Type().Field(i)
		if field.Type == reflect.TypeOf(metadata) {
			metadataValue = picardStruct.FieldByName(field.Name)
			break
		}
	}
	return metadataValue
}

func GetMetadataFromPicardStruct(picardStruct reflect.Value) Metadata {
	var metadata Metadata
	metadataValue := GetMetadataValue(picardStruct)
	if metadataValue.CanInterface() {
		metadata = metadataValue.Interface().(Metadata)
	}
	return metadata
}
