// Package kr provides a comprehensive system for managing key rotation in secure
// applications. It includes components for generating, storing, and cleaning keys,
// as well as managing key rotation and expiration.
//
// The key rotation system is controlled by a Rotator, which uses a KeyGenerator to
// create new keys, a KeyStorage to store and retrieve keys, and a KeyCleaner to
// remove expired keys. The Rotator can be started and stopped, and its operation
// can be customized with various settings.
//
// The package also defines a Key type, which represents a key with an ID, value,
// and expiration time, and a KeyRotatorError type, which represents an error in
// the key rotation process.
package kr

import (
	"context"
	"fmt"
	"time"

	cryptorand "crypto/rand"
	mathrand "math/rand"
)

var rotator *Rotator

func init() {
	rotator = New()
}

// RotatorHook is a function that is called before or after a rotation.
type RotatorHook func(rotator *Rotator)

// RotatorHooks is a collection of RotatorHook functions.
// It implements the Run method which runs all the hooks in the collection.
type RotatorHooks []RotatorHook

func (h RotatorHooks) Run(rotator *Rotator) {
	for _, hook := range h {
		hook(rotator)
	}
}

var (
	// EraseStorageHook is a predefined function that implements the RotatorHook interface.
	// When executed, it erases all keys from the associated Rotator's storage.
	// This is achieved by creating a new background context and invoking the Erase method of the Rotator's storage.
	// This hook is typically used when you want to clear all keys from the storage, such as during starting or stopping of the Rotator.
	//
	// Example usage:
	// 	rotator.OnStart(kr.EraseStorageHook)
	EraseStorageHook RotatorHook = func(rotator *Rotator) {
		rotator.storage.Erase(context.Background())
	}
)

// RotatorState is the state of the rotator.
type RotatorState uint

const (
	// RotatorStateIdle is the state of the rotator when it is not rotating.
	RotatorStateIdle RotatorState = iota

	// RotatorStateRotating is the state of the rotator when it is rotating.
	RotatorStateRotating
)

// RotatorStatus is the status of the rotator.
type RotatorStatus uint

const (
	// RotatorStatusInactive is the status of the rotator when it is not running.
	RotatorStatusInactive RotatorStatus = iota

	// RotatorStatusActive is the status of the rotator when it is running.
	RotatorStatusActive
)

const (
	// DefaultRotationKeyCount is the default number of keys to rotate.
	DefaultRotationKeyCount int = 5

	// DefaultKeyExpiration is the default expiration time for a key.
	// The key expiration is calculated as follows:
	//   key expiration = current time + rotation interval + key expiration
	DefaultKeyExpiration time.Duration = 12 * time.Hour

	// DefaultRotationInterval is the default interval between rotations.
	// The rotation interval is calculated as follows:
	//   rotation interval = current time + rotation interval
	DefaultRotationInterval time.Duration = 12 * time.Hour
)

// RotatorSettings is the settings for the rotator.
type RotatorSettings struct {
	// RotationKeyCount is the number of keys to rotate.
	// The default value is DefaultRotationKeyCount.
	// The minimum value is 1.
	RotationKeyCount int

	// KeyExpiration is the expiration time for a key.
	// The default value is DefaultKeyExpiration.
	KeyExpiration time.Duration

	// RotationInterval is the interval between rotations.
	// The default value is DefaultRotationInterval.
	RotationInterval time.Duration

	// AutoClearExpiredKeys is a flag that indicates whether to automatically clear expired keys.
	// The default value is true.
	AutoClearExpiredKeys bool
}

// DefaultRotatorSettings returns the default rotator settings.
func DefaultRotatorSettings() *RotatorSettings {
	return &RotatorSettings{
		RotationKeyCount:     DefaultRotationKeyCount,
		KeyExpiration:        DefaultKeyExpiration,
		RotationInterval:     DefaultRotationInterval,
		AutoClearExpiredKeys: true,
	}
}

// Validate validates the rotator settings.
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

// Rotator is a concurrent-safe key rotation manager.
// It generates and stores new keys at regular intervals while cleaning up expired keys.
// Suitable for rotating keys in encryption, decryption, signing, verification, and authentication.
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

	onStartHooks RotatorHooks
	onStopHooks  RotatorHooks

	hooksBeforeRotation RotatorHooks
	hooksAfterRotation  RotatorHooks
}

// New returns a newly initialized key rotator with default settings, storage, and key generator.
// Settings, storage and key generator can be set using the SetSettings, SetStorage, and SetGenerator methods.
func New() *Rotator {
	rotator := &Rotator{
		id: generateInstanceID(),
	}

	return rotator
}

