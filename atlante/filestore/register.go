package filestore

import (
	"sort"
	"sync"

	"github.com/gdey/errors"
	"github.com/go-spatial/tegola/dict"
)

// ErrProviderTypeExists occurs when a writer provider is trying to register with the same
// name as another provider.
// TODO(gdey): should this panic?
type ErrProviderTypeExists string

func (err ErrProviderTypeExists) Error() string {
	return "filestore provider (" + string(err) + ") already exists"
}

// ErrNoProvidersRegistered is returned when no providers are registered with the system
const ErrNoProvidersRegistered = errors.String("no providers registered")

// ErrUnknownProvider Returned when a requested provided is not registered
type ErrUnknownProvider string

func (err ErrUnknownProvider) Error() string {
	return "unknown filestore provider (" + string(err) + ")"
}

// Config is the interface that is passed to filestore providers to configure them
type Config interface {
	dict.Dicter
	// Returns a filestore provider for the given key in the config
	// If the a provider does not exist ErrUnknownProvider will be returned
	FileStoreFor(key string) (Provider, error)
}

// InitFunc initilizes a filestore provider given a config
// The InitFunc should validate the config and report any errors.
// Called by the For function.
type InitFunc func(Config) (Provider, error)

// CleanupFunc is called when the system is shuting down;
// Allows the filestore provider to do any needed cleanup.
type CleanupFunc func()

type funcs struct {
	init    InitFunc
	cleanup CleanupFunc
}

var providerLock sync.RWMutex
var providers map[string]funcs

// Register is called by the init functions of each of the providers
func Register(providerType string, init InitFunc, cleanup CleanupFunc) error {
	providerLock.Lock()
	defer providerLock.Unlock()

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

// Unregister will remove a provider and call it's cleanup function.
func Unregister(providerType string) {
	providerLock.Lock()
	defer providerLock.Unlock()

	p, ok := providers[providerType]
	if !ok {
		return // nothing to do
	}

	p.cleanup()
	delete(providers, providerType)
}

// Registered returns the providers that have been registered
func Registered() []string {
	p := make([]string, len(providers))
	i := 0
	providerLock.RLock()
	for k := range providers {
		p[i] = k
		i++
	}
	providerLock.RUnlock()
	sort.Strings(p)
	return p
}

//For function returns a configured provider given the type and config
func For(providerType string, config Config) (Provider, error) {
	providerLock.RLock()
	defer providerLock.RUnlock()

	if providers == nil {
		return nil, ErrNoProvidersRegistered
	}

	p, ok := providers[providerType]
	if !ok {
		return nil, ErrUnknownProvider(providerType)
	}
	return p.init(config)
}

// Cleanup should be called when the system is shutting down. This gives each provider
// a chance to do any needed cleanup. This will unregister all providers.
func Cleanup() {
	providerLock.Lock()
	for _, p := range providers {
		p.cleanup()
	}
	providers = make(map[string]funcs)
	providerLock.Unlock()
}
