package style

import (
	"sort"
	"sync"
)

var (
	global List
)

type Provider interface {
	// For will return a url to the style and if it was found.
	// if name is not found it will return the default style, and false for
	// found. If name is "", then For should return the default style, with
	// found being set to true
	// style should only be an empty style struct if no styles are defined in the system. Then
	// found should also be false
	For(name string) (style Style, found bool)

	// Styles returns the list of known styles.
	Styles() []string
}

// List holds a set of styles
type List struct {
	lck    sync.RWMutex
	styles []Style
	idx    map[string]int
}

// For returns the style in the list with the given name. If the style is not
// found it the default style will be returned and false.
func (s *List) For(name string) (Style, bool) {
	if s == nil {
		return global.For(name)
	}
	s.lck.RLock()
	defer s.lck.RUnlock()
	if len(s.styles) == 0 || len(s.idx) == 0 {
		return Style{}, false
	}
	if name == "" {
		return s.styles[0], true
	}
	idx, ok := s.idx[name]
	return s.styles[idx], ok
}

// Styles returns the known styles
func (s *List) Styles() []string {
	if s == nil {
		return global.Styles()
	}
	names := make([]string, len(s.styles))
	for i := range s.styles {
		names[i] = s.styles[i].Name
	}
	return names
}

// Append will add a style to the system
func (s *List) Append(styles ...Style) error {
	if s == nil {
		return global.Append(styles...)
	}
	s.lck.Lock()
	defer s.lck.Unlock()
	if s.idx == nil {
		s.idx = make(map[string]int)
	}
	for i := range styles {
		if _, ok := s.idx[styles[i].Name]; ok {
			return ErrAlreadyExists{
				Style: styles[i],
				List:  s,
			}
		}
		s.styles = append(s.styles, styles[i])
		s.idx[styles[i].Name] = len(s.styles) - 1
	}
	return nil
}

// SubList will return a Style list that will reference the main list and
// limit the styles to names
func (s *List) SubList(names ...string) *Sublist {

	if s == nil {
		return global.SubList(names...)
	}

	return &Sublist{
		Names: names,
		Main:  s,
	}
}

// Style describes a style
type Style struct {
	Name        string
	Location    string
	Description string
}

// Sublist is a sublist of the main set of list
type Sublist struct {
	Names []string
	Main  Provider
}

// For will return the Style for the give name
func (s Sublist) For(name string) (Style, bool) {
	if len(s.Names) == 0 {
		return s.Main.For(name)
	}
	if name == "" {
		return s.Main.For(s.Names[0])
	}

	if len(s.Names) == 1 {
		sty, _ := s.Main.For(s.Names[0])
		return sty, name == s.Names[0]
	}

	// make sure name is in our style list
	for i := range s.Names {
		if s.Names[i] == name {
			return s.Main.For(name)
		}
	}

	// if the name is in the our list return the default
	// with false for found
	sty, _ := s.Main.For(s.Names[0])
	return sty, false
}

// Styles returns the current styles names in the list
func (s Sublist) Styles() []string {

	var names []string
	mnames := s.Main.Styles()
	if len(mnames) == 0 {
		return []string{}
	}
	lookup := make(map[string]bool, len(mnames))
	for _, name := range mnames {
		lookup[name] = true
	}

	for _, name := range s.Names {
		if lookup[name] {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}

// SubList generates a sublist based on the names given
func (s *Sublist) SubList(names ...string) *Sublist {
	if s == nil {
		return global.SubList(names...)
	}
	return &Sublist{
		Names: names,
		Main:  s,
	}
}

// Append the style to the global list of styles
func Append(styles ...Style) error {
	return global.Append(styles...)
}

// SubList returns a sublist based on the global List
func SubList(names ...string) *Sublist { return global.SubList(names...) }

// Styles returns the last of styles in the global style list
func Styles() []string {
	return global.Styles()
}

// For returns the style for the given name and if it was found in the global registary
func For(name string) (Style, bool) { return global.For(name) }
