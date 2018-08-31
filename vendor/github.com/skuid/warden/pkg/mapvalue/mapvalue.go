package mapvalue

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
)

func IsString(sourceMap map[string]interface{}, key string) error {
	// Check that key exists in map
	val, ok := sourceMap[key]
	if !ok {
		return fmt.Errorf("%s must be provided", key)
	}

	// Check that value for key is a string
	_, ok = val.(string)
	if !ok {
		return fmt.Errorf("%s found with wrong type: expected string", key)
	}
	return nil
}

func String(sourceMap map[string]interface{}, key string) string {
	valAsString, ok := sourceMap[key].(string)
	if !ok {
		return ""
	}
	return valAsString
}
func Bool(sourceMap map[string]interface{}, key string) bool {
	valAsBool, ok := sourceMap[key].(bool)
	if !ok {
		return false
	}
	return valAsBool
}

func IsMapSlice(sourceMap map[string]interface{}, key string) error {
	// Check that key exists in map
	val, ok := sourceMap[key]
	if !ok {
		return fmt.Errorf("%s must be provided", key)
	}

	// Check that value of key is a slice
	valAsSlice, ok := val.([]interface{})
	if !ok {
		return fmt.Errorf("%s found with wrong type: expected JSON array", key)
	}

	// Check that the value of each element is a map
	for index, nestedValue := range valAsSlice {
		_, ok = nestedValue.(map[string]interface{})
		if !ok {
			return fmt.Errorf("Object at index %d in array found with wrong type: expected JSON object", index)
		}
	}

	return nil
}

func MapSlice(sourceMap map[string]interface{}, key string) []map[string]interface{} {
	var valAsMapSlice []map[string]interface{}
	// The type switch statement here complicates things, but it allows us to handle unmarshaled []interface{} slices
	// as well as already strongly-typed []map[string]interface{}, without worrying about the underlying abstraction.
	switch sourceMap[key].(type) {
	case []interface{}:
		// If sourceMap has the type []interface{}, then cast each value into a more strongly typed container.
		temp := sourceMap[key].([]interface{})
		valAsMapSlice = make([]map[string]interface{}, len(temp))
		for i, v := range temp {
			castVal, ok := v.(map[string]interface{})
			if !ok {
				return []map[string]interface{}{}
			}
			valAsMapSlice[i] = castVal
		}
	case []map[string]interface{}:
		// If sourceMap[key] is already typed as []map[string]interface{}, then we're fine and can proceed without casting each value.
		valAsMapSlice = sourceMap[key].([]map[string]interface{})
	default:
		return []map[string]interface{}{}
	}
	return valAsMapSlice
}

func StringSlice(sourceMap map[string]interface{}, key string) []string {
	valAsSlice, ok := sourceMap[key].([]interface{})
	if !ok {
		return []string{}
	}

	valAsStringSlice := make([]string, len(valAsSlice))
	for index, nestedValue := range valAsSlice {
		valAsStringSlice[index], ok = nestedValue.(string)
		if !ok {
			return []string{}
		}
	}
	return valAsStringSlice
}

func IsMap(sourceMap map[string]interface{}, key string) error {
	// Check that key exists in map
	val, ok := sourceMap[key]
	if !ok {
		return fmt.Errorf("%s must be provided", key)
	}

	// Check that value of key is a map
	_, ok = val.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s found with wrong type: expected JSON object", key)
	}

	return nil
}

func Map(sourceMap map[string]interface{}, key string) map[string]interface{} {
	val, ok := sourceMap[key].(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return val
}

func StringSliceContainsKey(strings []string, key string) bool {
	for _, item := range strings {
		if item == key {
			return true
		}
	}
	return false
}

func IsValidUUID(u string) bool {
	_, err := uuid.FromString(u)
	return err == nil
}
