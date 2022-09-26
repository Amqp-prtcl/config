package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type ConfigType int

const (
	Json ConfigType = iota
	Line
)

// Keeps config in-memory reopening and closing for each I/O operation
//
// Config is safe for concurrent use
type Config struct {
	Config   map[string]interface{}
	Filepath string
	mu       sync.RWMutex

	Type ConfigType
}

// Maybe ?: In the case that a value is present in both default map and in file
// but are of different kind, the value in default map is used

// Default values are optionals (set to nil or use an empty map to skip it); LoadConfigFile will use the map to make up
// for all value present in Default but not in file.
//
// JSON config only accepts strings; booleans; float64; and structs or arrays containing them.
// ints; units; and float32 are saved and decoded as float64
//
// Line config only supports values with kind float64, strings, and booleans.
// ints; units; and float32 are saved and decoded as float64
//
// Warning: LoadConfigFile only does shallow copies of values in default (take care about race conditions)
func LoadConfigFile(filepath string, configType ConfigType) (*Config, error) {
	var config = &Config{
		Config:   map[string]interface{}{},
		Filepath: filepath,
		mu:       sync.RWMutex{},
		Type:     configType,
	}
	f, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	defer f.Close()
	switch configType {
	case Json:
		config.Config, err = parseFileLine(f)
	case Line:
		config.Config, err = parseFileLine(f)
	default:
		config.Type = Json
		config.Config, err = parseFileJSON(f)
	}
	return config, err
}

func (c *Config) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	v, ok := c.Config[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *Config) Put(key string, val interface{}) {
	c.mu.Lock()
	c.Config[key] = val
	c.mu.Unlock()
}

// SyncWithDefaults will use the map to make up for all value present in default but not in file.
//
// Warning: SyncWithDefaults only does shallow copies of values in default (take care about race conditions)
/*func (c *Config) SyncWithDefaults(defaults map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range defaults {
		if _, ok := c.Config[k]; !ok {
			c.Config[k] = v
		}
	}
}*/

// Warning: GetCopyOfConfig only does shallow copies of values in default (take care about race conditions)
func (c *Config) GetCopyOfConfig() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	var m = map[string]interface{}{}
	for k, v := range c.Config {
		m[k] = v
	}
	return m
}

func (c *Config) SaveFile() error {
	f, err := os.Create(c.Filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.Type {
	case Json:
		err = writeFileLine(f, c.Config)
	case Line:
		err = writeFileLine(f, c.Config)
	default:
		c.Type = Json
		err = writeFileJSON(f, c.Config)
	}
	return err
}

type Getable interface {
	~string | ~float64 | []interface{} | map[string]interface{}
}

func Get[T Getable](config *Config, key string) (T, bool) {
	var ret T
	v, ok := config.Get(key)
	if !ok {
		return ret, false
	}

	ret, ok = v.(T)
	return ret, ok
}

// same as Get but gives more detail about what failed
func GetErr[T Getable](config *Config, key string) (T, error) {
	var ret T
	v, ok := config.Get(key)
	if !ok {
		return ret, ErrKeyNotFound
	}
	ret, ok = v.(T)
	if !ok {
		return ret, fmt.Errorf("config file Get: failed to cast value (wanted type: %T but got type: %T)", ret, v)
	}
	return ret, nil
}

func parseFileJSON(f *os.File) (map[string]interface{}, error) {
	var m = map[string]interface{}{}
	e := json.NewDecoder(f).Decode(&m)
	return m, e
}

func parseFileLine(f *os.File) (map[string]interface{}, error) {
	var m = map[string]interface{}{}
	var sc = bufio.NewScanner(f)
	for sc.Scan() {
		token := sc.Text()
		if strings.HasPrefix(token, "#") {
			continue
		}
		if sp := strings.SplitN(token, "=", 2); len(sp) == 2 {
			if f, err := strconv.ParseFloat(sp[1], 64); err == nil {
				m[sp[0]] = f
				continue
			}
			m[sp[0]] = sp[1]
		}
	}
	return m, sc.Err()
}

func writeFileJSON(f *os.File, m map[string]interface{}) error {
	return json.NewEncoder(f).Encode(m)
}

func writeFileLine(f *os.File, m map[string]interface{}) error {
	for k, v := range m {
		rt := reflect.TypeOf(v)
		switch rt.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64,
			reflect.String, reflect.Bool:
		default:
			continue
		}
		f.Write([]byte(fmt.Sprintf("%v=%v\n", k, v)))
	}
	return nil
}
