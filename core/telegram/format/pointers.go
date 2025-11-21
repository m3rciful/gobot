package format

// DerefString safely dereferences a *string and returns a default value if nil.
func DerefString(s *string, defaultVal string) string {
	if s != nil {
		return *s
	}
	return defaultVal
}

// DerefInt safely dereferences a *int and returns a default value if nil.
func DerefInt(i *int, defaultVal int) int {
	if i != nil {
		return *i
	}
	return defaultVal
}
