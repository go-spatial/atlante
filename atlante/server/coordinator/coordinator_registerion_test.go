package coordinator_test

import (
	"testing"

	_ "github.com/go-spatial/atlante/atlante/server/coordinator/logger"
	_ "github.com/go-spatial/atlante/atlante/server/coordinator/null"
	_ "github.com/go-spatial/atlante/atlante/server/coordinator/postgresql"

	"github.com/gdey/errors"
	"github.com/go-spatial/tegola/dict"

	"github.com/go-spatial/atlante/atlante/server/coordinator"
)

func TestKnownType(t *testing.T) {
	type tcase struct {
		Type  string
		Known bool
	}

	fn := func(tc tcase) (string, func(*testing.T)) {
		return tc.Type, func(t *testing.T) {
			_, err := coordinator.For(tc.Type, dict.Dict{})
			switch e := err.(type) {
			case coordinator.ErrUnknownProvider:
				if tc.Known {
					t.Errorf("known, expected true, got false")
				}
				return
			case errors.String:
				if e == coordinator.ErrNoProvidersRegistered {
					panic("No coordinators are registered, need at least one registered for test.")
					return
				}
			}
			if !tc.Known {
				t.Logf("Got the following error: %v", err)
				t.Errorf("known, expected false, got true")
			}
		}
	}
	tests := []tcase{
		{Type: "postgres"},
		{Type: "unkown"},
	}
	// Add all the known types as well.
	for _, typ := range coordinator.Registered() {
		tests = append(tests, tcase{Type: typ, Known: true})
	}
	for _, test := range tests {
		t.Run(fn(test))
	}
}
