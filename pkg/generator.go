package kr

import "crypto/rand"

type KeySize int

const (
	KeySize128 KeySize = 16
	KeySize192 KeySize = 24
	KeySize256 KeySize = 32
	KeySize512 KeySize = 64
)

type KeyGenerator interface {
	Generate() (any, error)
}

type keyGenerator struct {
	keySize int
}

func NewKeyGenerator(size KeySize) KeyGenerator {
	return &keyGenerator{
		keySize: int(size),
	}
}

func (g *keyGenerator) Generate() (any, error) {
	key := make([]byte, g.keySize)

	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}
