// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"aahframe.work/aah/config"
	"aahframe.work/aah/log"
)

// Cache errors
var (
	ErrEntryExists = errors.New("aah/cache: entry exists")
)

// EvictionMode for cache entries.
type EvictionMode uint8

// Eviction modes
const (
	EvictionModeTTL EvictionMode = 1 + iota
	EvictionModeNoTTL
	EvictionModeSlide
)

// Cache interface represents operation methods for cache store.
type Cache interface {
	// Name method returns the cache store name.
	Name() string

	// Get method returns the cached entry for given key if it exists otherwise nil.
	Get(k string) interface{}

	// GetOrPut method returns the cached entry for the given key if it exists otherwise
	// it puts the new entry into cache store and returns the value.
	GetOrPut(k string, v interface{}, d time.Duration) (interface{}, error)

	// Put method adds the cache entry with specified expiration. Returns error
	// if cache entry exists.
	Put(k string, v interface{}, d time.Duration) error

	// Delete method deletes the cache entry from cache store.
	Delete(k string) error

	// Exists method checks given key exists in cache store and its not expried.
	Exists(k string) bool

	// Flush methods flushes(deletes) all the cache entries from cache.
	Flush() error
}

// Provider interface represents cache provider implementation.
type Provider interface {
	// Init method invoked by aah cache manager on application start to initialize cache provider.
	Init(name string, appCfg *config.Config, logger log.Loggerer) error

	// Create method invoked by aah cache manager to create cache specific to provider.
	Create(cfg *Config) (Cache, error)
}

// Config struct represents the cache and cache provider configurations.
type Config struct {
	Name         string
	ProviderName string
	EvictionMode EvictionMode

	// SweepInterval only applicable to in-memory cache provider.
	SweepInterval time.Duration
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package exported methods
//______________________________________________________________________________

// NewManager method returns new cache Manager.
func NewManager() *Manager {
	m := &Manager{
		mu:        sync.RWMutex{},
		caches:    make(map[string]Cache),
		providers: make(map[string]Provider),
	}
	return m
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Manager type and its exported methods
//______________________________________________________________________________

// Manager struct represents the aah cache manager.
type Manager struct {
	mu        sync.RWMutex
	caches    map[string]Cache
	providers map[string]Provider
}

// AddProvider method adds given provider by name. If provider name exists
// it return an error otherwise nil.
func (m *Manager) AddProvider(name string, provider Provider) error {
	m.mu.Lock()
	if _, f := m.providers[name]; f {
		m.mu.Unlock()
		return fmt.Errorf("aah/cache: provider '%s' exists", name)
	}
	m.providers[name] = provider
	m.mu.Unlock()
	return nil
}

// InitProviders method initializes the cache providers.
func (m *Manager) InitProviders(appCfg *config.Config, logger log.Loggerer) error {
	m.mu.Lock()
	for n, p := range m.providers {
		if err := p.Init(n, appCfg, logger); err != nil {
			m.mu.Unlock()
			return err
		}
	}
	m.mu.Unlock()
	return nil
}

// Provider method returns provider by given name if exists otherwise nil.
func (m *Manager) Provider(name string) Provider {
	m.mu.RLock()
	p := m.providers[name]
	m.mu.RUnlock()
	return p
}

// ProviderNames returns all provider names from cache manager.
func (m *Manager) ProviderNames() []string {
	var names []string
	m.mu.RLock()
	for n := range m.providers {
		names = append(names, n)
	}
	m.mu.RUnlock()
	return names
}

// CreateCache method creates new cache in the cache manager for
// configuration.
func (m *Manager) CreateCache(cfg *Config) error {
	if len(cfg.Name) == 0 || len(cfg.ProviderName) == 0 {
		return fmt.Errorf("aah/cache: name and provider name is required")
	}
	if cfg.EvictionMode == 0 {
		cfg.EvictionMode = EvictionModeTTL
	}
	if cfg.SweepInterval == 0 {
		cfg.SweepInterval = 60 * time.Minute
	}

	p := m.Provider(cfg.ProviderName)
	if p == nil {
		return fmt.Errorf("aah/cache: provider '%s' not exists", cfg.ProviderName)
	}
	c, err := p.Create(cfg)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.caches[cfg.Name] = c
	m.mu.Unlock()

	return nil
}

// Cache method return cache by given name if exists otherwise nil.
func (m *Manager) Cache(name string) Cache {
	m.mu.RLock()
	c := m.caches[name]
	m.mu.RUnlock()
	return c
}

// CacheNames method returns all cache names from cache manager.
func (m *Manager) CacheNames() []string {
	var names []string
	m.mu.RLock()
	for n := range m.caches {
		names = append(names, n)
	}
	m.mu.RUnlock()
	return names
}