// NewWithSettings returns a newly initialized key rotator with the given settings.
// Storage and key generator can be set using the SetStorage and SetGenerator methods.
func NewWithSettings(settings *RotatorSettings) (*Rotator, error) {
	rotator := &Rotator{
		id: generateInstanceID(),
	}

	if err := rotator.SetSettings(settings); err != nil {
		return nil, err
	}

	return rotator, nil
}

// GetRotator returns the global instance of the Rotator.
func GetRotator() *Rotator {
	return rotator
}

func generateInstanceID() string {
	id := make([]byte, 8)
	if _, err := cryptorand.Read(id); err != nil {
		return ""
	}

	return fmt.Sprintf("kr#%x", id)
}

// ID returns the unique identifier associated with the rotator.
// This identifier, generated upon creation, is used for storage and key generation.
func (r *Rotator) ID() string {
	return r.id
}

// ID returns the unique identifier associated with the rotator.
// This identifier, generated upon creation, is used for storage and key generation.
func ID() string { return rotator.ID() }

// Status returns the current operational status of the rotator, which is either active or inactive.
// The rotator is considered active while running and marked as inactive when not in operation.
func (r *Rotator) Status() RotatorStatus {
	return r.status
}

// Status returns the current operational status of the rotator, which is either active or inactive.
// The rotator is considered active while running and marked as inactive when not in operation.
func Status() RotatorStatus { return rotator.Status() }

func (r *Rotator) setStatus(status RotatorStatus) {
	r.status = status
}

// State returns the current state of the rotator, which is either idle or rotating.
// The rotator is considered idle when not rotating and marked as rotating when in operation.
func (r *Rotator) State() RotatorState {
	return r.state
}

// State returns the current state of the rotator, which is either idle or rotating.
// The rotator is considered idle when not rotating and marked as rotating when in operation.
func State() RotatorState { return rotator.State() }

func (r *Rotator) setState(state RotatorState) {
	r.state = state
}

// RotationKeyCount returns the RotationKeyCount field of the Rotator's settings.
// It indicates the number of keys the Rotator is configured to keep when rotating keys.
func (r *Rotator) RotationKeyCount() int {
	return r.settings.RotationKeyCount
}

// RotationKeyCount returns the RotationKeyCount field of the Rotator's settings.
// It indicates the number of keys the Rotator is configured to keep when rotating keys.
func RotationKeyCount() int { return rotator.RotationKeyCount() }

// KeyExpiration returns the KeyExpiration field of the Rotator's settings.
// It indicates the duration after which the keys generated by the Rotator are configured to expire.
func (r *Rotator) KeyExpiration() time.Duration {
	return r.settings.KeyExpiration
}

// KeyExpiration returns the KeyExpiration field of the Rotator's settings.
// It indicates the duration after which the keys generated by the Rotator are configured to expire.
func KeyExpiration() time.Duration { return rotator.KeyExpiration() }

// RotationInterval returns the RotationInterval field of the Rotator's settings.
// It indicates the duration after which the Rotator is configured to rotate keys.
func (r *Rotator) RotationInterval() time.Duration {
	return r.settings.RotationInterval
}

// RotationInterval returns the RotationInterval field of the Rotator's settings.
// It indicates the duration after which the Rotator is configured to rotate keys.
func RotationInterval() time.Duration { return rotator.RotationInterval() }

// AutoClearExpiredKeys returns the AutoClearExpiredKeys field of the Rotator's settings.
// It indicates whether the Rotator is configured to automatically clear expired keys.
func (r *Rotator) AutoClearExpiredKeys() bool {
	return r.settings.AutoClearExpiredKeys
}

// AutoClearExpiredKeys returns the AutoClearExpiredKeys field of the Rotator's settings.
// It indicates whether the Rotator is configured to automatically clear expired keys.
func AutoClearExpiredKeys() bool { return rotator.AutoClearExpiredKeys() }

// SetSettings sets the settings field of the Rotator struct.
// It accepts a RotatorSettings type as an argument and returns an error.
// If the Rotator is currently active (i.e., r.status == RotatorStatusActive),
// the method immediately panics.
// This is a safety measure to prevent changing the settings while the Rotator is in use.
// If the provided RotatorSettings is nil, or if the settings are invalid,
// the method returns an appropriate error.
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

// SetSettings sets the settings field of the Rotator struct.
// It accepts a RotatorSettings type as an argument and returns an error.
// If the Rotator is currently active (i.e., r.status == RotatorStatusActive),
// the method immediately panics.
// This is a safety measure to prevent changing the settings while the Rotator is in use.
// If the provided RotatorSettings is nil, or if the settings are invalid,
// the method returns an appropriate error.
func SetSettings(settings *RotatorSettings) error { return rotator.SetSettings(settings) }

