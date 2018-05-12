// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/config source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package config is nice and handy layer built around `forge` config syntax;
// which is similar to HOCON syntax. Internally `aah/config` uses `forge`
// syntax developed by `https://github.com/brettlangdon`.
//
// aah framework is powered with `aahframework.org/config` library.
package config

import (
	"errors"
	"fmt"
	"strings"

	"aahframework.org/forge.v0"
	"aahframework.org/vfs.v0"
)

var errKeyNotFound = errors.New("config: not found")

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// NewEmpty method returns aah empty config instance.
func NewEmpty() *Config {
	cfg, _ := ParseString("")
	return cfg
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Config type and methods
//______________________________________________________________________________

// Config handles the configuration values and enables environment profile's,
// merge, etc. Also it provide nice and handly methods for accessing config values.
// Internally `aah config` uses `forge syntax` developed by `https://github.com/brettlangdon`.
type Config struct {
	profile string
	cfg     *forge.Section
}

// Profile returns current active profile
func (c *Config) Profile() string {
	return c.profile
}

// SetProfile actives the configuarion profile if found otherwise returns error
func (c *Config) SetProfile(profile string) error {
	if !c.HasProfile(profile) {
		return fmt.Errorf("profile doesn't exists: %v", profile)
	}

	c.profile = profile

	return nil
}

// ClearProfile clears currently active configuration `Profile`
func (c *Config) ClearProfile() {
	c.profile = ""
}

// HasProfile checks given configuration profile is exists or not
func (c *Config) HasProfile(profile string) bool {
	cfg, found := c.getraw(profile)
	return found && cfg.GetType() == forge.SECTION
}

// IsProfileEnabled returns true of profile enabled otherwise false
func (c *Config) IsProfileEnabled() bool {
	if c == nil {
		return false
	}
	return len(c.profile) > 0
}

// Keys returns all the key names at current level
func (c *Config) Keys() []string {
	return c.cfg.Keys()
}

// GetSubConfig create new sub config from the given key path. Only `Section`
// type can be created as sub config. Profile value is not propagated to sub config.
func (c *Config) GetSubConfig(key string) (*Config, bool) {
	v, err := c.cfg.Resolve(c.prepareKey(key))
	if err != nil {
		return nil, false
	}

	if s, ok := v.(*forge.Section); ok {
		return &Config{cfg: s}, true
	}
	return nil, false
}

// KeysByPath is similar to `Config.Keys()`, however it returns key names for
// given key path.
func (c *Config) KeysByPath(path string) []string {
	v, err := c.cfg.Resolve(path)
	if err != nil {
		return []string{}
	}

	if s, ok := v.(*forge.Section); ok {
		return s.Keys()
	}
	return []string{}
}

// String gets the `string` value for the given key from the configuration.
func (c *Config) String(key string) (string, bool) {
	if value, found := c.Get(key); found {
		return value.(string), found
	}

	return "", false
}

// StringDefault gets the `string` value for the given key from the configuration.
// If key does not exists it returns default value.
func (c *Config) StringDefault(key, defaultValue string) string {
	if value, found := c.String(key); found {
		return value
	}

	return defaultValue
}

// Bool gets the `bool` value for the given key from the configuration.
func (c *Config) Bool(key string) (bool, bool) {
	if value, found := c.Get(key); found {
		return value.(bool), found
	}

	return false, false
}

// BoolDefault gets the `bool` value for the given key from the configuration.
// If key does not exists it returns default value.
func (c *Config) BoolDefault(key string, defaultValue bool) bool {
	if value, found := c.Bool(key); found {
		return value
	}

	return defaultValue
}

// Int gets the `int` value for the given key from the configuration.
func (c *Config) Int(key string) (int, bool) {
	if value, found := c.Get(key); found {
		return int(value.(int64)), found
	}

	return 0, false
}

// Int64 gets the `int64` value for the given key from the configuration.
func (c *Config) Int64(key string) (int64, bool) {
	if value, found := c.Get(key); found {
		return value.(int64), found
	}

	return int64(0), false
}

// IntDefault gets the `int` value for the given key from the configuration.
// If key does not exists it returns default value.
func (c *Config) IntDefault(key string, defaultValue int) int {
	if value, found := c.Int(key); found {
		return value
	}

	return defaultValue
}

// Float32 gets the `float32` value for the given key from the configuration.
func (c *Config) Float32(key string) (float32, bool) {
	if value, found := c.Get(key); found {
		return float32(value.(float64)), found
	}

	return float32(0.0), false
}

// Float32Default gets the `float32` value for the given key from the configuration.
// If key does not exists it returns default value.
func (c *Config) Float32Default(key string, defaultValue float32) float32 {
	if value, found := c.Float32(key); found {
		return value
	}

	return defaultValue
}

// Float64 gets the `float64` value for the given key from the configuration.
func (c *Config) Float64(key string) (float64, bool) {
	if value, found := c.Get(key); found {
		return value.(float64), found
	}

	return float64(0.0), false
}

// Get gets the value from configuration returns as `interface{}`.
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) Get(key string) (interface{}, bool) {
	if c.IsProfileEnabled() {
		if value, found := c.getByProfile(key); found {
			return value, found
		}
	}

	return c.get(key)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// List methods
//______________________________________________________________________________

// StringList method returns the string slice value for the given key.
// 		Eaxmple:-
//
// 		Config:
// 			...
// 			excludes = ["*_test.go", ".*", "*.bak", "*.tmp", "vendor"]
// 			...
//
// 		Accessing Values:
// 			values, found := cfg.StringList("excludes")
// 			fmt.Println("Found:", found)
// 			fmt.Println("Values:", strings.Join(values, ", "))
//
// 		Output:
// 			Found: true
// 			Values: *_test.go, .*, *.bak, *.tmp, vendor
//
func (c *Config) StringList(key string) ([]string, bool) {
	values := []string{}
	if lst, found := c.getListValue(key); found {
		for idx := 0; idx < lst.Length(); idx++ {
			if v, err := lst.GetString(idx); err == nil {
				values = append(values, v)
			}
		}
		return values, found
	}
	return values, false
}

// IntList method returns the int slice value for the given key.
// 		Eaxmple:-
//
// 		Config:
// 			...
// 			int_list = [10, 20, 30, 40, 50]
// 			...
//
// 		Accessing Values:
// 			values, found := cfg.IntList("int_list")
// 			fmt.Println("Found:", found)
// 			fmt.Println("Values:", values)
//
// 		Output:
// 			Found: true
// 			Values: [10, 20, 30, 40, 50]
//
func (c *Config) IntList(key string) ([]int, bool) {
	var result []int
	values, found := c.Int64List(key)
	if !found {
		return result, found
	}

	for _, v := range values {
		result = append(result, int(v))
	}
	return result, true
}

// Int64List method returns the int64 slice value for the given key.
// 		Eaxmple:-
//
// 		Config:
// 			...
// 			int64_list = [100000001, 100000002, 100000003, 100000004, 100000005]
// 			...
//
// 		Accessing Values:
// 			values, found := cfg.Int64List("excludes")
// 			fmt.Println("Found:", found)
// 			fmt.Println("Values:", values)
//
// 		Output:
// 			Found: true
// 			Values: [100000001, 100000002, 100000003, 100000004, 100000005]
//
func (c *Config) Int64List(key string) ([]int64, bool) {
	values := []int64{}
	lst, found := c.getListValue(key)
	if lst == nil || !found {
		return values, found
	}

	for idx := 0; idx < lst.Length(); idx++ {
		if v, err := lst.GetInteger(idx); err == nil {
			values = append(values, v)
		}
	}

	return values, true
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Config Setter methods
//______________________________________________________________________________

// SetString sets the given value string for config key
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) SetString(key string, value string) {
	if err := c.updateValue(key, value); err == errKeyNotFound {
		c.addValue(key, forge.NewString(value))
	}
}

// SetInt sets the given value int for config key
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) SetInt(key string, value int) {
	c.SetInt64(key, int64(value))
}

