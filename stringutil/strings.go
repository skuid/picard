package stringutil

// StringSliceContainsKey determines if a string is present in a slice of strings
func StringSliceContainsKey(strings []string, key string) bool {
	for _, item := range strings {
		if item == key {
			return true
		}
	}
	return false
}
