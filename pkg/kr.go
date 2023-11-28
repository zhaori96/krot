package kr

import (
	"context"
	"fmt"
	"time"

	cryptorand "crypto/rand"
	mathrand "math/rand"
)

type RotatorHook func(rotator *Rotator)

type RotatorHooks []RotatorHook

func (h RotatorHooks) Run(rotator *Rotator) {
	for _, hook := range h {
		hook(rotator)
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

	lastGeneratedKeyIDs []string

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
	if _, err := cryptorand.Read(id); err != nil {
		return ""
	}

	return fmt.Sprintf("kr#%x", id)
}

func (r *Rotator) ID() string {
	return r.id
}

func (r *Rotator) Status() RotatorStatus {
	return r.status
}

func (r *Rotator) setStatus(status RotatorStatus) {
	r.status = status
}

func (r *Rotator) State() RotatorState {
	return r.state
}

func (r *Rotator) setState(state RotatorState) {
	r.state = state
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

func (r *Rotator) SetSettings(settings *RotatorSettings) error {
	if r.status == RotatorStatusActive {
		panic("cannot set settings while rotator is running")
	}

	if settings == nil {
		return fmt.Errorf("%w: settings cannot be nil", ErrInvalidArgument)
	}

	if err := settings.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSettings, err)
	}

	r.settings = settings
	return nil
}

func (r *Rotator) SetStorage(storage KeyStorage) error {
	if r.status == RotatorStatusActive {
		panic("cannot set storage while rotator is running")
	}

	if storage == nil {
		return fmt.Errorf("%w: storage cannot be nil", ErrInvalidArgument)
	}

	r.storage = storage
	return nil
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

func (r *Rotator) GetKeyID() (string, error) {
	r.controller.Lock()
	defer r.controller.Unlock()

	return r.getRandomKeyID()
}

func (r *Rotator) GetKeyByID(id string) (*Key, error) {
	r.controller.Lock()
	defer r.controller.Unlock()

	key, err := r.storage.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (r *Rotator) GetKey() (*Key, error) {
	r.controller.Lock()
	defer r.controller.Unlock()

	id, err := r.getRandomKeyID()
	if err != nil {
		return nil, err
	}

	key, err := r.storage.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (r *Rotator) Rotate() error {
	r.controller.Lock()
	defer r.controller.Unlock()

	r.setState(RotatorStateRotating)
	defer r.setState(RotatorStateIdle)

	r.hooksBeforeRotation.Run(r)

	r.lastGeneratedKeyIDs = make([]string, r.settings.RotationKeyCount)
	keys := make([]*Key, r.settings.RotationKeyCount)
	for i := 0; i < r.settings.RotationKeyCount; i++ {
		keyID := make([]byte, KeySize256)
		if _, err := cryptorand.Read(keyID); err != nil {
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

		keys[i] = key
		r.lastGeneratedKeyIDs[i] = key.ID
	}

	err := r.storage.Add(context.Background(), keys...)
	if err != nil {
		return err
	}

	r.hooksAfterRotation.Run(r)
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

	r.setStatus(RotatorStatusActive)

	if err := r.Rotate(); err != nil {
		return err
	}

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
	r.setStatus(RotatorStatusInactive)
}

func (r *Rotator) run() error {
	for {
		if r.controller.Disposed() {
			return nil
		}

		time.Sleep(r.settings.RotationInterval)

		if err := r.Rotate(); err != nil {
			return err
		}
	}
}

func (r *Rotator) getRandomKeyID() (string, error) {
	if len(r.lastGeneratedKeyIDs) == 0 {
		return "", ErrNoKeysGenerated
	}

	id := r.lastGeneratedKeyIDs[mathrand.Intn(len(r.lastGeneratedKeyIDs))]
	return id, nil
}
