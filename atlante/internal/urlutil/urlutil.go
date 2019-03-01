package urlutil

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func ReadAll(location *url.URL) (b []byte, err error) {
	if location == nil {
		return nil, errors.New("nil url provided")
	}

	err = VisitReader(location, func(r io.Reader) error {
		var e error
		b, e = ioutil.ReadAll(r)
		return e
	})
	return b, err
}

func VisitReader(location *url.URL, fn func(io.Reader) error) error {
	var reader io.Reader
	if location == nil {
		return errors.New("nil url provided")
	}
	switch strings.ToLower(location.Scheme) {
	case "", "file":

		// check the conf file exists
		filename := location.EscapedPath()
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return fmt.Errorf("config file at location (%v) not found!", filename)
		}

		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("error opening local config file (%v): %v", filename, err)
		}
		defer file.Close()

		reader = file

	case "http", "https":

		var httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}

		res, err := httpClient.Get(location.String())
		if err != nil {
			return fmt.Errorf("config file at location (%v) not found: %v", location.String(), err)
		}

		reader = res.Body

	default:

		return fmt.Errorf("scheme (%v) not supported.", location.Scheme)

	}

	return fn(reader)
}
