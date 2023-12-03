package kr_test

import (
	"context"
	"testing"
	"time"

	"github.com/zhaori96/kr"

	"github.com/stretchr/testify/mock"
)

type MockKeyCleaner struct {
	mock.Mock
}

func (m *MockKeyCleaner) Add(id string, expiration time.Time) {
	m.Called(id, expiration)
}

func (m *MockKeyCleaner) Start(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockKeyCleaner) Stop() {
	m.Called()
}

// Should group the tests in nested subtests
func TestKeyCleaner(t *testing.T) {
	t.Run("Should call storage.Delete if expiration is before now", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Add("1", time.Now().Add(-1*time.Second))

		storage.AssertCalled(t, "Delete", mock.Anything, []string{"1"})
	})

	t.Run("Should call storage.Delete when a key is expired", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(1*time.Second))

		time.Sleep(2 * time.Second)

		storage.AssertCalled(t, "Delete", mock.Anything, []string{"1"})
	})

	t.Run("Should not call storage.Delete when a key is not expired", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(2*time.Second))

		time.Sleep(1 * time.Second)

		storage.AssertNotCalled(t, "Delete", mock.Anything, []string{"1"})
	})

	t.Run("Should not call storage.Delete when a key is deleted", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(1*time.Second))
		cleaner.Add("1", time.Now().Add(2*time.Second))

		time.Sleep(1 * time.Second)

		storage.AssertNotCalled(t, "Delete", mock.Anything, []string{"1"})
	})

	t.Run("Should call storage.Delete when a cleanered key is re-added", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)
		storage.On("Delete", mock.Anything, []string{"2"}).Return(nil)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(time.Second))
		cleaner.Add("2", time.Now().Add(2*time.Second))

		time.Sleep(2 * time.Second)

		cleaner.Add("1", time.Now().Add(time.Second))

		time.Sleep(2 * time.Second)

		storage.AssertNumberOfCalls(t, "Delete", 3)
		storage.AssertCalled(t, "Delete", mock.Anything, []string{"1"})
		storage.AssertCalled(t, "Delete", mock.Anything, []string{"2"})
	})

	t.Run("Should call storage.Delete when a cleanered key is re-added with a new expiration", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)
		storage.On("Delete", mock.Anything, []string{"2"}).Return(nil)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(1*time.Second))
		cleaner.Add("2", time.Now().Add(2*time.Second))

		time.Sleep(2 * time.Second)

		cleaner.Add("1", time.Now().Add(3*time.Second))

		time.Sleep(4 * time.Second)

		storage.AssertNumberOfCalls(t, "Delete", 3)
		storage.AssertCalled(t, "Delete", mock.Anything, []string{"1"})
	})

	t.Run("Call storage.Get should return ErrKeyNotFound after a key is cleaned", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)
		storage.On("Get", mock.Anything, "1").Return(nil, kr.ErrKeyNotFound)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(1*time.Second))

		time.Sleep(2 * time.Second)

		storage.Get(context.Background(), "1")
		storage.AssertExpectations(t)
	})

	t.Run("Call storage.Get should return ErrKeyNotFound after a key is cleaned and re-added", func(t *testing.T) {
		storage := &MockKeyStorage{}
		storage.On("Delete", mock.Anything, []string{"1"}).Return(nil)
		storage.On("Get", mock.Anything, "1").Return(nil, kr.ErrKeyNotFound)

		cleaner := kr.NewKeyCleaner(storage)
		cleaner.Start(context.Background())

		cleaner.Add("1", time.Now().Add(1*time.Second))

		time.Sleep(2 * time.Second)

		cleaner.Add("1", time.Now().Add(2*time.Second))

		time.Sleep(2 * time.Second)

		storage.Get(context.Background(), "1")
		storage.AssertExpectations(t)
	})
}
