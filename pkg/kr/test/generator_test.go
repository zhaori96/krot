package kr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zhaori96/kr/pkg/kr"
)

type MockKeyGenerator struct {
	mock.Mock
}

func (m *MockKeyGenerator) Generate() string {
	args := m.Called()
	return args.String(0)
}

func TestKeyGenerator(t *testing.T) {

	t.Run("Should generate 128 bit key", func(t *testing.T) {
		generator := kr.NewKeyGenerator(kr.KeySize128)

		key, err := generator.Generate()
		assert.NoError(t, err)

		assert.NotEmpty(t, key)
	})

	t.Run("Should generate 192 bit key", func(t *testing.T) {
		generator := kr.NewKeyGenerator(kr.KeySize192)

		key, err := generator.Generate()
		assert.NoError(t, err)

		assert.NotEmpty(t, key)
	})

	t.Run("Should generate 256 bit key", func(t *testing.T) {
		generator := kr.NewKeyGenerator(kr.KeySize256)

		key, err := generator.Generate()
		assert.NoError(t, err)

		assert.NotEmpty(t, key)
	})

	t.Run("Should generate 512 bit key", func(t *testing.T) {
		generator := kr.NewKeyGenerator(kr.KeySize512)

		key, err := generator.Generate()
		assert.NoError(t, err)

		assert.NotEmpty(t, key)
	})

}