// SetInt64 sets the given value int64 for config key
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) SetInt64(key string, value int64) {
	if err := c.updateValue(key, value); err == errKeyNotFound {
		c.addValue(key, forge.NewInteger(value))
	}
}

// SetFloat32 sets the given value float32 for config key
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) SetFloat32(key string, value float32) {
	c.SetFloat64(key, float64(value))
}

// SetFloat64 sets the given value float64 for config key
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) SetFloat64(key string, value float64) {
	if err := c.updateValue(key, value); err == errKeyNotFound {
		c.addValue(key, forge.NewFloat(value))
	}
}

// SetBool sets the given value bool for config key
// First it tries to get value within enabled profile
// otherwise it tries without profile
func (c *Config) SetBool(key string, value bool) {
	if err := c.updateValue(key, value); err == errKeyNotFound {
		c.addValue(key, forge.NewBoolean(value))
	}
}

// Merge merges the given section to current section. Settings from source
// section overwites the values in the current section
func (c *Config) Merge(source *Config) error {
	if source == nil {
		return errors.New("source is nil")
	}
	return c.cfg.Merge(source.cfg)
}

// IsExists returns true if given is exists in the config otherwise returns false
func (c *Config) IsExists(key string) bool {
	_, found := c.Get(key)
	return found
}

