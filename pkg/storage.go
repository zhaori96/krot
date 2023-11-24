package kr

import (
	"context"
	"errors"
	"fmt"
)

type KeyStorage interface {
	Get(context context.Context, id string) (*Key, error)
	Set(context context.Context, keys ...*Key) error
	Delete(context context.Context, id string) error
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

func (s *inMemoryStorage) Get(context context.Context, id string) (*Key, error) {
	key, ok := s.keys[id]
	if !ok {
		return nil, errors.Join(ErrKeyNotFound, fmt.Errorf("key %s not found", id))
	}

	return key, nil
}

func (s *inMemoryStorage) Set(context context.Context, keys ...*Key) error {
	for _, key := range keys {
		if key == nil || key.ID == "" || key.Value == "" {
			continue
		}

		s.keys[key.ID] = key
	}

	return nil
}

func (s *inMemoryStorage) Delete(context context.Context, id string) error {
	delete(s.keys, id)
	return nil
}

func (s *inMemoryStorage) Erase(context context.Context) error {
	s.keys = make(map[string]*Key)
	return nil
}
