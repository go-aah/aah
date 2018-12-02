// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"aahframe.work/config"
	"aahframe.work/essentials"
	"aahframe.work/log"
	"github.com/stretchr/testify/assert"
)

func TestCacheManager(t *testing.T) {
	mgr := NewManager()

	t.Log("Adding new providers")
	provider1 := &dummyProvider{name: "provider1"}
	provider2 := &dummyProvider{name: "provider2"}
	provider3 := &dummyProvider{name: "provider3"}
	assert.Nil(t, mgr.AddProvider("provider1", provider1))
	assert.Nil(t, mgr.AddProvider("provider2", provider2))
	assert.Nil(t, mgr.AddProvider("provider3", provider3))

	t.Log("Init Providers")
	l, _ := log.New(config.NewEmpty())
	l.SetWriter(ioutil.Discard)
	err := mgr.InitProviders(config.NewEmpty(), l)
	assert.Nil(t, err)

	t.Log("Get Provider")
	p := mgr.Provider("provider2")
	assert.NotNil(t, p)

	t.Log("Get Provider names")
	providerNames := mgr.ProviderNames()
	assert.Equal(t, 3, len(providerNames))
	assert.True(t, ess.IsSliceContainsString(providerNames, "provider3"))

	t.Log("Create cache using provider")
	err = mgr.CreateCache(&Config{Name: "cache1", ProviderName: "provider1"})
	assert.Nil(t, err)
	err = mgr.CreateCache(&Config{Name: "cache3", ProviderName: "provider3"})
	assert.Nil(t, err)

	t.Log("Get cache names")
	cacheNames := mgr.CacheNames()
	assert.Equal(t, 2, len(cacheNames))
	assert.True(t, ess.IsSliceContainsString(cacheNames, "cache1"))

	t.Log("Get one cache")
	c := mgr.Cache("cache3")
	assert.Equal(t, "cache3", c.Name())
}

func TestCacheManagerValidations(t *testing.T) {
	mgr := NewManager()

	err := mgr.AddProvider("provider1", &dummyProvider2{})
	assert.Nil(t, err)

	t.Log("Adding already exists provider")
	err = mgr.AddProvider("provider1", &dummyProvider2{})
	assert.Equal(t, errors.New("aah/cache: provider 'provider1' exists"), err)

	t.Log("Init Providers error")
	l, _ := log.New(config.NewEmpty())
	l.SetWriter(ioutil.Discard)
	err = mgr.InitProviders(config.NewEmpty(), l)
	assert.Equal(t, errors.New("aah/cache: provider provider1 init error"), err)

	t.Log("Create cache error")
	err = mgr.CreateCache(&Config{Name: "cache1", ProviderName: "provider1"})
	assert.Equal(t, errors.New("aah/cache: cache create error for cache1"), err)

	t.Log("Create cache provider not exists")
	err = mgr.CreateCache(&Config{Name: "cache1", ProviderName: "provider11"})
	assert.Equal(t, errors.New("aah/cache: provider 'provider11' not exists"), err)

	t.Log("Required config error")
	err = mgr.CreateCache(&Config{})
	assert.Equal(t, errors.New("aah/cache: name and provider name is required"), err)
}

type dummyProvider struct {
	name string
}

var _ Provider = (*dummyProvider)(nil)

// Init method is not applicable for in-memory cache provider.
func (p *dummyProvider) Init(name string, _ *config.Config, _ log.Loggerer) error { return nil }

// Create method creates new in-memory cache with given options.
func (p *dummyProvider) Create(cfg *Config) (Cache, error) {
	return &dummyCache{p: p, cfg: cfg}, nil
}

type dummyProvider2 struct{}

var _ Provider = (*dummyProvider2)(nil)

// Init method is not applicable for in-memory cache provider.
func (p *dummyProvider2) Init(name string, _ *config.Config, _ log.Loggerer) error {
	return fmt.Errorf("aah/cache: provider %s init error", name)
}

// Create method creates new in-memory cache with given options.
func (p *dummyProvider2) Create(cfg *Config) (Cache, error) {
	return nil, fmt.Errorf("aah/cache: cache create error for %s", cfg.Name)
}

type dummyCache struct {
	p   *dummyProvider
	cfg *Config
}

var _ Cache = (*dummyCache)(nil)

// Name method returns the cache store name.
func (c *dummyCache) Name() string { return c.cfg.Name }

// Get method returns the cached entry for given key if it exists otherwise nil.
func (c *dummyCache) Get(k string) interface{} { return nil }

// GetOrPut method returns the cached entry for given key if it exists otherwise
// it adds the entry into cache store and returns the value.
func (c *dummyCache) GetOrPut(k string, v interface{}, d time.Duration) (interface{}, error) {
	return nil, nil
}

// Put method adds the cache entry with specified expiration. Returns error
// if cache entry exists.
func (c *dummyCache) Put(k string, v interface{}, d time.Duration) error { return nil }

// Delete method deletes the cache entry from cache store.
func (c *dummyCache) Delete(k string) error { return nil }

// Exists method checks given key exists in cache store and its not expried.
func (c *dummyCache) Exists(k string) bool { return false }

func (c *dummyCache) Flush() error { return nil }
