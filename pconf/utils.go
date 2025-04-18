package pconf

func Ptrify[T any](v T) *T {
	return &v
}
