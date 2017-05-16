// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestConsoleLoggerTextJSON(t *testing.T) {
	// Text 1
	textConfigStr1 := `
  log {
    receiver = "console"
    level = "debug"
    pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"
  }
  `
	testConsoleLogger(t, textConfigStr1)

	// Text 2
	textConfigStr2 := `
  log {
    receiver = "console"
    level = "debug"
    pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %shortfile %line %custom:- %message"
  }
  `
	testConsoleLogger(t, textConfigStr2)

	// JSON
	jsonConfigStr := `
  log {
    receiver = "console"
    level = "debug"
    format = "json"
  }
  `
	testConsoleLogger(t, jsonConfigStr)
}

func TestConsoleLoggerUnsupportedFormat(t *testing.T) {
	configStr := `
  log {
    # default config plus
    pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"
    format = "xml"
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.Nil(t, logger)
	assert.Equal(t, "log: unsupported format 'xml'", err.Error())
}

func TestConsoleLoggerUnknownFormatFlag(t *testing.T) {
	configStr := `
  log {
    # default config plus
    pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %myfile %line %custom:- %message"
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.Nil(t, logger)
	assert.Equal(t, "unrecognized log format flag: myfile", err.Error())
}

func TestConsoleLoggerUnknownLevel(t *testing.T) {
	configStr := `
  log {
    # default config plus
		level = "MYLEVEL"
    pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %message"
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.Nil(t, logger)
	assert.Equal(t, "log: unknown log level 'MYLEVEL'", err.Error())
}

func TestConsoleLoggerDefaults(t *testing.T) {
	configStr := `
  log {
    # default config
  }
  `
	cfg, _ := config.ParseString(configStr)
	logger, err := New(cfg)
	assert.NotNil(t, logger)
	assert.Nil(t, err)

	// receiver nil scenario
	logger.receiver = nil
	err = logger.SetPattern("%time:2006-01-02 15:04:05.000 %level:-5 %message")
	assert.Equal(t, "log: receiver is nil", err.Error())
}

func testConsoleLogger(t *testing.T, cfgStr string) {
	cfg, _ := config.ParseString(cfgStr)
	logger, err := New(cfg)
	assert.FailNowOnError(t, err, "unexpected error")

	logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)

	logger.Debug("I would like to see this message, debug is useful for dev")
	logger.Debugf("I would like to see this message, debug is useful for %v", "dev")

	logger.Info("Yes, I would love to see")
	logger.Infof("Yes, I would love to %v", "see")

	logger.Warn("Yes, yes it's an warning")
	logger.Warnf("Yes, yes it's an %v", "warning")

	logger.Error("Yes, yes, yes - finally an error")
	logger.Errorf("Yes, yes, yes - %v", "finally an error")

	waitToDrain(logger)
}
