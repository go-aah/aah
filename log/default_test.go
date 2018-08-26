// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"os"
	"testing"

	"aahframe.work/aah/config"
	"github.com/stretchr/testify/assert"
)

func TestDefaultLogger(t *testing.T) {
	err := dl.SetPattern("%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message")
	assert.Nil(t, err)

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

	assert.Equal(t, "DEBUG", Level())
	assert.True(t, IsLevelDebug())
	assert.False(t, IsLevelError())
	assert.False(t, IsLevelInfo())
	assert.False(t, IsLevelWarn())
	assert.False(t, IsLevelTrace())
	assert.False(t, IsLevelFatal())
	assert.False(t, IsLevelPanic())

	exit = func(code int) {}
	Fatal("fatal msg 1")
	Fatalln("fatal msg", 2)
	Fatalf("fatal msg %v", 3)
	exit = os.Exit
}

func TestDefaultLoggerMisc(t *testing.T) {
	cfg, _ := config.ParseString("log { }")
	newStd, _ := New(cfg)
	SetDefaultLogger(newStd)
	Print("welcome 2 print")
	Printf("welcome 2 printf")
	Println("welcome 2 println")

	assert.Equal(t, "DEBUG", newStd.Level())
	assert.Nil(t, SetLevel("trace"))
	assert.Nil(t, SetPattern("%level:-5 %message"))
}

func TestDefaultContextLogging(t *testing.T) {
	_ = SetPattern("%utctime:2006-01-02 15:04:05.000 %level:-5 %longfile %line %custom:- %message %fields")
	_ = SetLevel("trace")

	AddContext(Fields{"myname": "default logger"})

	Trace("I would like to see this message, trace is more fine grained for dev")
	Tracef("I would like to see this message, trace is more fine grained for dev: %v", 4)

	Debug("I would like to see this message, debug is useful for dev")
	Debugf("I would like to see this message, debug is useful for %v", "dev")

	Info("Yes, I would love to see")
	Infof("Yes, I would love to %v", "see")

	WithField("warnkey1", "warn value 1").WithField("warnkey2", "warn value 2").Warn("Yes, yes it's an warning")
	Warnf("Yes, yes it's an %v", "warning")

	Error("Yes, yes, yes - finally an error")
	Errorf("Yes, yes, yes - %v", "finally an error")

	exit = func(code int) {}
	Fatal("Yes, yes, yes - at last fatal")
	Fatalf("Yes, yes, yes - %v", "at last fatal")
	Fatalln("Yes, yes, yes ", "at last fatal")
	exit = os.Exit

	// With Context
	cfg, _ := config.ParseString("log { }")
	ctx, _ := NewWithContext(cfg, Fields{
		"myname": "logger with context",
		"key1":   "key 1 value",
		"key2":   "key 2 value",
	})

	ctx.Trace("I would like to see this message, trace is more fine grained for dev")
	ctx.Tracef("I would like to see this message, trace is more fine grained for dev: %v", 4)

	ctx.WithField("key3", "key 3 value").Debug("I would like to see this message, debug is useful for dev")
	ctx.Debugf("I would like to see this message, debug is useful for %v", "dev")

	ctx.Info("Yes, I would love to see")
	ctx.Infof("Yes, I would love to %v", "see")

	ctx.Warn("Yes, yes it's an warning")
	ctx.Warnf("Yes, yes it's an %v", "warning")

	ctx.Error("Yes, yes, yes - finally an error")
	ctx.Errorf("Yes, yes, yes - %v", "finally an error")

	exit = func(code int) {}
	ctx.Fatal("Yes, yes, yes - at last fatal")
	ctx.Fatalf("Yes, yes, yes - %v", "at last fatal")
	ctx.Fatalln("Yes, yes, yes ", "at last fatal")
	exit = os.Exit

	ctx2 := dl.New(Fields{"ctx2": "ctx 2 value"})
	ctx2.Print("hi fields")
}

func TestDefaultFieldsLogging(t *testing.T) {
	_ = SetPattern("%time:2006-01-02 15:04:05.000 %level:-5 %appname %reqid %principal %message %fields")

	e1 := WithFields(Fields{"appname": "value1", "key1": "value 1"})
	e1.Info("e1 logger")

	e2 := e1.WithField("key2", "value 2")
	e2.Info("e2 logger")

	e3 := e1.WithFields(Fields{"key3": "value 3"})
	e3.Info("e3 logger")

	old := Writer()
	SetWriter(old)
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
		Panicln(msg)
	}
}
