// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/config source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package config is nice and handy thin layer around `forge` config syntax;
// which is similar to HOCON syntax. aah framework is powered with `aah/config`
// library. Internally `aah/config` uses `forge` syntax developed by
// `https://github.com/brettlangdon`.
package config

import (
	"errors"
	"fmt"

	"github.com/go-aah/forge"
)

// Version no. of go-aah/config library
var Version = "0.1"

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
	_, err := c.cfg.GetSection(profile)
	return err == nil
}

// IsProfileEnabled returns true of profile enabled otherwise false
func (c *Config) IsProfileEnabled() bool {
	return len(c.profile) > 0
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

// Merge merges the given section to current section. Settings from source
// section overwites the values in the current section
func (c *Config) Merge(source *Config) error {
	if source == nil {
		return errors.New("source is nil")
	}
	return c.cfg.Merge(source.cfg)
}

func (c *Config) getByProfile(key string) (interface{}, bool) {
	return c.get(fmt.Sprintf("%s.%s", c.profile, key))
}

func (c *Config) get(key string) (interface{}, bool) {
	v, err := c.cfg.Resolve(key)
	if err != nil {
		return nil, false
	}

	return v.GetValue(), true
}

// Configuration loading methods

// LoadFile loads the configuration given config file
func LoadFile(file string) (*Config, error) {
	setting, err := forge.ParseFile(file)
	if err != nil {
		return nil, err
	}

	return &Config{
		cfg: setting,
	}, nil
}

// LoadFiles loads the configuration given config files and
// does merging of configuration in the order they are given
func LoadFiles(files ...string) (*Config, error) {
	settings := forge.NewSection()
	for _, file := range files {
		setting, err := forge.ParseFile(file)
		if err != nil {
			return nil, err
		}

		if err = settings.Merge(setting); err != nil {
			return nil, err
		}
	}

	return &Config{
		cfg: settings,
	}, nil
}

// ParseString parses the configuration values from string
func ParseString(cfg string) (*Config, error) {
	setting, err := forge.ParseString(cfg)
	if err != nil {
		return nil, err
	}

	return &Config{
		cfg: setting,
	}, nil
}
