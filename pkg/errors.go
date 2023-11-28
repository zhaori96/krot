package kr

// KeyRotatorError represents an error in the key rotation process. It is a string
// type that implements the error interface.
type KeyRotatorError string

// Error returns the error message.
func (e KeyRotatorError) Error() string {
	return string(e)
}

const (
	// ErrKeyNotFound is returned when a key is not found in the storage.
	ErrKeyNotFound = KeyRotatorError("key not found")

	// ErrInvalidSettings is returned when the settings are invalid.
	ErrInvalidSettings = KeyRotatorError("invalid settings")

	// ErrInvalidRotationKeyCount is returned when the rotation key count is invalid.
	ErrInvalidRotationKeyCount = KeyRotatorError("invalid rotation key count")

	// ErrInvalidRotationInterval is returned when the rotation interval is invalid.
	ErrInvalidRotationInterval = KeyRotatorError("invalid rotation interval")

	// ErrInvalidKeyExpiration is returned when the key expiration is invalid.
	ErrInvalidKeyExpiration = KeyRotatorError("invalid key expiration")

	// ErrRotatorAlreadyRunning is returned when the rotator is already running.
	ErrRotatorAlreadyRunning = KeyRotatorError("rotator already running")

	// ErrNoKeysGenerated is returned when no keys were generated.
	ErrNoKeysGenerated = KeyRotatorError("no keys generated")

	// ErrInvalidArgument is returned when an invalid argument is passed.
	ErrInvalidArgument = KeyRotatorError("invalid argument")
)
