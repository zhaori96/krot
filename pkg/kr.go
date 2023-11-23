package kr

import (
	"context"
	"time"
)

type RotatorHook func(rotator Rotator, activeKeyIds []*Key)

type RotatorState uint

const (
	RotatorStateIdle RotatorState = iota
	RotatorStateRotating
)

type RotatorStatus uint

const (
	RotatorStatusInactive RotatorStatus = iota
	RotatorStatusActive
)

type Rotator interface {
	Status() RotatorStatus
	State() RotatorState
	RotationInterval() time.Duration
	Start() error
	Stop()
	Rotate(context context.Context) error
	GetKey(context context.Context) (string, any, error)
	GetKeyID(context context.Context) (string, error)
	GetKeyByID(context context.Context, id string) (any, error)
	WhenRotate(hooks ...RotatorHook)
}
