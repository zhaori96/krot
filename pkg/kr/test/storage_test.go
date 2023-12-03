package kr_test

import (
	"context"
	"testing"

	"github.com/zhaori96/kr/pkg/kr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockKeyStorage struct {
	mock.Mock
}

func (m *MockKeyStorage) Get(ctx context.Context, id string) (*kr.Key, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*kr.Key), args.Error(1)
}

func (m *MockKeyStorage) Add(ctx context.Context, keys ...*kr.Key) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockKeyStorage) Delete(ctx context.Context, ids ...string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockKeyStorage) Erase(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestInMemoryKeyStorage(t *testing.T) {
	storage := kr.NewKeyStorage()

	storage.Add(context.Background(), &kr.Key{ID: "1", Value: "1"})
	storage.Add(context.Background(), &kr.Key{ID: "2", Value: "2"})
	storage.Add(context.Background(), &kr.Key{ID: "3", Value: "3"})

	t.Run("Get", func(t *testing.T) {
		key, err := storage.Get(context.Background(), "1")
		assert.NoError(t, err)
		assert.Equal(t, "1", key.Value)

		key, err = storage.Get(context.Background(), "2")
		assert.NoError(t, err)
		assert.Equal(t, "2", key.Value)

		key, err = storage.Get(context.Background(), "3")
		assert.NoError(t, err)
		assert.Equal(t, "3", key.Value)
	})

	t.Run("Delete", func(t *testing.T) {
		err := storage.Delete(context.Background(), "1")
		assert.NoError(t, err)
	})

	t.Run("Get after Delete", func(t *testing.T) {
		key, err := storage.Get(context.Background(), "1")
		assert.ErrorIs(t, err, kr.ErrKeyNotFound)
		assert.Nil(t, key)
	})

	t.Run("Erase", func(t *testing.T) {
		err := storage.Erase(context.Background())
		assert.NoError(t, err)
	})

	t.Run("Get after Erase", func(t *testing.T) {
		key, err := storage.Get(context.Background(), "1")
		assert.ErrorIs(t, err, kr.ErrKeyNotFound)
		assert.Nil(t, key)

		key, err = storage.Get(context.Background(), "2")
		assert.ErrorIs(t, err, kr.ErrKeyNotFound)
		assert.Nil(t, key)

		key, err = storage.Get(context.Background(), "3")
		assert.ErrorIs(t, err, kr.ErrKeyNotFound)
		assert.Nil(t, key)
	})
}
