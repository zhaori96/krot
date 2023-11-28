package kr

import "crypto/rand"

type KeySize int

const (
	KeySize128 KeySize = 16
	KeySize192 KeySize = 24
	KeySize256 KeySize = 32
	KeySize512 KeySize = 64
)

// KeyGenerator defines the interface for generating keys. It provides a method
// for generating a key.
type KeyGenerator interface {
	// Generate creates a new key. If the key cannot be generated, it returns an error.
	//
	//     key, err := generator.Generate()
	//     if err != nil {
	//         log.Fatal(err)
	//     }
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
