// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-aah/test/assert"
)

func TestNewCustomUTCConsoleReceiver(t *testing.T) {
	config := `
# console logger configuration
# "CONSOLE" uppercasse works too
receiver = "console"

# "debug" lowercase works too and if not supplied then defaults to DEBUG
level = "debug"

# if not suppiled then default pattern is used
pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %line %shortfile:-25 %custom:- %message"
 `
	logger, err := New(config)
	assert.FailNowOnError(t, err, "unexpected error")

	logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	logger.Debug("I would like to see this message, debug is useful for dev")
	logger.Info("Yes, I would love to")
	logger.Warn("Yes, yes it's an warning")
	logger.Error("Yes, yes, yes - finally an error")

	stats := logger.Stats()
	t.Logf("First round: %#v\n", stats)
	assert.Equal(t, int64(433), stats.bytes)

	logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
	logger.Debugf("I would like to see this message, debug is useful for dev: %v", 3)
	logger.Infof("Yes, I would love to: %v", 2)
	logger.Warnf("Yes, yes it's an warning: %v", 1)
	logger.Errorf("Yes, yes, yes - finally an error: %v", 0)

	stats = logger.Stats()
	t.Logf("Second round: %#v\n", stats)
	assert.Equal(t, int64(878), stats.bytes)

	Tracef("I shoudn't see this msg: %v", 46583)
	Debugf("I would like to see this message, debug is useful for dev: %v", 334545)

	stats = logger.Stats()
	t.Logf("Third round: %#v\n", stats)
	assert.Equal(t, int64(878), stats.bytes)
}

func TestNewCustomConsoleReceiver(t *testing.T) {
	config := `
# console logger configuration
# "CONSOLE" uppercasse works too
receiver = "CONSOLE"
 `
	logger, err := New(config)
	assert.FailNowOnError(t, err, "unexpected error")

	logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	logger.Debug("I would like to see this message, debug is useful for dev")
	logger.Info("Yes, I would love to")
	logger.Warn("Yes, yes it's an warning")
	logger.Error("Yes, yes, yes - finally an error")

	stats := logger.Stats()
	t.Logf("First round: %#v\n", stats)
	assert.Equal(t, int64(313), stats.bytes)

	logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
	logger.Debugf("I would like to see this message, debug is useful for dev: %v", 3)
	logger.Infof("Yes, I would love to: %v", 2)
	logger.Warnf("Yes, yes it's an warning: %v", 1)
	logger.Errorf("Yes, yes, yes - finally an error: %v", 0)

	stats = logger.Stats()
	t.Logf("Second round: %#v\n", stats)
	assert.Equal(t, int64(638), stats.bytes)

	Tracef("I shoudn't see this msg: %v", 46583)
	Debugf("I would like to see this message, debug is useful for dev: %v", 334545)

	stats = logger.Stats()
	t.Logf("Third round: %#v\n", stats)
	assert.Equal(t, int64(638), stats.bytes)
}

func TestNewCustomFileReceiverDailyRotation(t *testing.T) {
	defer cleaupFiles("*.log")

	fileLoggerConfig := `
# file logger configuration
# "FILE" uppercasse works too
receiver = "file"

# "debug" lowercase works too and if not supplied then defaults to DEBUG
level = "info"

# if not suppiled then default pattern is used
pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"

file = "daily-aah-filename.log"

rotate {
	mode = "daily"
}
 `

	logger, err := New(fileLoggerConfig)
	assert.FailNowOnError(t, err, "unexpected error")

	for i := 0; i < 25; i++ {
		logger.Trace("I shoudn't see this msg, because standard logger level is DEBUG")
		logger.Debug("I would like to see this message, debug is useful for dev")
		logger.Info("Yes, I would love to")
		logger.Warn("Yes, yes it's an warning")
		logger.Error("Yes, yes, yes - finally an error")
	}

	_ = logger.SetPattern("%time:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message")
	for i := 0; i < 25; i++ {
		logger.Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)
		logger.Debugf("I would like to see this message, debug is useful for dev: %v", 3)
		logger.Infof("Yes, I would love to: %v", 2)
		logger.Warnf("Yes, yes it's an warning: %v", 1)
		logger.Errorf("Yes, yes, yes - finally an error: %v", 0)
	}

	// Close scenario
	logger.Close()
	assert.Equal(t, true, logger.Closed())

	logger.Info("This won't be written to file")

	// once again
	logger.Close()
}

func TestNewCustomFileReceiverLinesRotation(t *testing.T) {
	defer cleaupFiles("*.log")

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
	assert.FailNowOnError(t, err, "unexpected error")

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
	defer cleaupFiles("*.log")

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
	assert.FailNowOnError(t, err, "unexpected error")

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
	assert.Equal(t, ErrFormatStringEmpty, err)

	_, err = parseFlag("%time:2006-01-02 15:04:05.000 %level:-5 %longfile %unknown %custom:- %message")
	if !strings.Contains(err.Error(), "unrecognized log format flag") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}

func TestNewMisc(t *testing.T) {
	_, err := New("")
	assert.Equal(t, "logger config is empty", err.Error())

	_, err = New(`receiver = "file" level="info"`)
	if !strings.HasPrefix(err.Error(), "syntax error") {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = New(`level="info";`)
	if !strings.HasPrefix(err.Error(), "receiver configuration") {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = New(`receiver = "file"; level="unknown";`)
	if !strings.HasPrefix(err.Error(), "unrecognized log level") {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = New(`receiver = "remote"; level="debug";`)
	if !strings.HasPrefix(err.Error(), "unsupported receiver") {
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}

	_, err = New(`receiver = "file"; level="debug"; rotate { mode="size"; size=2500; }`)
	if !strings.HasPrefix(err.Error(), "maximum 2GB file size") {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = New(`receiver = "console"; level="debug"; pattern="%time:2006-01-02 15:04:05.000 %level:-5 %unknown %message";`)
	if !strings.HasPrefix(err.Error(), "unrecognized log format flag") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestLevelUnknown(t *testing.T) {
	var level Level
	assert.Equal(t, "FATAL", level.String())

	level = 9 // Unknown log level
	assert.Equal(t, "Unknown", level.String())
}

func TestStats(t *testing.T) {
	stats := ReceiverStats{
		lines: 200,
		bytes: 764736,
	}

	assert.Equal(t, int64(764736), stats.Bytes())
	assert.Equal(t, int64(200), stats.Lines())
}

func cleaupFiles(match string) {
	pwd, _ := os.Getwd()

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
