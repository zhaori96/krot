package kr

import (
	"context"
	"log"
	"sync"
	"time"
)

type KeyCleaner interface {
	Add(keys ...*Key)
	Start(ctx context.Context)
}

type keyCleaner struct {
	keys []*Key

	locker     *sync.Mutex
	keyExpired *sync.Cond

	storage KeyStorage
}

func NewKeyCleaner(storage KeyStorage) KeyCleaner {
	locker := &sync.Mutex{}
	keyExpired := sync.NewCond(locker)

	return &keyCleaner{
		locker:     locker,
		keyExpired: keyExpired,
		storage:    storage,
	}
}

func (c *keyCleaner) Add(keys ...*Key) {
	c.keys = append(c.keys, keys...)
}

func (c *keyCleaner) Start(ctx context.Context) {
	c.locker.Lock()
	defer c.locker.Unlock()

	for {
		if len(c.keys) == 0 {
			c.keyExpired.Wait()
			continue
		}

		key := c.keys[0]
		if key.Expires.Before(time.Now()) {
			c.keys = c.keys[1:]
			err := c.storage.Delete(ctx, key.ID)
			if err != nil {
				log.Printf("failed to delete key %s: %v", key.ID, err)
			}
			continue
		}
	}
}
