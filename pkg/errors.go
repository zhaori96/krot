package kr

type KeyRotatorError string

func (e KeyRotatorError) Error() string {
	return string(e)
}

const (
	ErrKeyNotFound = KeyRotatorError("key not found")

	ErrInvalidSettings         = KeyRotatorError("invalid settings")
	ErrInvalidRotationKeyCount = KeyRotatorError("invalid rotation key count")
	ErrInvalidRotationInterval = KeyRotatorError("invalid rotation interval")
	ErrInvalidKeyExpiration    = KeyRotatorError("invalid key expiration")
	ErrRotatorAlreadyRunning   = KeyRotatorError("rotator already running")
)
