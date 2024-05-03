package krot

import (
	"context"
	"fmt"
	"time"
)

// KeyCleanerHook defines the signature for key cleaner hooks.
type KeyCleanerHook func(cleaner KeyCleaner)

// KeyCleanerHooks defines a collection of key cleaner hooks.
type KeyCleanerHooks []KeyCleanerHook

// Run executes the key cleaner hooks.
func (h KeyCleanerHooks) Run(cleaner KeyCleaner) {
	for _, hook := range h {
		hook(cleaner)
	}
}

// KeyCleanerState defines the state of the key cleaner.
type KeyCleanerState uint

const (
	// KeyCleanerStateIdle represents the idle state of the cleaner.
	KeyCleanerStateIdle KeyCleanerState = iota

	// KeyCleanerStateCleaning represents the cleaning state of the cleaner.
	KeyCleanerStateCleaning
)

// KeyCleanerStatus defines the status of the key cleaner.
type KeyCleanerStatus uint

const (
	// KeyCleanerStatusStopped represents the stopped status of the cleaner.
	KeyCleanerStatusStopped KeyCleanerStatus = iota

	// KeyCleanerStatusStarted represents the started status of the cleaner.
	KeyCleanerStatusStarted
)

type KeyCleaner interface {
	// State returns the current state of the cleaner.
	State() KeyCleanerState

	// Status returns the current status of the cleaner.
	Status() KeyCleanerStatus

	// OnStart registers hooks to be executed when the cleaner starts.
	OnStart(hooks ...KeyCleanerHook)

	// OnStop registers hooks to be executed when the cleaner stops.
	OnStop(hooks ...KeyCleanerHook)

	// BeforeCleaning registers hooks to be executed before the cleaner starts cleaning.
	BeforeCleaning(hooks ...KeyCleanerHook)

	// AfterCleaning registers hooks to be executed after the cleaner finishes cleaning.
	AfterCleaning(hooks ...KeyCleanerHook)

	// Start begins the key cleaning process. It requires a context for managing
	Start(ctx context.Context, interval time.Duration) error

	// Stop halts the key cleaning process.
	Stop()
}

type keyCleaner struct {
	status KeyCleanerStatus
	state  KeyCleanerState

	onStartHooks KeyCleanerHooks
	onStopHooks  KeyCleanerHooks

	beforeCleaningHooks KeyCleanerHooks
	afterCleaningHooks  KeyCleanerHooks

	ctx    context.Context
	cancel context.CancelFunc

	storage KeyStorage
}

func NewKeyCleaner(storage KeyStorage) KeyCleaner {
	return &keyCleaner{storage: storage}
}

func (c *keyCleaner) State() KeyCleanerState {
	return c.state
}

func (c *keyCleaner) Status() KeyCleanerStatus {
	return c.status
}

func (c *keyCleaner) OnStart(hooks ...KeyCleanerHook) {
	c.onStartHooks = append(c.onStartHooks, hooks...)
}

func (c *keyCleaner) OnStop(hooks ...KeyCleanerHook) {
	c.onStopHooks = append(c.onStopHooks, hooks...)
}

func (c *keyCleaner) BeforeCleaning(hooks ...KeyCleanerHook) {
	c.beforeCleaningHooks = append(c.beforeCleaningHooks, hooks...)
}

func (c *keyCleaner) AfterCleaning(hooks ...KeyCleanerHook) {
	c.afterCleaningHooks = append(c.afterCleaningHooks, hooks...)
}

func (c *keyCleaner) Start(ctx context.Context, interval time.Duration) error {
	if c.status == KeyCleanerStatusStarted {
		return fmt.Errorf("cleaner is already running")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.status = KeyCleanerStatusStarted
	go c.run(ctx, interval)

	c.onStartHooks.Run(c)
	return nil
}

func (c *keyCleaner) Stop() {
	if c.cancel != nil {
		c.status = KeyCleanerStatusStopped
		c.cancel()
		c.onStopHooks.Run(c)
	}
}

func (c *keyCleaner) run(context context.Context, interval time.Duration) {
	for {
		select {
		case <-c.ctx.Done():
			return

		default:
			c.state = KeyCleanerStateIdle
			time.Sleep(interval)

			c.beforeCleaningHooks.Run(c)
			c.state = KeyCleanerStateCleaning
			c.storage.ClearDeprecated(context)
			c.afterCleaningHooks.Run(c)
		}
	}
}
