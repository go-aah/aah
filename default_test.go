// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestDefaultLogger(t *testing.T) {
	cfg, _ := config.ParseString(`
  log {
    pattern = "%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message"
  }
  `)
	std, _ = New(cfg)

	Print("welcome print")
	Printf("welcome printf")
	Println("welcome println")

	Trace("I shoudn't see this msg, because standard logger level is DEBUG")
	Tracef("I shoudn't see this msg, because standard logger level is DEBUG: %v", 4)

	Debug("I would like to see this message, debug is useful for dev")
	Debugf("I would like to see this message, debug is useful for %v", "dev")

	Info("Yes, I would love to see")
	Infof("Yes, I would love to %v", "see")

	Warn("Yes, yes it's an warning")
	Warnf("Yes, yes it's an %v", "warning")

	Error("Yes, yes, yes - finally an error")
	Errorf("Yes, yes, yes - %v", "finally an error")

	testStdPanic("panic", "this is panic")
	testStdPanic("panicf", "this is panicf")
	testStdPanic("panicln", "this is panicln")

}

func TestDefaultLoggerMisc(t *testing.T) {
	cfg, _ := config.ParseString("log { }")
	newStd, _ := New(cfg)
	SetDefaultLogger(newStd)
	Print("welcome 2 print")
	Printf("welcome 2 printf")
	Println("welcome 2 println")

	assert.Nil(t, SetLevel("trace"))
	assert.Nil(t, SetPattern("%level:-5 %message"))
}

func testStdPanic(method, msg string) {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()

	if method == "panic" {
		Panic(msg)
	} else if method == "panicf" {
		Panicf("%s", msg)
	} else if method == "panicln" {
		Panicln("%s", msg)
	}
}
