package queuer

import (
	"fmt"
	"sort"
	"sync"

	"github.com/gdey/errors"
	"github.com/go-spatial/tegola/dict"
	"github.com/prometheus/common/log"
)

// ErrProviderTypeExists is returned when a provider is already registered with that name
type ErrProviderTypeExists string

func (err ErrProviderTypeExists) Error() string {
	return "queue provider (" + string(err) + ") already exists"
}

const (
	// ErrNoProvidersRegistered is returned when no queuers are registered with the system
	ErrNoProvidersRegistered = errors.String("no queue providers registered")
)

// ErrUnknownProvider is returned when a requested queuer is not registered
type ErrUnknownProvider string

func (err ErrUnknownProvider) Error() string {
	return fmt.Sprintf("error unknown queue provider %v", string(err))
}

// Config is the interface that is passed to the queue provider to configure them
type Config interface {
	dict.Dicter
}

// InitFunc initilizes a queue provider given a config
// The InitFunc should validate the config and report any errors.
// Called by the For function
type InitFunc func(Config) (Provider, error)

// CleanupFunc is called when the system is shuting down;
// Allows queue provider a way to do cleanup
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
	log.Infof("registered queue provider: %v", providerType)
	return nil
}

// Unregister will remove a provider and call it's cleanup function
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

// For function returns a configured provider given the trype and config
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

// Cleanup should be called when the system is shutting down. This gives weach provider
// a chance to do any needed cleanup. this will unreigster all providers
func Cleanup() {
	providerLock.Lock()
	for _, p := range providers {
		if p.cleanup == nil {
			continue
		}
		p.cleanup()
	}
	providers = make(map[string]funcs)
	providerLock.Unlock()
}
