package randpool

import (
	"bufio"
	"crypto/rand"
	"io"
	"sync"
)

var _sysrandPool sync.Pool = sync.Pool{
	New: func() interface{} {
		return bufio.NewReader(rand.Reader)
	},
}

func _bsysrand(b []byte) error {
	r := _sysrandPool.Get().(*bufio.Reader)
	_, err := io.ReadFull(r, b)
	_sysrandPool.Put(r)
	return err
}

func SYS_RAND(dst []byte) {
	err := _bsysrand(dst)
	if err != nil {
		panic(err)
	}
}
