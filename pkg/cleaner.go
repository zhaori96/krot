package kr

import (
	"context"
	"log"
	"sync"
	"time"
)

type KeyCleaner interface {
	Add(id string, expiration time.Time)
	Start(ctx context.Context)
	Stop()
}

type keyCleaner struct {
	ids         []string
	expirations []time.Time

	ctx        context.Context
	cancel     context.CancelFunc
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

func (c *keyCleaner) Add(id string, expiration time.Time) {
	c.ids = append(c.ids, id)
	c.expirations = append(c.expirations, expiration)

	c.keyExpired.Broadcast()
}

func (c *keyCleaner) Start(ctx context.Context) {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.ctx, c.cancel = context.WithCancel(ctx)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if len(c.ids) == 0 {
				c.keyExpired.Wait()
				continue
			}

			id := c.ids[0]
			expiration := c.expirations[0]
			if expiration.Before(time.Now()) {
				c.ids = c.ids[1:]
				c.expirations = c.expirations[1:]

				err := c.storage.Delete(ctx, id)
				if err != nil {
					log.Printf("failed to delete key %s: %v", id, err)
				}

				continue
			}
		}
	}
}

func (c *keyCleaner) Stop() {
	c.cancel()
}
