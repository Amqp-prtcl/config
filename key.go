package config

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

var (
	ErrKeyNotFound = fmt.Errorf("config file Get: key not found")
)

type Keyable interface {
	~string | ~[]interface{} | ~map[string]interface{} | ~bool |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type Key[T Keyable] string // will return zero value if key is not present or if is not parsable

type TimeKey string // will return zero value if key is not present or if is not parsable

// if key is not present in Config or cannot be converted into T, Get() return the zero value of T.
//
// conversions are made by reflect.Value.Convert(), and as a special case booleans and strings are
// automatically parse between each other
func (k Key[T]) Get(c *Config) T {
	var ret T
	v, ok := c.Get(string(k))
	if !ok {
		return ret
	}
	switch v := v.(type) {
	case T:
		return v
	case bool:
		if reflect.ValueOf(ret).Kind() == reflect.String {
			return interface{}(strconv.FormatBool(v)).(T)
		}
	case string:
		if reflect.ValueOf(ret).Kind() == reflect.Bool {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return interface{}(b).(T)
			}
		}
	}
	rv := reflect.ValueOf(v)
	if rv.CanConvert(reflect.TypeOf(ret)) { // does not support string to bool and vice versa
		return rv.Convert(reflect.TypeOf(ret)).Interface().(T)
	}
	return ret
}

func (k Key[T]) GetErr(c *Config) (T, error) {
	var ret T
	v, ok := c.Get(string(k))
	if !ok {
		return ret, ErrKeyNotFound
	}
	switch v := v.(type) {
	case T:
		return v, nil
	case bool:
		if reflect.ValueOf(ret).Kind() == reflect.String {
			return interface{}(strconv.FormatBool(v)).(T), nil
		}
	case string:
		if reflect.ValueOf(ret).Kind() == reflect.Bool {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return interface{}(b).(T), nil
			}
		}
	}
	rv := reflect.ValueOf(v)
	if rv.CanConvert(reflect.TypeOf(ret)) {
		return rv.Convert(reflect.TypeOf(ret)).Interface().(T), nil
	}
	return ret, fmt.Errorf("config file Get: failed to cast value (wanted type: %T but got type: %T)", ret, v)
}

func (k Key[T]) Put(c *Config, v T) {
	c.Put(string(k), v)
}

// if key is not present in Config or cannot be converted into T, Get() return the zero value of T
func (k TimeKey) Get(c *Config) time.Time {
	var t time.Time
	var ck = Key[string](k)
	str := ck.Get(c)
	_ = t.UnmarshalText([]byte(str))
	return t
}

func (k TimeKey) GetErr(c *Config) (time.Time, error) {
	var t time.Time
	var ck = Key[string](k)
	str, err := ck.GetErr(c)
	if err != nil {
		return t, err
	}
	err = t.UnmarshalText([]byte(str))
	return t, err
}

func (k TimeKey) Put(c *Config, v time.Time) {
	c.Put(string(k), v)
}
