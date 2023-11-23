package kr

type KeyRotatorError string

func (e KeyRotatorError) Error() string {
	return string(e)
}

const (
	ErrKeyNotFound = KeyRotatorError("key not found")
)
