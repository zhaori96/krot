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

type Rotator struct {
	settings *RotatorSettings

	state  RotatorState
	status RotatorStatus

	controller *RotationController

	storage   KeyStorage
	generator KeyGenerator

	hooksBeforeRotation RotatorHooks
	hooksAfterRotation  RotatorHooks
}

func New() *Rotator {
	return &Rotator{
		controller: NewRotationController(),
	}
}

func NewWithSettings(settings *RotatorSettings) (*Rotator, error) {
	rotator := &Rotator{controller: NewRotationController()}
	if err := rotator.SetSettings(settings); err != nil {
		return nil, err
	}

	return rotator, nil
}

func (r *Rotator) Status() RotatorStatus {
	return r.status
}

func (r *Rotator) State() RotatorState {
	return r.state
}

func (r *Rotator) SetSettings(settings *RotatorSettings) error {
	if settings == nil {
		return fmt.Errorf("%w: settings cannot be nil", ErrInvalidSettings)
	}

	if err := settings.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSettings, err)
	}

	r.settings = settings

	return nil
}

func (r *Rotator) RotationKeyCount() int {
	return r.settings.RotationKeyCount
}

func (r *Rotator) KeyExpiration() time.Duration {
	return r.settings.KeyExpiration
}

func (r *Rotator) RotationInterval() time.Duration {
	return r.settings.RotationInterval
}

func (r *Rotator) AutoClearExpiredKeys() bool {
	return r.settings.AutoClearExpiredKeys
}

func (r *Rotator) SetStorage(storage KeyStorage) {
	if r.status == RotatorStatusActive {
		panic("cannot set storage while rotator is running")
	}

	r.storage = storage
}

func (r *Rotator) SetGenerator(generator KeyGenerator) {
	if r.status == RotatorStatusActive {
		panic("cannot set generator while rotator is running")
	}

	if generator == nil {
		panic("generator cannot be nil")
	}

	r.generator = generator
}

func (r *Rotator) BeforeRotation(hooks ...RotatorHook) {
	r.hooksBeforeRotation = append(hooks, r.hooksBeforeRotation...)
}

func (r *Rotator) AfterRotation(hooks ...RotatorHook) {
	r.hooksAfterRotation = append(r.hooksAfterRotation, hooks...)
}
