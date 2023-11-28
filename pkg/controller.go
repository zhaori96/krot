package kr

import (
	"context"
	"sync"
)

type RotationController struct {
	context context.Context
	cancel  context.CancelFunc
	mutex   sync.Mutex
}

func NewRotationController() *RotationController {
	ctx, cancel := context.WithCancel(context.Background())
	return &RotationController{
		context: ctx,
		cancel:  cancel,
	}
}

func (c *RotationController) Context() context.Context {
	return c.context
}

func (c *RotationController) Lock() {
	c.mutex.Lock()
}

func (c *RotationController) Unlock() {
	c.mutex.Unlock()
}

func (c *RotationController) Disposed() bool {
	select {
	case <-c.context.Done():
		return true
	default:
		return false
	}
}

func (c *RotationController) Dipose() {
	c.cancel()
}
