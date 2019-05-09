package notifiers

import (
	"sort"
	"sync"

	"github.com/gdey/errors"
	"github.com/go-spatial/tegola/dict"
)

type ErrNotifierAlreadyExists string

func (err ErrNotifierAlreadyExists) Error() string {
	return "notifier (" + string(err) + ") already exists"
}
func (err ErrNotifierAlreadyExists) Cause() error { return nil }

const (
	ErrNoNotifiersRegistered = errors.String("no notifiers registered")
	ErrKey                   = errors.String("bad key provided")
)

type NotifierConfiger interface {
	dict.Dicter
	// NamedNotifierProvider returns a configured Notifer for the provided key.
	// if the named provider does not exist ErrNoNotifiersRegistered will be returned
	NamedNotifierProvider(name string) (Notifier, error)
}

/*****************************************************************************/

// InitFunc initilizes a notifier given the config.
// InitFunc should validate the config, and report any errors.
type InitFunc func(NotifierConfiger) (Notifer, error)

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
		return ErrNotifierAlreadyExists(notifierType)
	}
	notifiers[notiferType] = funcs{
		init:    init,
		cleanup: cleanup,
	}
	return nil
}

// Unregister will remove a notifier and call it's clean up function
func Unregister(notifierType string) {
	notifiersLock.Lock()
	defer notifiersLock.Unlock()

	n, ok := notifiers[notifiersType]
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
func For(notifierType string, config NotifierConfig) (Notifier, error) {
	notifiersLock.RLock()
	defer notifiersLock.RUnlock()
	if notifiers == nil {
		return nil, ErrNoNotifiersRegistered
	}
	n, ok := notifiers[notifierType]
	if !ok {
		return nil, ErrNotifiernotRegistered(notifierType)
	}
	return n.init(config)
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
