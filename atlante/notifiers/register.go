package notifiers

import (
	"sort"
	"sync"

	"github.com/gdey/errors"
	"github.com/go-spatial/tegola/dict"
)

type ErrAlreadyExists string

func (err ErrAlreadyExists) Error() string {
	return "notifier (" + string(err) + ") already exists"
}
func (err ErrAlreadyExists) Cause() error { return nil }

const (
	ErrNoneRegistered = errors.String("no notifiers registered")
	ErrKey            = errors.String("bad key provided")

	// ConfigKeyType is the name for the config key
	ConfigKeyType = "type"
)

type Config interface {
	dict.Dicter
}

/*****************************************************************************/

// InitFunc initilizes a notifier given the config.
// InitFunc should validate the config, and report any errors.
type InitFunc func(Config) (Provider, error)

// CleanupFunc is called when the system is shutting down.
// this allows the provider do any needed cleanup.
type CleanupFunc func()

type funcs struct {
	init    InitFunc
	cleanup CleanupFunc
}

var (
	notifiersLock sync.RWMutex
	notifiers     map[string]funcs
)

// Register is called by the init functions of the provider
func Register(notifierType string, init InitFunc, cleanup CleanupFunc) error {
	notifiersLock.Lock()
	defer notifiersLock.Unlock()

	if notifiers == nil {
		notifiers = make(map[string]funcs)
	}
	if _, ok := notifiers[notifierType]; ok {
		return ErrAlreadyExists(notifierType)
	}
	notifiers[notifierType] = funcs{
		init:    init,
		cleanup: cleanup,
	}
	return nil
}

// Unregister will remove a notifier and call it's clean up function
func Unregister(notifierType string) {
	notifiersLock.Lock()
	defer notifiersLock.Unlock()

	n, ok := notifiers[notifierType]
	if !ok {
		return // nothing to do.
	}
	if n.cleanup != nil {
		n.cleanup()
	}
	delete(notifiers, notifierType)
}

// Registered returns the notifiers that have been registered
func Registered() (n []string) {
	n = make([]string, len(notifiers))
	i := 0
	notifiersLock.RLock()
	for k := range notifiers {
		n[i] = k
		i++
	}
	notifiersLock.RUnlock()
	sort.Strings(n)
	return n
}

// For function returns a configured provider of the given type, and provided the correct config.
func For(notifierType string, config Config) (Provider, error) {
	notifiersLock.RLock()
	defer notifiersLock.RUnlock()
	if notifiers == nil {
		return nil, ErrNoneRegistered
	}
	n, ok := notifiers[notifierType]
	if !ok {
		return nil, ErrNoneRegistered
	}
	return n.init(config)
}

// From is like For but assumes that the config has a ConfigKeyType value informing the type
// of provider being configured
func From(config Config) (Provider, error) {
	cType, err := config.String(ConfigKeyType, nil)
	if err != nil {
		return nil, err
	}
	return For(cType, config)
}

// Cleanup should be called when the system is shutting down. This given each provider
// a chance to do any needed cleanup. This will unregister all providers.
func Cleanup() {
	notifiersLock.Lock()
	for _, n := range notifiers {
		if n.cleanup != nil {
			n.cleanup()
		}
	}
	notifiers = make(map[string]funcs)
	notifiersLock.Unlock()
}
