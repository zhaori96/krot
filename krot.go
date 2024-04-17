// package krot provides a comprehensive system for managing key rotation in secure
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
package krot

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
	// 	rotator.OnStart(krot.EraseStorageHook)
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
	// RotatorStatusStopped is the status of the rotator when it is not running.
	RotatorStatusStopped RotatorStatus = iota

	// RotatorStatusStarted is the status of the rotator when it is running.
	RotatorStatusStarted
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

	// KeyProvidingMode is the strategy used for providing keys.
	// The default value is AutoKeyProvidingMode.
	KeyProvidingMode KeyProvidingMode
}

// DefaultRotatorSettings returns the default rotator settings.
func DefaultRotatorSettings() *RotatorSettings {
	return &RotatorSettings{
		RotationKeyCount:     DefaultRotationKeyCount,
		KeyExpiration:        DefaultKeyExpiration,
		RotationInterval:     DefaultRotationInterval,
		AutoClearExpiredKeys: true,
		KeyProvidingMode:     AutoKeyProvidingMode,
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

	if s.KeyProvidingMode < AutoKeyProvidingMode ||
		s.KeyProvidingMode > NonRepeatingCyclicKeyProvidingMode {
		return fmt.Errorf(
			"%w: key providing mode must be between %d and %d (got %d)",
			ErrInvalidKeyProvidingMode,
			AutoKeyProvidingMode,
			NonRepeatingCyclicKeyProvidingMode,
			s.KeyProvidingMode,
		)
	}

	return nil
}

// KeyProvidingMode represents the strategy used for providing keys.
type KeyProvidingMode int

const (
	// AutoKeyProvidingMode: This mode automatically selects the key providing
	// strategy based on the number of keys:
	// 	- Single key: Always returns the single available key.
	// 	- Two to five keys: Uses NonRepeatingKeyProvidingMode.
	// 	- More than five keys: Uses NonRepeatingCyclicKeyProvidingMode.
	AutoKeyProvidingMode KeyProvidingMode = iota

	// RandomKeyProvidingMode: This mode randomly selects a key from the
	// available keys. The same key can be selected multiple times in a row.
	RandomKeyProvidingMode

	// NonRepeatingKeyProvidingMode: This mode randomly selects a key from the
	// available keys, but ensures that the same key is not selected twice in a row.
	NonRepeatingKeyProvidingMode

	// CyclicKeyProvidingMode: This mode cycles through the keys in order,
	// starting from the first key and returning to the first key after the last key.
	CyclicKeyProvidingMode

	// NonRepeatingCyclicKeyProvidingMode: This mode cycles through the keys in
	// order, but ensures that the same key is not selected twice in a row.
	// After all keys have been selected, it starts a new cycle.
	NonRepeatingCyclicKeyProvidingMode
)

// KeyIDProvider manages the provision of keys (IDs) based on a specified
// strategy.
type KeyIDProvider struct {
	mode KeyProvidingMode

	ids               []string
	availableIndexes  []int
	lastSelectedIndex int
	round             int
}

// NewKeyIDProvider returns a new KeyIDProvider with the specified mode and IDs.
func NewKeyIDProvider(mode KeyProvidingMode, ids ...string) *KeyIDProvider {
	provider := &KeyIDProvider{mode: mode}
	provider.Set(ids...)

	return provider
}

// Set replaces the existing IDs in the KeyIDProvider with the provided IDs.
// Note that any previous IDs are lost when this method is called.
func (s *KeyIDProvider) Set(ids ...string) {
	s.ids = ids
	s.reloadAvailableIndexes()

	if s.mode == AutoKeyProvidingMode {
		switch len(s.ids) {
		case 1:
			s.mode = RandomKeyProvidingMode

		case 2, 3, 4, 5:
			s.mode = NonRepeatingKeyProvidingMode

		default:
			s.mode = NonRepeatingCyclicKeyProvidingMode
		}
	}
}

// Get returns an ID based on the current KeyProvidingMode. If no IDs are
// available, it returns an error. The behavior varies depending on the mode:
func (i *KeyIDProvider) Get() (string, error) {
	if len(i.ids) == 0 {
		return "", ErrNoKeysGenerated
	}

	if len(i.ids) == 1 {
		return i.ids[0], nil
	}

	switch i.mode {
	case RandomKeyProvidingMode:
		return i.ids[mathrand.Intn(len(i.ids))], nil

	case NonRepeatingKeyProvidingMode:
		index := mathrand.Intn(len(i.ids) - 1)
		if index >= i.lastSelectedIndex {
			index++
		}

		i.lastSelectedIndex = index
		return i.ids[index], nil

	case CyclicKeyProvidingMode:
		if i.round == len(i.ids) {
			i.round = 0
		}

		id := i.ids[i.round]
		i.round++

		return id, nil

	case NonRepeatingCyclicKeyProvidingMode:
		if len(i.availableIndexes) == 0 {
			i.reloadAvailableIndexes()
		}

		var index int
		if len(i.availableIndexes) > 1 {
			index = mathrand.Intn(len(i.availableIndexes) - 1)
			if i.availableIndexes[index] == i.lastSelectedIndex {
				index = len(i.availableIndexes) - 1
			}
		} else {
			index = 0
		}

		i.lastSelectedIndex = i.availableIndexes[index]
		id := i.ids[i.lastSelectedIndex]

		i.availableIndexes[index] = i.availableIndexes[len(i.availableIndexes)-1]
		i.availableIndexes[len(i.availableIndexes)-1] = i.lastSelectedIndex

		i.availableIndexes = i.availableIndexes[:len(i.availableIndexes)-1]

		return id, nil

	default:
		return "", fmt.Errorf("%w: %d", ErrInvalidKeyProvidingMode, i.mode)
	}
}

func (i *KeyIDProvider) reloadAvailableIndexes() {
	i.availableIndexes = make([]int, len(i.ids))
	for index := range i.ids {
		i.availableIndexes[index] = index
	}
}

type rotatorContextKey struct {
	alias string
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

	storage    KeyStorage
	generator  KeyGenerator
	idProvider KeyIDProvider
	cleaner    KeyCleaner

	onStartHooks RotatorHooks
	onStopHooks  RotatorHooks

	hooksBeforeRotation RotatorHooks
	hooksAfterRotation  RotatorHooks
}

// New returns a newly initialized key rotator with default settings, storage, and key generator.
// Settings, storage and key generator can be set using the SetSettings, SetStorage, and SetGenerator methods.
func New() *Rotator {
	rotator := &Rotator{
		id:         generateInstanceID(),
		generator:  NewKeyGenerator(KeySize256),
		storage:    NewKeyStorage(),
		settings:   DefaultRotatorSettings(),
		controller: NewRotationController(),
	}

	rotator.cleaner = NewKeyCleaner(rotator.storage)

	return rotator
}

// NewWithSettings returns a newly initialized key rotator with the given settings.
// Storage and key generator can be set using the SetStorage and SetGenerator methods.
func NewWithSettings(settings *RotatorSettings) (*Rotator, error) {
	rotator := &Rotator{
		id:         generateInstanceID(),
		generator:  NewKeyGenerator(KeySize256),
		storage:    NewKeyStorage(),
		controller: NewRotationController(),
	}

	if err := rotator.SetSettings(settings); err != nil {
		return nil, err
	}

	rotator.cleaner = NewKeyCleaner(rotator.storage)

	return rotator, nil
}

// NewWithContext initializes a key rotator with default settings, storage, and key generator.
// It returns the rotator and a context that has the rotator associated with it.
// You can customize the rotator by using SetSettings, SetStorage, and SetGenerator methods.
//
// The rotator can be retrieved from the context using FromContext method.
//
// If an alias is provided, it's used to associate the rotator with the context.
// This allows for multiple rotators to be associated with a single context.
//
// If no alias is provided, the rotator is associated with the context using the default alias "default".
//
// Example without alias:
// 	rotator, ctx := krot.NewWithContext(context.Background())
// 	krot.FromContext(ctx).Start() // Starts the default rotator
//
// Example with alias:
// 	rotator, ctx := krot.NewWithContext(context.Background(), "my-rotator")
// 	krot.FromContext(ctx, "my-rotator").Start() // Starts the rotator with the alias "my-rotator"
func NewWithContext(ctx context.Context, alias ...string) (*Rotator, context.Context) {
	key := rotatorContextKey{}

	if len(alias) > 0 {
		key.alias = alias[0]
	} else {
		key.alias = "default"
	}

	rotator := New()
	ctx = context.WithValue(ctx, key, rotator)

	return rotator, ctx
}

// FromContext retrieves the Rotator linked with the given context.
// If no alias is specified, it returns the Rotator associated with the default alias "default".
// If an alias is provided, it returns the Rotator associated with that specific alias.
// If no Rotator is linked with the context (or the specified alias), it returns nil.
//
// Example wihtout alias:
// 	rotator, ctx := krot.NewWithContext(context.Background())
// 	krot.FromContext(ctx).Start() // Starts the default rotator
//
// Example with alias:
// 	rotator, ctx := krot.NewWithContext(context.Background(), "my-rotator")
// 	krot.FromContext(ctx, "my-rotator").Start() // Starts the rotator with the alias "my-rotator"
func FromContext(ctx context.Context, alias ...string) *Rotator {
	key := rotatorContextKey{}

	if len(alias) > 0 {
		key.alias = alias[0]
	} else{
		key.alias = "default"
	}

	rotator, _ := ctx.Value(key).(*Rotator)
	return rotator
}

// GetRotator returns the global instance of the Rotator.
func GetRotator() *Rotator {
	return rotator
}

func generateInstanceID() string {
	id := make([]byte, KeySize64)
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
	if r.status == RotatorStatusStarted {
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
	if r.status == RotatorStatusStarted {
		panic("cannot set storage while rotator is running")
	}

	if storage == nil {
		return fmt.Errorf("%w: storage cannot be nil", ErrInvalidArgument)
	}

	r.cleaner.Stop()
	r.cleaner = NewKeyCleaner(storage)

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
	if r.status == RotatorStatusStarted {
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

	return r.idProvider.Get()
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

	id, err := r.idProvider.Get()
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
	r.hooksBeforeRotation.Run(r)

	var err error
	defer func() {
		if err == nil {
			r.hooksAfterRotation.Run(r)
		}
	}()

	r.controller.Lock()
	defer r.controller.Unlock()

	r.setState(RotatorStateRotating)
	defer r.setState(RotatorStateIdle)

	ids := make([]string, r.settings.RotationKeyCount)
	keys := make([]*Key, r.settings.RotationKeyCount)
	for i := 0; i < r.settings.RotationKeyCount; i++ {
		keyID := make([]byte, KeySize64)
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
		ids[i] = key.ID
	}

	err = r.storage.Add(context.Background(), keys...)
	if err != nil {
		return err
	}

	r.idProvider.Set(ids...)
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
	if r.status == RotatorStatusStarted {
		return ErrRotatorAlreadyRunning
	}

	if r.settings.AutoClearExpiredKeys {
		r.cleaner.Start(context.Background())
	}

	r.controller.TurnOn()
	if err := r.Rotate(); err != nil {
		return err
	}

	go r.run()

	r.setStatus(RotatorStatusStarted)
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
	if r.status == RotatorStatusStopped {
		return
	}

	r.controller.TurnOff()
	r.cleaner.Stop()
	r.setStatus(RotatorStatusStopped)

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
