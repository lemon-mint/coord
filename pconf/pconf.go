package pconf

type Config interface {
	Apply(g interface{})
}
