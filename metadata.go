package picard

import (
	"reflect"
)

func addDefinedField(metadataValue reflect.Value, fieldName string) {
	// Put Undefined values into the Undefined nested struct
	if metadataValue.IsValid() {
		definedFields := metadataValue.FieldByName("DefinedFields")
		definedFields.Set(reflect.Append(definedFields, reflect.ValueOf(fieldName)))
	}
	return
}

func getMetadataValue(picardStruct reflect.Value) reflect.Value {
	var metadataValue reflect.Value
	var structMetadata StructMetadata
	for i := 0; i < picardStruct.Type().NumField(); i++ {
		field := picardStruct.Type().Field(i)
		if field.Type == reflect.TypeOf(structMetadata) {
			metadataValue = picardStruct.FieldByName(field.Name)
			break
		}
	}
	return metadataValue
}

func getMetadataFromPicardStruct(picardStruct reflect.Value) StructMetadata {
	var structMetadata StructMetadata
	metadataValue := getMetadataValue(picardStruct)
	if metadataValue.CanInterface() {
		structMetadata = metadataValue.Interface().(StructMetadata)
	}
	return structMetadata
}
