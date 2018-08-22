// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"os"
	"testing"
	"time"

	"aahframe.work/aah/config"
	"github.com/stretchr/testify/assert"
)

func TestLogAddHook(t *testing.T) {
	err := AddHook("hook1", func(e Entry) {
		assert.NotNil(t, e)
	})
	assert.Nil(t, err)

	err = AddHook("hook2", func(e Entry) {
		assert.NotNil(t, e)
	})
	assert.Nil(t, err)

	// Already added
	err = AddHook("hook1", func(e Entry) {
		assert.NotNil(t, e)
	})
	assert.Equal(t, "log: hook name 'hook1' is already added, skip it", err.Error())

	// Nil hook
	err = AddHook("nilhook", nil)
	assert.Equal(t, ErrHookFuncIsNil, err)
}

func TestLogHook(t *testing.T) {
	configStr := `
  log {
    receiver = "console"
    level = "debug"
    pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %line %custom:- %message"
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.Nil(t, err, "unexpected error")

	// Add hook
	_ = logger.AddHook("hook1", func(e Entry) {
		assert.NotNil(t, e)
		fmt.Println(e)
	})

	logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	logger.Debug("I would like to see this message, debug is useful for dev")
	logger.Info("Yes, I would love to see")
	logger.Warn("Yes, yes it's an warning")
	logger.Error("Yes, yes, yes - finally an error")

	exit = func(code int) {}
	logger.Fatal("Yes, yes, yes - at last fatal")
	exit = os.Exit

	time.Sleep(1 * time.Millisecond)
}
