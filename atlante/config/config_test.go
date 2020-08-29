package config

import (
	"os"
	"testing"
)

func resetEnv(newEnv map[string]string) (old map[string]string) {
	envs := os.Environ()
	old = make(map[string]string, len(envs))
	for _, s := range envs {
		for j := 0; j < len(s); j++ {
			if s[j] == '=' {
				old[s[:j]] = s[j+1:]
			}
		}
	}
	os.Clearenv()
	if newEnv == nil {
		return old
	}
	for k, v := range newEnv {
		os.Setenv(k, v)
	}
	return old
}
func TestLoad(t *testing.T) {

	type tcase struct {
		file   string            // This name of the file to load from the testdata directory
		Err    error             // expected error
		config Config            // Expected Config
		Env    map[string]string // The expected environment.
	}

	// These test can not run in parallel, as we are modifying the os.Env
	fn := func(tc tcase) (string, func(*testing.T)) {
		return tc.file, func(t *testing.T) {

			oldEnv := resetEnv(tc.Env)
			// make sure we clean up after ourselfs
			defer resetEnv(oldEnv)

		}
	}
	tests := []tcase{
		//...
	}
	for _, test := range tests {
		t.Run(fn(test))
	}

}
