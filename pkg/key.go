package kr

import "time"

type Key struct {
	ID      string    `json:"id"`
	Value   string    `json:"value"`
	Expires time.Time `json:"expires"`
}

func (k *Key) Deprecated() bool {
	return k.Expires.After(time.Now())
}
