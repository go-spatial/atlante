package insetmap

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/go-spatial/atlante/internal/env"
	"github.com/gorilla/css/scanner"
)

type ConfigBoundarySql struct {
	Main     env.String `toml:"sql"`
	Boundary env.String `toml:"boundary_sql"`
}

type ConfigLayer struct {
	Name env.String `toml:"name"`
	SQL  env.String `toml:"sql"`
}

type Config struct {
	Scale      env.Uint            `toml:"scale"`
	ViewBuffer env.Uint            `toml:"view_buffer"`
	Sheet      env.String          `toml:"main_sql"`
	Adjoining  env.String          `toml:"adjoining_sql"`
	Layers     []ConfigLayer       `toml:"layers"`
	Boundaries []ConfigBoundarySql `toml:"boundaries"`
	CSSDir     env.String          `toml:"css_dir"`
	CSSDefault env.String          `toml:"css_default"`
}

type CSSInfo struct {
	Path string
	Desc string
}

type CSSMap map[string]CSSInfo

func cssGetDesc(filepath string) (string, error) {
	contents, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read css file(%v): %w", filepath, err)
	}
	if len(contents) == 0 {
		return "", nil
	}
	s := scanner.New(string(contents))
	for {
		token := s.Next()
		if token.Type == scanner.TokenEOF || token.Type == scanner.TokenError {
			break
		}
		if token.Type != scanner.TokenComment {
			continue
		}
		if len(token.Value) <= 4 {
			continue
		}
		return strings.TrimSpace(token.Value[2 : len(token.Value)-2]), nil
	}
	return "", nil
}

//GetStyleSheets will load and parse css files in the
// given directory. The desc is the first comment in
// the css file.
func (cssmap CSSMap) GetStyleSheets(dir string) error {

	if debug {
		log.Printf("[DEBUG] Getting style sheets for %v -- %v", dir, len(cssmap))
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading dir %v: %w", dir, err)
	}
	for _, f := range files {
		name := f.Name()
		if debug {
			log.Printf("looking @ %v", name)
		}
		if f.IsDir() {
			// TODO(gdey):  should we recurse into the dir.
			continue
		}
		ext := filepath.Ext(name)
		if ext != ".css" {
			continue
		}
		filename := filepath.Join(dir, name)
		desc, err := cssGetDesc(filename)
		if err != nil {
			return err
		}
		key := name[:len(name)-len(ext)]
		if debug {
			log.Printf("[DEBUG-css] for key: %v found %v -- %v", key, name, desc)
		}
		cssmap[key] = CSSInfo{
			Path: filename,
			Desc: desc,
		}
	}
	if debug {
		log.Printf("[DEBUG] got style sheets for %v -- %v", dir, len(cssmap))
	}
	return nil
}
