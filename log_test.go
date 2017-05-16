// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestLogDefault(t *testing.T) {
	configStr := `
  log {
    # default config
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.NotNil(t, logger)
	assert.Nil(t, err)
	assert.Equal(t, "DEBUG", logger.Level())

	logger.Print("This is print")
	logger.Printf("This is print - %s", "yes")
	logger.Println("This is print - %s", "yes")

	testPanic(logger, "panic", "this is panic")
	testPanic(logger, "panicf", "this is panicf")
	testPanic(logger, "panicln", "this is panicln")
	time.Sleep(1 * time.Millisecond)
}

func TestMisc(t *testing.T) {
	stats := receiverStats{
		lines: 200,
		bytes: 764736,
	}

	assert.Equal(t, int64(764736), stats.Bytes())
	assert.Equal(t, int64(200), stats.Lines())

	configStr := `
  log {
    # default config
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.NotNil(t, logger)
	assert.Nil(t, err)
	assert.Equal(t, "DEBUG", logger.Level())

	err = logger.SetReceiver(nil)
	assert.Equal(t, "log: receiver is nil", err.Error())

	err = logger.SetLevel("MYLEVEL")
	assert.Equal(t, "log: unknown log level 'MYLEVEL'", err.Error())

	logger, err = New(nil)
	assert.Nil(t, logger)
	assert.NotNil(t, err)
	assert.Equal(t, "log: config is nil", err.Error())

	// Discard
	discard := DiscardReceiver{}
	_ = discard.Init(nil)
	discard.Log(&Entry{})
	_ = discard.SetPattern("nothing")
	assert.False(t, discard.IsCallerInfo())

	// util
	assert.Nil(t, getReceiverByName("SMTP"))
	assert.Equal(t, "", formatTime(time.Time{}))
}

func testPanic(logger *Logger, method, msg string) {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()

	if method == "panic" {
		logger.Panic(msg)
	} else if method == "panicf" {
		logger.Panicf("%s", msg)
	} else if method == "panicln" {
		logger.Panicln("%s", msg)
	}
}

func getPwd() string {
	pwd, _ := os.Getwd()
	return pwd
}

func cleaupFiles(match string) {
	pwd := getPwd()
	dir, err := os.Open(pwd)
	if err != nil {
		return
	}

	infos, err := dir.Readdir(-1)
	if err != nil {
		return
	}

	for _, info := range infos {
		if !info.IsDir() {
			if found, _ := filepath.Match(match, info.Name()); found {
				_ = os.Remove(filepath.Join(pwd, info.Name()))
			}
		}
	}
}
