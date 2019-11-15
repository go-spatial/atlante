package remote

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
	"github.com/go-spatial/maptoolkit/atlante/internal/urlutil"
)

var RemoteDir = "remoteasset"

type remoteMeta struct {
	Filename string    `json:"filename"`
	ETag     string    `json:"etag"`
	URL      string    `json:"url"`
	Date     time.Time `json:"timestamp"`
}

func attemptWrite(w filestore.FileWriter, name string, data []byte) (n int, err error) {
	writer, err := w.Writer(name, true)
	if err != nil {
		return 0, err
	}
	return writer.Write(data)
}

func Remote(location string, fswriter filestore.FileWriter, useCached bool) (string, error) {
	loc, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	loc.Host = strings.ToLower(loc.Hostname())
	shaBytes := sha1.Sum([]byte(loc.String()))
	shaFilename := fmt.Sprintf("%x", shaBytes)
	svgBytes, err := urlutil.ReadAll(loc)
	if err != nil {
		return "", err
	}
	svgFilename := filepath.Join(RemoteDir, shaFilename+".svg")

	if useCached {
		exister, ok := fswriter.(filestore.Exister)
		if ok {
			if exister.Exists(svgFilename) {
				return svgFilename, nil
			}
		}
	}

	metaFilename := filepath.Join(RemoteDir, shaFilename+".json")
	meta := remoteMeta{
		Filename: svgFilename,
		URL:      location,
		Date:     time.Now(),
	}
	// if there is an err  the meta data let's not worry about it.
	if metaBytes, err := json.Marshal(meta); err == nil {
		if _, err = attemptWrite(fswriter, metaFilename, metaBytes); err != nil {
			return "", err
		}
	} else {
		log.Printf("issue marshaling meta data for %v: %v", location, err)
	}

	if _, err = attemptWrite(fswriter, svgFilename, svgBytes); err != nil {
		return "", err
	}
	return svgFilename, nil
}
