package krot

import (
	"context"
	"errors"
	"fmt"
)

// KeyStorage defines the interface for key storage operations. It provides methods
// for getting, adding, deleting, and erasing keys.
type KeyStorage interface {
	// Get retrieves a key with the specified ID from the storage. If the key is not
	// found, it returns an error.
	//
	//     key, err := storage.Get(ctx, "keyID")
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	Get(context context.Context, id string) (*Key, error)

	// Add adds one or more keys to the storage. If a key cannot be added, it returns
	// an error.
	//
	//     err := storage.Add(ctx, key1, key2)
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	Add(context context.Context, keys ...*Key) error

	// Delete removes a key with the specified ID from the storage. If the key cannot
	// be deleted, it returns an error.
	//
	//     err := storage.Delete(ctx, "keyID")
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	Delete(context context.Context, ids ...string) error

	// ClearDeprecated iterates over the keys in the storage and removes any keys that are either nil or expired.
	// It returns an error if any issues occur during the operation.
	//
	//     err := storage.ClearDeprecated(ctx)
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	ClearDeprecated(context context.Context) error

	// Erase removes all keys from the storage. If the keys cannot be erased, it
	// returns an error.
	//
	//     err := storage.Erase(ctx)
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	Erase(context context.Context) error
}

type inMemoryStorage struct {
	keys map[string]*Key
}

func NewKeyStorage() KeyStorage {
	return &inMemoryStorage{
		keys: make(map[string]*Key),
	}
}

func (s *inMemoryStorage) Get(_ context.Context, id string) (*Key, error) {
	key, ok := s.keys[id]
	if !ok {
		return nil, errors.Join(ErrKeyNotFound, fmt.Errorf("key %s not found", id))
	}

	return key, nil
}

func (s *inMemoryStorage) Add(_ context.Context, keys ...*Key) error {
	for _, key := range keys {
		if key == nil || key.ID == "" || key.Value == "" {
			continue
		}

		s.keys[key.ID] = key
	}

	return nil
}

func (s *inMemoryStorage) Delete(_ context.Context, ids ...string) error {
	for _, keyID := range ids {
		delete(s.keys, keyID)
	}

	return nil
}

func (s *inMemoryStorage) ClearDeprecated(_ context.Context) error {
	for key, value := range s.keys {
		if value == nil || value.Expired() {
			delete(s.keys, key)
		}
	}

	return nil
}

func (s *inMemoryStorage) Erase(_ context.Context) error {
	s.keys = make(map[string]*Key)
	return nil
}