// SetStorage sets the storage field of the Rotator struct.
// It accepts a KeyStorage type as an argument and returns an error.
// If the Rotator is currently active (i.e., r.status == RotatorStatusActive),
// the method immediately panics.
// This is a safety measure to prevent changing the storage while the Rotator is in use.
// If the provided KeyStorage is nil, the method returns an ErrInvalidArgument.
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

// SetStorage sets the storage field of the Rotator struct.
// It accepts a KeyStorage type as an argument and returns an error.
// If the Rotator is currently active (i.e., r.status == RotatorStatusActive),
// the method immediately panics.
// This is a safety measure to prevent changing the storage while the Rotator is in use.
// If the provided KeyStorage is nil, the method returns an ErrInvalidArgument.
func SetStorage(storage KeyStorage) error { return rotator.SetStorage(storage) }

// SetGenerator sets the generator field of the Rotator struct.
// It accepts a KeyGenerator type as an argument and returns an error.
// If the Rotator is currently active (i.e., r.status == RotatorStatusActive),
// the method immediately panics.
// This is a safety measure to prevent changing the generator while the Rotator is in use.
// If the provided KeyGenerator is nil, the method returns an ErrInvalidArgument.
func (r *Rotator) SetGenerator(generator KeyGenerator) error {
	if r.status == RotatorStatusActive {
		panic("cannot set generator while rotator is running")
	}

	if generator == nil {
		return fmt.Errorf("%w: generator cannot be nil", ErrInvalidArgument)
	}

	r.generator = generator
	return nil
}

// SetGenerator sets the generator field of the Rotator struct.
// It accepts a KeyGenerator type as an argument and returns an error.
// If the Rotator is currently active (i.e., r.status == RotatorStatusActive),
// the method immediately panics.
// This is a safety measure to prevent changing the generator while the Rotator is in use.
// If the provided KeyGenerator is nil, the method returns an ErrInvalidArgument.
func SetGenerator(generator KeyGenerator) error { return rotator.SetGenerator(generator) }

// OnStart appends provided hooks that can be called when the Rotator starts.
func (r *Rotator) OnStart(hooks ...RotatorHook) {
	r.onStartHooks = append(r.onStartHooks, hooks...)
}

// OnStart appends provided hooks that can be called when the Rotator starts.
func OnStart(hooks ...RotatorHook) { rotator.OnStart(hooks...) }

// OnStop appends provided hooks that can be called when the Rotator stops.
func (r *Rotator) OnStop(hooks ...RotatorHook) {
	r.onStopHooks = append(r.onStopHooks, hooks...)
}

// OnStop appends provided hooks that can be called when the Rotator stops.
func OnStop(hooks ...RotatorHook) { rotator.OnStop(hooks...) }

// BeforeRotation appends provided hooks to the beginning of the Rotator's hooksBeforeRotation slice.
// These hooks are executed before a rotation occurs.
func (r *Rotator) BeforeRotation(hooks ...RotatorHook) {
	r.hooksBeforeRotation = append(hooks, r.hooksBeforeRotation...)
}

// BeforeRotation appends provided hooks to the beginning of the Rotator's hooksBeforeRotation slice.
// These hooks are executed before a rotation occurs.
func BeforeRotation(hooks ...RotatorHook) { rotator.BeforeRotation(hooks...) }

// AfterRotation appends provided hooks to the end of the Rotator's hooksAfterRotation slice.
// These hooks are executed after a rotation occurs.
func (r *Rotator) AfterRotation(hooks ...RotatorHook) {
	r.hooksAfterRotation = append(r.hooksAfterRotation, hooks...)
}

// AfterRotation appends provided hooks to the end of the Rotator's hooksAfterRotation slice.
// These hooks are executed after a rotation occurs.
func AfterRotation(hooks ...RotatorHook) { rotator.AfterRotation(hooks...) }

// GetKeyID retrieves a random key ID from the Rotator.
// It returns the retrieved key ID and any error that occurred.
func (r *Rotator) GetKeyID() (string, error) {
	r.controller.Lock()
	defer r.controller.Unlock()

	return r.getRandomKeyID()
}

// GetKeyID retrieves a random key ID from the Rotator.
// It returns the retrieved key ID and any error that occurred.
func GetKeyID() (string, error) { return rotator.GetKeyID() }

