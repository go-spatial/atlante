package config

import "testing"

func TestLoad(t *testing.T) {

	type tcase struct {
		file   string // This name of the file to load from the testdata directory
		Err    error  // expected error
		config Config // Expected Config
	}

	fn := func(tc tcase) (string, func(*testing.T)) {
		return tc.file, func(t *testing.T) {
			//...
		}
	}
	tests := []tcase{
		//...
	}
	for _, test := range tests {
		t.Run(fn(test))
	}

}
