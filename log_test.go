// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"strings"
	"testing"
)

func TestDefaultStandardLogger(t *testing.T) {
	Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	Debug("I would like to see this message, debug is useful for dev")
	Info("Yes, I would love to")
	Warn("Yes, yes it's an warning")
	Error("Yes, yes, yes - finally an error")

	fmt.Printf("First round: %#v\n\n", Stats())

	Tracef("I shoudn't see this msg: %v", 4)
	Debugf("I would like to see this message, debug is useful for dev: %v", 3)
	Infof("Yes, I would love to: %v", 2)
	Warnf("Yes, yes it's an warning: %v", 1)
	Errorf("Yes, yes, yes - finally an error: %v", 0)

	fmt.Printf("Second round: %#v\n\n", Stats())
}

func TestNewCustomUTCConsoleReceiver(t *testing.T) {
	config := `
# console logger configuration
# "CONSOLE" uppercasse works too
receiver = "console"

# "debug" lowercase works too and if not supplied then defaults to DEBUG
level = "debug"

# if not suppiled then default pattern is used
pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %shortfile:-25 %custom:- %message"
 `
	logger, err := New(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	logger.Debug("I would like to see this message, debug is useful for dev")
	logger.Info("Yes, I would love to")
	logger.Warn("Yes, yes it's an warning")
	logger.Error("Yes, yes, yes - finally an error")

	stats := logger.Stats()
	fmt.Printf("First round: %#v\n\n", stats)
	if stats.bytes != 417 {
		t.Errorf("Expected: 417, got: %v", stats.bytes)
	}

	logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
	logger.Debugf("I would like to see this message, debug is useful for dev: %v", 3)
	logger.Infof("Yes, I would love to: %v", 2)
	logger.Warnf("Yes, yes it's an warning: %v", 1)
	logger.Errorf("Yes, yes, yes - finally an error: %v", 0)

	stats = logger.Stats()
	fmt.Printf("Second round: %#v\n\n", stats)
	if stats.bytes != 846 {
		t.Errorf("Expected: 846, got: %v", stats.bytes)
	}

	Tracef("I shoudn't see this msg: %v", 46583)
	Debugf("I would like to see this message, debug is useful for dev: %v", 334545)

	stats = logger.Stats()
	fmt.Printf("Third round: %#v\n\n", stats)
	if stats.bytes != 846 {
		t.Errorf("Expected: 846, got: %v", stats.bytes)
	}
}

func TestNewCustomConsoleReceiver(t *testing.T) {
	config := `
# console logger configuration
# "CONSOLE" uppercasse works too
receiver = "CONSOLE"
 `
	logger, err := New(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	logger.Debug("I would like to see this message, debug is useful for dev")
	logger.Info("Yes, I would love to")
	logger.Warn("Yes, yes it's an warning")
	logger.Error("Yes, yes, yes - finally an error")

	stats := logger.Stats()
	fmt.Printf("First round: %#v\n\n", stats)
	if stats.bytes != 313 {
		t.Errorf("Expected: 313, got: %v", stats.bytes)
	}

	logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
	logger.Debugf("I would like to see this message, debug is useful for dev: %v", 3)
	logger.Infof("Yes, I would love to: %v", 2)
	logger.Warnf("Yes, yes it's an warning: %v", 1)
	logger.Errorf("Yes, yes, yes - finally an error: %v", 0)

	stats = logger.Stats()
	fmt.Printf("Second round: %#v\n\n", stats)
	if stats.bytes != 638 {
		t.Errorf("Expected: 638, got: %v", stats.bytes)
	}

	Tracef("I shoudn't see this msg: %v", 46583)
	Debugf("I would like to see this message, debug is useful for dev: %v", 334545)

	stats = logger.Stats()
	fmt.Printf("Third round: %#v\n\n", stats)
	if stats.bytes != 638 {
		t.Errorf("Expected: 638, got: %v", stats.bytes)
	}
}

func TestNewCustomFileReceiverLinesRotation(t *testing.T) {
	fileLoggerConfig := `
# file logger configuration
# "FILE" uppercasse works too
receiver = "file"

# "debug" lowercase works too and if not supplied then defaults to DEBUG
level = "trace"

# if not suppiled then default pattern is used
pattern = "%time:2006-01-02 15:04:05.000 %level:-5 %shortfile %line %custom:- %message"

file = "test-aah-filename.log"

rotate {
	mode = "lines"

	# this value is needed if rotate="lines"; default is unlimited
	lines = 20

	# this value is needed in MB if rotate="size"; default is unlimited
	#size = 250
}
 `

	logger, err := New(fileLoggerConfig)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	for i := 0; i < 25; i++ {
		logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
		logger.Debug("I would like to see this message, debug is useful for dev")
		logger.Info("Yes, I would love to")
		logger.Warn("Yes, yes it's an warning")
		logger.Error("Yes, yes, yes - finally an error")

		logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
		logger.Debugf("I would like to see this message, debug is useful for dev: %v", 3)
		logger.Infof("Yes, I would love to: %v", 2)
		logger.Warnf("Yes, yes it's an warning: %v", 1)
		logger.Errorf("Yes, yes, yes - finally an error: %v", 0)
	}
}

func TestNewCustomFileReceiverSizeRotation(t *testing.T) {
	fileLoggerConfig := `
# file logger configuration
# "FILE" uppercasse works too
receiver = "file"

# if not suppiled then default pattern is used
pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"
rotate {
	mode = "size"

	# this value is needed in MB if rotate="size"; default is unlimited
	size = 1
}
 `

	logger, err := New(fileLoggerConfig)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	// Size based rotation, dump more value into receiver
	for i := 0; i < 5000; i++ {
		logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
		logger.Debug("I would like to see this message, debug is useful for dev")
		logger.Info("Yes, I would love to, Yes, I would love to, Yes, I would love to, Yes, I would love to")
		logger.Warn("Yes, yes it's an warning, Yes, yes it's an warning,Yes, yes it's an warning, Yes, yes it's an warning")
		logger.Error("Yes, yes, yes - finally an error")

		logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
		logger.Debugf("I would like to see this message, debug is useful for dev: %v, %d", 3, 333)
		logger.Infof("Yes, I would love to: %v, Yes, I would love to: %v", 2, 22, 222)
		logger.Warnf("Yes, yes it's an warning: %v, Yes, yes it's an warning: %v, Yes, yes it's an warning: %v", 1, 11, 111)
		logger.Errorf("Yes, yes, yes - finally an error: %v, finally an error: %v, finally an error: %v", 0, 000, 0000)
	}
}

func TestUnknownFormatFlag(t *testing.T) {
	_, err := parseFlag("")
	if err != ErrFormatStringEmpty {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = parseFlag("%time:2006-01-02 15:04:05.000 %level:-5 %longfile %unknown %custom:- %message")
	if !strings.Contains(err.Error(), "unrecognized log format flag") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}

func TestNewMisc(t *testing.T) {
	_, err := New("")
	if err.Error() != "logger config is empty" {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`receiver = "file" level="info"`)
	if !strings.HasPrefix(err.Error(), "syntax error") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`level="info";`)
	if !strings.HasPrefix(err.Error(), "receiver configuration") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`receiver = "file"; level="unknown";`)
	if !strings.HasPrefix(err.Error(), "unrecognized log level") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`receiver = "remote"; level="debug";`)
	if !strings.HasPrefix(err.Error(), "unsupported receiver") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`receiver = "file"; level="debug"; rotate { mode="size"; size=2500; }`)
	if !strings.HasPrefix(err.Error(), "maximum 2GB file size") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`receiver = "console"; level="debug"; pattern="%time:2006-01-02 15:04:05.000 %level:-5 %unknown %message";`)
	if !strings.HasPrefix(err.Error(), "unrecognized log format flag") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}
