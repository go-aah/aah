// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrHookFuncIsNil is returned when hook function is nil.
	ErrHookFuncIsNil = errors.New("log: hook func is nil")

	hooks = make(map[string]HookFunc)
	mu    = &sync.RWMutex{}
)

// HookFunc type is aah framework logger custom hook.
type HookFunc func(e Entry)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AddHook method is to add logger hook function.
func AddHook(name string, hook HookFunc) error {
	if hook == nil {
		return ErrHookFuncIsNil
	}

	mu.Lock()
	defer mu.Unlock()
	if _, found := hooks[name]; found {
		return fmt.Errorf("log: hook name '%v' is already added, skip it", name)
	}

	hooks[name] = hook
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func executeHooks(e Entry) {
	mu.RLock()
	defer mu.RUnlock()
	for _, fn := range hooks {
		go fn(e)
	}
}
