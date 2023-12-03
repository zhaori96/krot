package kr

import (
	"context"
	"sync"
)

type RotationController struct {
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.Mutex
}

func NewRotationController() *RotationController {
	controller := &RotationController{}
	controller.TurnOn()

	return controller
}

func (c *RotationController) Context() context.Context {
	return c.ctx
}

func (c *RotationController) Lock() {
	c.mutex.Lock()
}

func (c *RotationController) Unlock() {
	c.mutex.Unlock()
}

func (c *RotationController) Disposed() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

func (c *RotationController) TurnOn() {
	c.ctx, c.cancel = context.WithCancel(context.Background())
}

func (c *RotationController) TurnOff() {
	c.cancel()
}
