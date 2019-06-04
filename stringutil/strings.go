package stringutil

import (
	"errors"
	"reflect"
	"strings"
)

// StringSliceContainsKey determines if a string is present in a slice of strings
func StringSliceContainsKey(strings []string, key string) bool {
	for _, item := range strings {
		if item == key {
			return true
		}
	}
	return false
}

// GetValueFromLookupString will look through a string recursively to get the property value
func GetValueFromLookupString(value reflect.Value, lookupString string) reflect.Value {
	// If the lookupString has a dot in it, recursively look up the property's value
	propertyKeys := strings.Split(lookupString, ".")
	if len(propertyKeys) > 1 {
		subValue := value.FieldByName(propertyKeys[0])
		return GetValueFromLookupString(subValue, strings.Join(propertyKeys[1:], "."))
	}
	return value.FieldByName(lookupString)
}

// GetStructValue returns the reflected value of a struct interface
func GetStructValue(v interface{}) (reflect.Value, error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	if value.Kind() != reflect.Struct {
		return value, errors.New("Models must be structs")
	}
	return value, nil
}

// GetFilterType returns the type of the interface if it is a struct
// If it is a slice, it returns the type of the elements inside the struct
func GetFilterType(v interface{}) (reflect.Type, error) {
	value := reflect.Indirect(reflect.ValueOf(v))
	kind := value.Kind()
	if kind == reflect.Struct {
		return value.Type(), nil
	} else if kind == reflect.Slice {
		return value.Type().Elem(), nil
	}
	return nil, errors.New("Filter must be struct or slice of structs")
}
