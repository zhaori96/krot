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

func (c *RotationController) Wait() {
	<-c.context.Done()
}

func (c *RotationController) WaitWithCallback(callback func()) {
	<-c.context.Done()
	callback()
}

func (c *RotationController) Dipose() {
	c.cancel()
}
