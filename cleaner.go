package krot

import (
	"context"
	"log"
	"sync"
	"time"
)

// KeyCleaner defines the interface for managing key expiration. It provides
// methods for adding keys with expiration times, starting the cleaner, and
// stopping the cleaner.
type KeyCleaner interface {
	// Add adds a key with the specified ID and expiration time to the cleaner.
	//
	//     cleaner.Add("keyID", time.Now().Add(24 * time.Hour))
	Add(id string, expiration time.Time)

	// Start begins the key cleaning process. It requires a context for managing
	// timeouts and cancellations.
	//
	//     ctx := context.Background()
	//     cleaner.Start(ctx)
	Start(ctx context.Context)

	// Stop halts the key cleaning process.
	//
	//     cleaner.Stop()
	Stop()
}

type keyCleaner struct {
	ids         []string
	expirations []time.Time

	ctx         context.Context
	cancel      context.CancelFunc
	locker      *sync.Mutex
	newKeyAdded *sync.Cond

	storage KeyStorage
}

func NewKeyCleaner(storage KeyStorage) KeyCleaner {
	locker := &sync.Mutex{}
	keyExpired := sync.NewCond(locker)

	return &keyCleaner{
		locker:      locker,
		newKeyAdded: keyExpired,
		storage:     storage,
	}
}

func (c *keyCleaner) Add(id string, expiration time.Time) {
	if expiration.Before(time.Now()) {
		c.storage.Delete(context.Background(), id)
		return
	}

	c.ids = append(c.ids, id)
	c.expirations = append(c.expirations, expiration)

	if len(c.ids) == 1 {
		c.newKeyAdded.Signal()
	}
}

func (c *keyCleaner) Start(ctx context.Context) {
	c.ctx, c.cancel = context.WithCancel(ctx)
	go c.run()
}

func (c *keyCleaner) run() {
	c.locker.Lock()
	defer c.locker.Unlock()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if len(c.ids) == 0 {
				c.newKeyAdded.Wait()
				continue
			}

			expiration := c.expirations[0]
			if expiration.Before(time.Now()) {
				c.deleteLatestExpiredKey()
				continue
			}

			timer := time.NewTimer(time.Until(expiration))
			<-timer.C

			select {
			case <-c.ctx.Done():
				return
			default:
				c.deleteLatestExpiredKey()
			}
		}
	}
}

func (c *keyCleaner) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *keyCleaner) deleteLatestExpiredKey() {
	id := c.ids[0]

	c.ids = c.ids[1:]
	c.expirations = c.expirations[1:]

	err := c.storage.Delete(c.ctx, id)
	if err != nil {
		log.Printf("failed to delete key %s: %v", id, err)
	}
}
