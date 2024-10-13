package lib

// Replace replaces the value of a pointer variable with a new value and returns a function
// that restores the original value.
func Replace[T any](k *T, v T) func() {
	value := *k
	*k = v
	return func() {
		*k = value
	}
}
