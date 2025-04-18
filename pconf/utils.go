package pconf

func Ptrify[T any](v T) *T {
	z := v
	return &z
}
