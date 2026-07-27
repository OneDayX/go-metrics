package service

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
