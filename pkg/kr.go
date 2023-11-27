package kr

import (
	"context"
	"crypto/rand"
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
	id string

	settings *RotatorSettings

	state  RotatorState
	status RotatorStatus

	controller *RotationController

	storage   KeyStorage
	generator KeyGenerator
	cleaner   KeyCleaner

	hooksBeforeRotation RotatorHooks
	hooksAfterRotation  RotatorHooks
}

func New() *Rotator {
	rotator := &Rotator{
		id: generateInstanceID(),
	}

	return rotator
}

func NewWithSettings(settings *RotatorSettings) (*Rotator, error) {
	rotator := &Rotator{
		id: generateInstanceID(),
	}

	if err := rotator.SetSettings(settings); err != nil {
		return nil, err
	}

	return rotator, nil
}

func generateInstanceID() string {
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		return ""
	}

	return fmt.Sprintf("kr#%x", id)
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

func (r *Rotator) Rotate() error {
	r.controller.Lock()
	defer r.controller.Unlock()

	keys := make([]*Key, 0, r.settings.RotationKeyCount)
	for i := 0; i < r.settings.RotationKeyCount; i++ {
		keyID := make([]byte, KeySize256)
		if _, err := rand.Read(keyID); err != nil {
			return err
		}

		keyValue, err := r.generator.Generate()
		if err != nil {
			return err
		}

		keyExpiration := time.Now().
			Add(r.settings.RotationInterval).
			Add(r.settings.KeyExpiration)

		r.cleaner.Add(string(keyID), keyExpiration)

		key := &Key{
			ID:      fmt.Sprintf("%s:%x", r.id, keyID),
			Value:   keyValue,
			Expires: keyExpiration,
		}

		keys = append(keys, key)
	}

	err := r.storage.Add(context.Background(), keys...)
	if err != nil {
		return err
	}

	return nil
}

func (r *Rotator) Start() error {
	if r.status == RotatorStatusActive {
		return ErrRotatorAlreadyRunning
	}

	if r.generator == nil {
		r.generator = NewKeyGenerator(KeySize256)
	}

	if r.storage == nil {
		r.storage = NewKeyStorage()
	}

	if r.settings == nil {
		r.settings = DefaultRotatorSettings()
	}

	r.controller = NewRotationController()
	r.cleaner = NewKeyCleaner(r.storage)

	r.status = RotatorStatusActive

	go r.run()
	go r.cleaner.Start(context.Background())

	return nil
}

func (r *Rotator) Stop() {
	if r.status == RotatorStatusInactive {
		return
	}

	r.controller.Dipose()
	r.cleaner.Stop()
	r.status = RotatorStatusInactive
}

func (r *Rotator) run() error {
	defer func() {
		r.state = RotatorStateIdle
	}()

	if err := r.Rotate(); err != nil {
		return err
	}

	for {
		if r.controller.Disposed() {
			return nil
		}

		time.Sleep(r.settings.RotationInterval)
		r.state = RotatorStateRotating
		if err := r.Rotate(); err != nil {
			return err
		}

		r.state = RotatorStateIdle
	}
}
