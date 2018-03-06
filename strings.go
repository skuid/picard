package picard

func stringSliceContainsKey(strings []string, key string) bool {
	for _, item := range strings {
		if item == key {
			return true
		}
	}
	return false
}