// ToJSON method returns the configuration values as JSON string.
func (c *Config) ToJSON() string {
	if b, err := c.cfg.ToJSON(); err == nil {
		return string(b)
	}
	return "{}"
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Config load/parse methods
//______________________________________________________________________________

// LoadFile loads the configuration from given config file.
func LoadFile(file string) (*Config, error) {
	return VFSLoadFile(nil, file)
}

// VFSLoadFile loads the configuration from given vfs and config file.
func VFSLoadFile(fs *vfs.VFS, file string) (*Config, error) {
	setting, err := loadFile(fs, file)
	return &Config{cfg: setting}, err
}

// LoadFiles loads the configuration from given config files.
// It does merging of configuration in the order they are given.
func LoadFiles(files ...string) (*Config, error) {
	return VFSLoadFiles(nil, files...)
}

// VFSLoadFiles loads the configuration from given config vfs and files.
// It does merging of configuration in the order they are given.
func VFSLoadFiles(fs *vfs.VFS, files ...string) (*Config, error) {
	settings := forge.NewSection()
	for _, file := range files {
		setting, err := loadFile(fs, file)
		if err != nil {
			return nil, err
		}

		if err = settings.Merge(setting); err != nil {
			return nil, err
		}
	}

	return &Config{cfg: settings}, nil
}

// ParseString parses the configuration values from string
func ParseString(cfg string) (*Config, error) {
	setting, err := forge.ParseString(cfg)
	if err != nil {
		return nil, err
	}
	return &Config{cfg: setting}, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Config unexported methods
//______________________________________________________________________________

func loadFile(fs *vfs.VFS, file string) (*forge.Section, error) {
	if _, err := vfs.Stat(fs, file); err != nil {
		return nil, fmt.Errorf("configuration does not exists: %v", file)
	}
	return forge.VFSParseFile(fs, file)
}

func (c *Config) prepareKey(key string) string {
	if c.IsProfileEnabled() {
		return fmt.Sprintf("%s.%s", c.profile, key)
	}
	return key
}

func (c *Config) getByProfile(key string) (interface{}, bool) {
	return c.get(c.prepareKey(key))
}

func (c *Config) get(key string) (interface{}, bool) {
	if v, found := c.getraw(key); found {
		return v.GetValue(), true // found
	}
	return nil, false // not found
}

func (c *Config) getListValue(key string) (*forge.List, bool) {
	value, found := c.getraw(c.prepareKey(key))
	if !found {
		value, found = c.getraw(key)
		if !found {
			return nil, found
		}
	}

	if value.GetType() != forge.LIST {
		return nil, false
	}

	return value.(*forge.List), true
}

func (c *Config) getraw(key string) (forge.Value, bool) {
	if c == nil || c.cfg == nil {
		return nil, false
	}

	v, err := c.cfg.Resolve(key)
	if err != nil {
		return nil, false // not found
	}
	return v, true // found
}

func (c *Config) updateValue(key string, value interface{}) error {
	if v, found := c.getraw(c.prepareKey(key)); found {
		_ = v.UpdateValue(value)
	}
	return errKeyNotFound
}

func (c *Config) getSection(parts []string) *forge.Section {
	current := c.cfg
	for _, part := range parts {
		if nc, err := current.GetSection(part); err == nil { // exists
			current = nc
			continue
		}
		current = current.AddSection(part)
	}
	return current
}

func (c *Config) addValue(key string, value forge.Value) {
	parts := strings.Split(c.prepareKey(key), ".")
	if len(parts) > 1 {
		section := c.getSection(parts[:len(parts)-1])
		section.Set(parts[len(parts)-1], value)
	}
}
