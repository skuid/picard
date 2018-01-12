package picard

import (
	"reflect"
)

// ModelMetadata is a field type that can be easily detected by picard.
// Used as an embedded type on a model struct, and certain metadata can be added as struct tags.
// Currently supported tags:
//   tablename
type ModelMetadata struct {
	DefinedFields []string
}

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
	var modelMetadata ModelMetadata
	for i := 0; i < picardStruct.Type().NumField(); i++ {
		field := picardStruct.Type().Field(i)
		if field.Type == reflect.TypeOf(modelMetadata) {
			metadataValue = picardStruct.FieldByName(field.Name)
			break
		}
	}
	return metadataValue
}

func getMetadataFromPicardStruct(picardStruct reflect.Value) ModelMetadata {
	var modelMetadata ModelMetadata
	metadataValue := getMetadataValue(picardStruct)
	if metadataValue.CanInterface() {
		modelMetadata = metadataValue.Interface().(ModelMetadata)
	}
	return modelMetadata
}
