package kr

import (
	"fmt"
	"time"
)

type RotatorHook func(rotator Rotator, activeKeyIds []*Key)

type RotatorHooks []RotatorHook

func (h RotatorHooks) Run(rotator Rotator, activeKeyIds []*Key) {
	for _, hook := range h {
		hook(rotator, activeKeyIds)
	}
}

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

const (
	DefaultRotationKeyCount int           = 5
	DefaultKeyExpiration    time.Duration = 12 * time.Hour
	DefaultRotationInterval time.Duration = 12 * time.Hour
)

type RotatorSettings struct {
	RotationKeyCount     int
	KeyExpiration        time.Duration
	RotationInterval     time.Duration
	AutoClearExpiredKeys bool
}

func DefaultRotatorSettings() *RotatorSettings {
	return &RotatorSettings{
		RotationKeyCount:     DefaultRotationKeyCount,
		KeyExpiration:        DefaultKeyExpiration,
		RotationInterval:     DefaultRotationInterval,
		AutoClearExpiredKeys: true,
	}
}

func (s *RotatorSettings) Validate() error {
	if s.RotationKeyCount < 1 {
		return fmt.Errorf(
			"%w: rotation key count must be greater than 0 (got %d)",
			ErrInvalidRotationKeyCount,
			s.RotationKeyCount,
		)
	}

	if s.RotationInterval < 0 {
		return fmt.Errorf(
			"%w: rotation interval must be greater than 0 (got %s)",
			ErrInvalidRotationInterval,
			s.RotationInterval,
		)
	}

	if s.KeyExpiration < 0 {
		return fmt.Errorf(
			"%w: key expiration must be greater than 0 (got %s)",
			ErrInvalidKeyExpiration,
			s.KeyExpiration,
		)
	}

	return nil
}

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
