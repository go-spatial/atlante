package grids

import (
	"errors"
	"sort"
	"sync"

	"github.com/go-spatial/tegola/dict"
)

// ErrProviderTypeExists is returned when the Provider type was already registered.
type ErrProviderTypeExists string

func (err ErrProviderTypeExists) Error() string {
	return "provider (" + string(err) + ") already exists"
}

// ErrNoProvidersRegistered is returned when providers have not been registered with the system
var ErrNoProvidersRegistered = errors.New("no providers registered")

// ErrProviderNotRegistered is returned when the requested provider has not registered
type ErrProviderNotRegistered string

func (err ErrProviderNotRegistered) Error() string {
	return "provider (" + string(err) + ") not registered"
}

// ProviderConfig implements the ProviderConfig interface
type ProviderConfig interface {
	dict.Dicter
	// Returns the Grid Provider for the key
	// if the a Provider does not exist ErrKeyMissingProvider will be returned
	// if the key does not exist ErrNoProvidersRegistered will be returned
	NameGridProvider(Key string) (Provider, error)
}

/******************************************************************************/

// InitFunc initilizes a grid Provider given a config map.
// The InitFunc should validate the config map, and report any
// errors.
// this is called by the For function.
type InitFunc func(ProviderConfig) (Provider, error)

// CleanupFunc is called when the system is shuting down;
// allowing the provider to do any needed cleanup.
type CleanupFunc func()
type funcs struct {
	init    InitFunc
	cleanup CleanupFunc
}

var providersLock sync.RWMutex
var providers map[string]funcs

// Register is called by the init functions of the provider
func Register(providerType string, init InitFunc, cleanup CleanupFunc) error {
	providersLock.Lock()
	defer providersLock.Unlock()

	if providers == nil {
		providers = make(map[string]funcs)
	}
	if _, ok := providers[providerType]; ok {
		return ErrProviderTypeExists(providerType)
	}
	providers[providerType] = funcs{
		init:    init,
		cleanup: cleanup,
	}
	return nil
}

// Unregister will remove a provider and call it's clean up function.
func Unregister(providerType string) {

	providersLock.Lock()
	defer providersLock.Unlock()

	p, ok := providers[providerType]
	if !ok {
		return // nothing to do
	}
	p.cleanup()

	delete(providers, providerType)
}

// Registered returns the providers that have been registered
func Registered() (p []string) {
	p = make([]string, len(providers))
	i := 0
	providersLock.RLock()
	for k := range providers {
		p[i] = k
		i++
	}
	providersLock.RUnlock()
	sort.Strings(p)
	return p
}

// For function returns a configured provider of the given type, provided the correct config.
func For(providerType string, config ProviderConfig) (Provider, error) {
	providersLock.RLock()
	defer providersLock.RUnlock()
	if providers == nil {
		return nil, ErrNoProvidersRegistered
	}

	p, ok := providers[providerType]
	if !ok {
		return nil, ErrProviderNotRegistered(providerType)
	}
	return p.init(config)
}

// Cleanup should be called when the system is shutting down. This given each provider
// a chance to do any needed cleanup. This will unregister all providers.
func Cleanup() {
	providersLock.Lock()
	for _, p := range providers {
		p.cleanup()
	}
	providers = make(map[string]funcs)
	providersLock.Unlock()
}
