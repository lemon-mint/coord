package randpool

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"log"
	"runtime"
	"sync"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/sys/cpu"
)

var _csprng_fallback = func() *chacha20.Cipher {
	var initdata [12 + 32]byte // 12 byte nonce, 32 byte key
	_, err := io.ReadFull(rand.Reader, initdata[:])
	if err != nil {
		panic(err)
	}
	c, err := chacha20.NewUnauthenticatedCipher(initdata[12:], initdata[:12])
	if err != nil {
		panic(err)
	}
	return c
}()

type chacha20rng struct {
	c    *chacha20.Cipher
	used uint64
}

var _chacha20rngPool sync.Pool = sync.Pool{
	New: func() interface{} {
		var initdata [12 + 32]byte // 12 byte nonce, 32 byte key
		err := _bsysrand(initdata[:])
		if err != nil {
			// if system rand fails, use fallback and print log
			log.Println("randpool: chacha20rng init failed to read from system rand, using fallback")
			_csprng_fallback.XORKeyStream(initdata[:], initdata[:])
		}
		c, err := chacha20.NewUnauthenticatedCipher(initdata[12:], initdata[:12])
		if err != nil {
			panic(err) // should never happen
		}
		return &chacha20rng{
			c: c,
		}
	},
}

func _chacha20rng() *chacha20rng {
	return _chacha20rngPool.Get().(*chacha20rng)
}

func _CHACHA20_RAND(dst []byte) {
	c := _chacha20rng()
	c.used += uint64(len(dst))
	c.c.XORKeyStream(dst, dst)
	if c.used < 50*1<<30 {
		// Return to pool only if we haven't used more than 50GiB
		_chacha20rngPool.Put(c)
	}
}

type aesrng struct {
	c      cipher.Block
	stream cipher.Stream
	used   uint64
}

var _aesctrprngrandPool sync.Pool = sync.Pool{
	New: func() interface{} {
		var initdata [16 + 32]byte // 16 byte nonce, 32 byte key
		err := _bsysrand(initdata[:])
		if err != nil {
			// if system rand fails, use fallback and print log
			log.Println("randpool: aesctrprng init failed to read from system rand, using fallback")
			_csprng_fallback.XORKeyStream(initdata[:], initdata[:])
		}
		c, err := aes.NewCipher(initdata[16:])
		if err != nil {
			panic(err) // should never happen
		}
		return &aesrng{
			c:      c,
			stream: cipher.NewCTR(c, initdata[:16]),
		}
	},
}

func _aesctrprng() *aesrng {
	return _aesctrprngrandPool.Get().(*aesrng)
}

func _AESCTRPRNG_RAND(dst []byte) {
	c := _aesctrprng()
	c.used += uint64(len(dst))
	c.stream.XORKeyStream(dst, dst)
	if c.used < 50*1<<30 {
		// Return to pool only if we haven't used more than 50GiB
		_aesctrprngrandPool.Put(c)
	}
}

var useAES bool = false

func init() {
	// Use AES if available on arm64 or amd64
	if (runtime.GOARCH == "arm64" && cpu.ARM64.HasAES) ||
		(runtime.GOARCH == "amd64" && cpu.X86.HasAES) ||
		(runtime.GOARCH == "arm64" && runtime.GOOS == "darwin") || // Apple M1, M2, etc.
		(runtime.GOARCH == "amd64" && runtime.GOOS == "darwin") { // Intel Macs or Rosetta
		useAES = true
	}
}

func CSPRNG_RAND(dst []byte) {
	if useAES {
		_AESCTRPRNG_RAND(dst)
	} else {
		_CHACHA20_RAND(dst)
	}
}
