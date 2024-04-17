package krot

import (
	"encoding/json"
	"errors"
	"fmt"
)

// KrotErrorCode is an integer type used to represent different types of errors in the application.
type KrotErrorCode int

const (
	_ KrotErrorCode = iota

	// ErrCodeInvalidSettings is used when the settings provided are invalid.
	ErrCodeInvalidSettings
)

const (
	_ KrotErrorCode = 99 + iota

	// ErrCodeInvalidRotationKeyCount is used when the rotation key count is invalid.
	ErrCodeInvalidRotationKeyCount

	// ErrCodeInvalidRotationInterval is used when the rotation interval is invalid.
	ErrCodeInvalidRotationInterval

	// ErrCodeInvalidKeyExpiration is used when the key expiration is invalid.
	ErrCodeInvalidKeyExpiration

	// ErrCodeInvalidArgument is used when an invalid argument is passed.
	ErrCodeInvalidArgument

	// ErrCodeInvalidKeyProvidingMode is used when the key providing mode is invalid.
	ErrCodeInvalidKeyProvidingMode
)

const (
	_ KrotErrorCode = 199 + iota

	// ErrCodeRotatorAlreadyRunning is used when the rotator is already running.
	ErrCodeRotatorAlreadyRunning

	// ErrCodeNoKeysGenerated is used when no keys were generated.
	ErrCodeNoKeysGenerated
)

const (
	_ KrotErrorCode = 299 + iota

	// ErrCodeKeyNotFound is used when a key is not found in the storage.
	ErrCodeKeyNotFound
)

type KrotError interface {
	// Code returns the error code.
	Code() KrotErrorCode
	// Cause returns the cause of the error.
	Cause() error
	// Wrap wraps the error with the cause.
	Wrap(err error) error
	// Error returns the error message.
	Error() string
}

var (
	// ErrKeyNotFound is returned when a key is not found in the storage.
	ErrKeyNotFound = newError(ErrCodeKeyNotFound, "key not found")

	// ErrInvalidSettings is returned when the settings are invalid.
	ErrInvalidSettings = newError(ErrCodeInvalidSettings, "invalid settings")

	// ErrInvalidRotationKeyCount is returned when the rotation key count is invalid.
	ErrInvalidRotationKeyCount = newError(ErrCodeInvalidRotationKeyCount, "invalid rotation key count")

	// ErrInvalidRotationInterval is returned when the rotation interval is invalid.
	ErrInvalidRotationInterval = newError(ErrCodeInvalidRotationInterval, "invalid rotation interval")

	// ErrInvalidKeyExpiration is returned when the key expiration is invalid.
	ErrInvalidKeyExpiration = newError(ErrCodeInvalidKeyExpiration, "invalid key expiration")

	// ErrRotatorAlreadyRunning is returned when the rotator is already running.
	ErrRotatorAlreadyRunning = newError(ErrCodeRotatorAlreadyRunning, "rotator already running")

	// ErrNoKeysGenerated is returned when no keys were generated.
	ErrNoKeysGenerated = newError(ErrCodeNoKeysGenerated, "no keys generated")

	// ErrInvalidArgument is returned when an invalid argument is passed.
	ErrInvalidArgument = newError(ErrCodeInvalidArgument, "invalid argument")

	// ErrInvalidKeyProvidingMode is returned when the key providing mode is invalid.
	ErrInvalidKeyProvidingMode = newError(ErrCodeInvalidKeyProvidingMode, "invalid key providing mode")
)

type krotErrorJSON struct {
	Code    KrotErrorCode `json:"code"`
	Message string        `json:"message"`
	Cause   any           `json:"cause,omitempty"`
}

type krotError struct {
	code    KrotErrorCode
	message string
	cause   error
}

func newError(code KrotErrorCode, message string) KrotError {
	return &krotError{
		code:    code,
		message: message,
	}
}

func (e *krotError) Code() KrotErrorCode {
	return e.code
}

func (e *krotError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

func (e *krotError) Cause() error {
	return e.cause
}

func (e *krotError) Wrap(err error) error {
	if e.cause != nil || err == nil {
		return e
	}

	wrapped := *e
	wrapped.cause = err
	return &wrapped
}

func (e *krotError) Unwrap() error {
	return e.cause
}

func (e *krotError) Is(target error) bool {
	if e == nil {
		return false
	}

	err, ok := target.(KrotError)
	if ok {
		return e.Code() == err.Code()
	}

	return false
}

func (e *krotError) MarshalJSON() ([]byte, error) {
	errorJSON := &krotErrorJSON{
		Code:    e.code,
		Message: e.message,
	}

	if e.cause == nil {
		return json.Marshal(errorJSON)
	}

	switch cause := e.cause.(type) {
	case KrotError:
		errorJSON.Cause = cause

	case interface{ Unwrap() []error }:
		originalErrs := cause.Unwrap()
		errs := make([]string, 0, len(originalErrs))
		for _, err := range originalErrs {
			errs = append(errs, err.Error())
		}

		errorJSON.Cause = errs

	default:
		if errors.Unwrap(cause) == nil {
			errorJSON.Cause = cause.Error()
			break
		}

		errs := []string{}
		for ; cause != nil; cause = errors.Unwrap(cause) {
			errs = append(errs, cause.Error())
		}

		errorJSON.Cause = errs
	}

	return json.Marshal(errorJSON)
}

func (e *krotError) UnmarshalJSON(source []byte) error {
	err := &krotErrorJSON{}
	if err := json.Unmarshal(source, err); err != nil {
		return err
	}

	e.code = err.Code
	e.message = err.Message

	switch cause := err.Cause.(type) {
	case string:
		e.cause = errors.New(cause)

	case []any:
		errs := make([]error, len(cause))
		for i, err := range cause {
			if err == nil {
				continue
			}

			if errString, ok := err.(string); ok {
				errs[i] = errors.New(errString)
			}
		}

		e.cause = errors.Join(errs...)

	case map[string]any:
		causeBytes, _ := json.Marshal(cause)
		err := &krotError{}
		if err := json.Unmarshal(causeBytes, err); err != nil {
			return err
		}

		e.cause = err
	}

	return nil
}
