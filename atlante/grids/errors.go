package grids

import (
	"github.com/gdey/errors"
)

const (
	// ErrNotFound is returned when an requested object does not exist in the
	// grid provider
	ErrNotFound = errors.String("grid not found")

	// ErrNoProvidersRegistered is returned when providers have not been registered with the system
	ErrNoProvidersRegistered = errors.String("no providers registered")
)

// ErrProviderTypeExists is returned when the Provider type was already registered.
type ErrProviderTypeExists string

func (err ErrProviderTypeExists) Error() string {
	return "provider (" + string(err) + ") already exists"
}

// ErrProviderNotRegistered is returned when the requested provider has not registered
type ErrProviderNotRegistered string

func (err ErrProviderNotRegistered) Error() string {
	return "provider (" + string(err) + ") not registered"
}