// GetKeyByID retrieves a key from the Rotator's storage by its ID.
// It returns the retrieved key and any error that occurred.
func (r *Rotator) GetKeyByID(id string) (*Key, error) {
	r.controller.Lock()
	defer r.controller.Unlock()

	key, err := r.storage.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// GetKeyByID retrieves a key from the Rotator's storage by its ID.
// It returns the retrieved key and any error that occurred.
func GetKeyByID(id string) (*Key, error) { return rotator.GetKeyByID(id) }

// GetKey retrieves a random key from the Rotator's storage.
// It returns the retrieved key and any error that occurred.
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

// GetKey retrieves a random key from the Rotator's storage.
// It returns the retrieved key and any error that occurred.
func GetKey() (*Key, error) { return rotator.GetKey() }

// Rotate generates a new set of keys and stores them in the Rotator's storage.
// It first sets the Rotator's state to Rotating, runs any BeforeRotation hooks,
// and then generates and stores the new keys.
// After storing the keys, it runs any AfterRotation hooks and sets the state back to Idle.
// It returns any error that occurred during the process.
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

// Rotate generates a new set of keys and stores them in the Rotator's storage.
// It first sets the Rotator's state to Rotating, runs any BeforeRotation hooks,
// and then generates and stores the new keys.
// After storing the keys, it runs any AfterRotation hooks and sets the state back to Idle.
// It returns any error that occurred during the process.
func Rotate() error { return rotator.Rotate() }

// Start initiates the key rotation process. If components like the key generator,
// storage, rotation settings, rotation controller, or key cleaner are not set,
// they are initialized with default values. The Rotator's status is then set to
// active, and the rotation and cleaning processes are launched in separate goroutines.
//
// If the Rotator is already active when Start is called, it returns an
// ErrRotatorAlreadyRunning error. If an error occurs during the initial key rotation,
// the error is returned and the Rotator does not start.
//
// By default, the key generator is a KeyGenerator with a key size of 256 bits. The
// storage is a new KeyStorage instance, and the rotation settings are the
// DefaultRotatorSettings. These defaults are used if the corresponding components
// are not set before calling Start.
//
// This method is safe for concurrent use.
//
// Example:
//
//	rotator := NewRotator()
//	err := rotator.Start()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer rotator.Stop()
//
// If the Rotator starts successfully, Start returns nil.
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

	r.controller.TurnOn()

	r.cleaner = NewKeyCleaner(r.storage)
	r.cleaner.Start(context.Background())

	r.setStatus(RotatorStatusActive)

	if err := r.Rotate(); err != nil {
		return err
	}

	go r.run()
	r.onStartHooks.Run(r)

	return nil
}

// Start initiates the key rotation process. If components like the key generator,
// storage, rotation settings, rotation controller, or key cleaner are not set,
// they are initialized with default values. The Rotator's status is then set to
// active, and the rotation and cleaning processes are launched in separate goroutines.
//
// If the Rotator is already active when Start is called, it returns an
// ErrRotatorAlreadyRunning error. If an error occurs during the initial key rotation,
// the error is returned and the Rotator does not start.
//
// By default, the key generator is a KeyGenerator with a key size of 256 bits. The
// storage is a new KeyStorage instance, and the rotation settings are the
// DefaultRotatorSettings. These defaults are used if the corresponding components
// are not set before calling Start.
//
// This method is safe for concurrent use.
//
// Example:
//
//	rotator := NewRotator()
//	err := rotator.Start()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer rotator.Stop()
//
// If the Rotator starts successfully, Start returns nil.
func Start() error { return rotator.Start() }

// Stop halts the key rotation process. If the Rotator is already inactive, it
// immediately returns. Otherwise, it disposes the rotation controller, stops the
// key cleaner, and sets the Rotator's status to inactive.
//
// This method is safe to call even if the Rotator is already stopped or has not been
// started. It ensures that the key rotation and cleaning processes are properly
// terminated.
//
// Example:
//
//	rotator := NewRotator()
//	err := rotator.Start()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// ... use the rotator ...
//	rotator.Stop()
//
// After calling Stop, the Rotator can be restarted with the Start method.
func (r *Rotator) Stop() {
	if r.status == RotatorStatusInactive {
		return
	}

	r.controller.TurnOff()
	r.cleaner.Stop()
	r.setStatus(RotatorStatusInactive)

	r.onStopHooks.Run(r)
}

// Stop halts the key rotation process. If the Rotator is already inactive, it
// immediately returns. Otherwise, it disposes the rotation controller, stops the
// key cleaner, and sets the Rotator's status to inactive.
//
// This method is safe to call even if the Rotator is already stopped or has not been
// started. It ensures that the key rotation and cleaning processes are properly
// terminated.
//
// Example:
//
//	rotator := NewRotator()
//	err := rotator.Start()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// ... use the rotator ...
//	rotator.Stop()
//
// After calling Stop, the Rotator can be restarted with the Start method.
func Stop() { rotator.Stop() }

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
