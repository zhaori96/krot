package krot

import "time"

// Key represents a key with an ID, value, and expiration time. The ID is a unique
// identifier for the key. The value is the actual key data. The expiration time
// is the time at which the key expires.
type Key struct {
	ID      string    `json:"id"`
	Value   any       `json:"value"`
	Expires time.Time `json:"expires"`
}

// Expired checks if the key has expired. It returns true if the key's expiration
// time is after the current time, and false otherwise.
//
//	if key.Expired() {
//	    fmt.Println("The key has expired.")
//	} else {
//	    fmt.Println("The key has not expired.")
//	}
func (k *Key) Expired() bool {
	return k.Expires.After(time.Now())
}
