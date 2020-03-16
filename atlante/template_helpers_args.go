package atlante

import (
	"fmt"
	"strings"

	"github.com/gdey/as"
	"github.com/prometheus/common/log"
)

// tplArgs provides a key value based way to pass around
// template parameters
type tplArgs struct {
	data map[string]interface{}
}

// tplArgsKeyReplacer is used to normalize the args key
var tplArgsKeyReplacer = strings.NewReplacer("_", "-", " ", "-", ".", "-", "/", "-", "\n", "-", "\r", "-")

// normalizeKey will lowercase the key and normalize it using the argsKeyReplacer
func normalizeKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	return tplArgsKeyReplacer.Replace(key)
}

func checkArgs(args *tplArgs, keys ...string) (bool, error) {
	if args == nil {
		return false, fmt.Errorf("Invalid args")
	}
	missing := args.Required(keys...)
	if len(missing) > 0 {
		return false, fmt.Errorf("Missing required keys: %v", strings.Join(missing, ","))
	}
	return true, nil
}

func NewTplArgsFromMapStringString(args map[string]string) *tplArgs {
	tplargs := &tplArgs{
		data: make(map[string]interface{}, len(args)),
	}
	if len(args) == 0 {
		return tplargs
	}
	for k, v := range args {
		tplargs.data[k] = v
	}
	return tplargs
}

func NewTplArgs(keyvals ...interface{}) (*tplArgs, error) {
	var (
		args tplArgs
		key  string
		ok   bool
		val  interface{}
	)
	args.data = make(map[string]interface{}, len(keyvals)/2)
	for i := 0; i < len(keyvals); i++ {
		// first entry is the key
		key, ok = as.String(keyvals[i])
		if !ok {
			return nil, fmt.Errorf("item %v invalid key %v", i, keyvals[i])
		}
		key = normalizeKey(key)
		i++
		val = nil
		if i < len(keyvals) {
			val = keyvals[i]
		}
		args.data[key] = val
	}
	return &args, nil
}

func (a *tplArgs) With(keys ...string) (*tplArgs, error) {
	if a == nil {
		return nil, nil
	}
	na := &tplArgs{
		data: make(map[string]interface{}, len(a.data)),
	}
	if a.data == nil {
		return na, nil
	}
	for _, key := range keys {
		val, ok := a.data[key]
		if !ok {
			continue
		}
		na.data[key] = val
	}
	return na, nil
}

// Get get the normalized version of the key's data
func (a *tplArgs) Get(okey string) (interface{}, error) {
	key := normalizeKey(okey)
	if a == nil || a.data == nil {
		log.Infof("a or a.data is nil")
		return nil, fmt.Errorf("Unknown key %v (%v)", okey, key)
	}
	val, ok := a.data[key]
	if !ok {
		log.Infof("Known keys: %v", a.data)
		return val, fmt.Errorf("Unknown key %v (%v)", okey, key)
	}
	return val, nil
}
func (a *tplArgs) Set(keyvals ...interface{}) (*tplArgs, error) {
	if a == nil {
		return a, fmt.Errorf("args is nil")
	}
	var (
		key string
		ok  bool
		val interface{}
	)

	if a.data == nil {
		a.data = make(map[string]interface{})
	}

	for i := 0; i < len(keyvals); i++ {
		// first entry is the key
		key, ok = as.String(keyvals[i])
		if !ok {
			return nil, fmt.Errorf("item %v invalid key %v", i, keyvals[i])
		}
		key = normalizeKey(key)
		i++
		val = nil
		if i < len(keyvals) {
			val = keyvals[i]
		}
		a.data[key] = val
	}
	return a, nil
}
func (a *tplArgs) SetOptional(keyvals ...interface{}) (*tplArgs, error) {
	if a == nil {
		return a, fmt.Errorf("args is nil")
	}
	var (
		key string
		ok  bool
		val interface{}
	)

	if a.data == nil {
		a.data = make(map[string]interface{})
	}

	for i := 0; i < len(keyvals); i++ {
		// first entry is the key
		key, ok = as.String(keyvals[i])
		if !ok {
			return nil, fmt.Errorf("item %v invalid key %v", i, keyvals[i])
		}
		key = normalizeKey(key)
		i++
		val = nil
		if i < len(keyvals) {
			val = keyvals[i]
		}
		if _, ok := a.data[key]; !ok {
			a.data[key] = val
		}
	}
	return a, nil

}
func (a *tplArgs) Has(key string) bool {
	if a == nil || a.data == nil {
		return false
	}
	_, ok := a.data[normalizeKey(key)]
	return ok
}

func (a *tplArgs) Required(keys ...string) []string {
	var (
		ok            bool
		key           string
		missingParams []string
	)
	if a == nil || a.data == nil {
		return keys
	}
	for i := range keys {
		key = normalizeKey(keys[i])
		if _, ok = a.data[key]; !ok {
			missingParams = append(missingParams, key)
		}
	}
	return missingParams
}

func (a *tplArgs) GetAsFloat64(key string) (float64, error) {
	val, err := a.Get(key)
	if err != nil {
		return 0.0, err
	}
	num, ok := as.Float64(val)
	if !ok {
		return 0.0, fmt.Errorf("%v not a float value", key)
	}
	return num, nil
}
func (a *tplArgs) GetAsInt64(key string) (int64, error) {
	val, err := a.Get(key)
	if err != nil {
		return 0, err
	}
	num, ok := as.Int64(val)
	if !ok {
		return 0, fmt.Errorf("%v not an int value", key)
	}
	return num, nil
}
func (a *tplArgs) GetAsInt(key string) (uint, error) {
	num, err := a.GetAsInt64(key)
	return uint(num), err
}

func (a *tplArgs) GetAsUint64(key string) (uint64, error) {
	val, err := a.Get(key)
	if err != nil {
		return 0, err
	}
	num, ok := as.Uint64(val)
	if !ok {
		return 0, fmt.Errorf("%v not an uint value", key)
	}
	return num, nil
}
func (a *tplArgs) GetAsUint(key string) (uint, error) {
	num, err := a.GetAsUint64(key)
	return uint(num), err
}
func (a *tplArgs) GetAsBool(key string) (bool, error) {
	val, err := a.Get(key)
	if err != nil {
		return false, err
	}
	boolean, ok := as.Bool(val)
	if !ok {
		return false, fmt.Errorf("%v not an bool value", key)
	}
	return boolean, nil
}
func (a *tplArgs) GetAsString(key string) (string, error) {
	val, err := a.Get(key)
	if err != nil {
		return "", err
	}
	str, ok := as.String(val)
	if !ok {
		return "", fmt.Errorf("%v not an string value", key)
	}
	return str, nil

}
func (a *tplArgs) GetAsArgs(key string) (*tplArgs, error) {
	val, err := a.Get(key)
	if err != nil {
		return nil, err
	}
	switch newArgs := val.(type) {
	case *tplArgs:
		return newArgs, nil
	case tplArgs:
		return &newArgs, nil
	}
	return nil, fmt.Errorf("%k does not contains args")
}
