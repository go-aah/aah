// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"testing"

	"aahframework.org/test/assert"
)

func TestDefaultStandardLogger(t *testing.T) {
	SetLevel(LevelInfo)
	_ = SetPattern("%time:2006-01-02 15:04:05.000 %level:-5 %line %custom:- %message")
	Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	Debug("I would like to see this message, debug is useful for dev")
	Info("Yes, I would love to")
	Warn("Yes, yes it's an warning")
	Error("Yes, yes, yes - finally an error")
	fmt.Println()

	t.Logf("First round: %#v\n\n", Stats())

	SetLevel(LevelDebug)
	_ = SetPattern("%time:2006-01-02 15:04:05.000 %level:-5 %shortfile %line %custom:- %message")
	Tracef("I shoudn't see this msg: %v", 4)
	Debugf("I would like to see this message, debug is useful for dev: %v", 3)
	Infof("Yes, I would love to: %v", 2)
	Warnf("Yes, yes it's an warning: %v", 1)
	Errorf("Yes, yes, yes - finally an error: %v", 0)

	t.Logf("Second round: %#v\n\n", Stats())

	err := SetPattern("%level:-5 %shortfile %line %unknown")
	assert.NotNil(t, err)

	newLogger, _ := New(`receiver = "CONSOLE"; level = "DEBUG";`)
	SetOutput(newLogger)
	Info("Fresh new face ...")
}

func TestPanicDefaultStandardLogger(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()

	SetLevel(levelPanic)
	Panic("This is panic message")
}

func TestPanicfDefaultStandardLogger(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()

	SetLevel(levelPanic)
	Panicf("This is panic %v", "message from param")
}
