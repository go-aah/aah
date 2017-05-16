// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"os"

	"aahframework.org/config.v0"
)

var std *Logger

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods
//_______________________________________

// Error logs message as `ERROR`. Arguments handled in the mananer of `fmt.Print`.
func Error(v ...interface{}) {
	std.output(LevelError, 3, nil, v...)
}

// Errorf logs message as `ERROR`. Arguments handled in the mananer of `fmt.Printf`.
func Errorf(format string, v ...interface{}) {
	std.output(LevelError, 3, &format, v...)
}

// Warn logs message as `WARN`. Arguments handled in the mananer of `fmt.Print`.
func Warn(v ...interface{}) {
	std.output(LevelWarn, 3, nil, v...)
}

// Warnf logs message as `WARN`. Arguments handled in the mananer of `fmt.Printf`.
func Warnf(format string, v ...interface{}) {
	std.output(LevelWarn, 3, &format, v...)
}

// Info logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func Info(v ...interface{}) {
	std.output(LevelInfo, 3, nil, v...)
}

// Infof logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func Infof(format string, v ...interface{}) {
	std.output(LevelInfo, 3, &format, v...)
}

// Debug logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Print`.
func Debug(v ...interface{}) {
	std.output(LevelDebug, 3, nil, v...)
}

// Debugf logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Printf`.
func Debugf(format string, v ...interface{}) {
	std.output(LevelDebug, 3, &format, v...)
}

// Trace logs message as `TRACE`. Arguments handled in the mananer of `fmt.Print`.
func Trace(v ...interface{}) {
	std.output(LevelTrace, 3, nil, v...)
}

// Tracef logs message as `TRACE`. Arguments handled in the mananer of `fmt.Printf`.
func Tracef(format string, v ...interface{}) {
	std.output(LevelTrace, 3, &format, v...)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods - Drop-in replacement
// for Go standard logger
//_______________________________________

// Print logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func Print(v ...interface{}) {
	std.output(LevelInfo, 3, nil, v...)
}

// Printf logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func Printf(format string, v ...interface{}) {
	std.output(LevelInfo, 3, &format, v...)
}

// Println logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func Println(format string, v ...interface{}) {
	std.output(LevelInfo, 3, &format, v...)
}

// Fatal logs message as `FATAL` and call to os.Exit(1).
func Fatal(v ...interface{}) {
	std.output(levelFatal, 3, nil, v...)
	os.Exit(1)
}

// Fatalf logs message as `FATAL` and call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	std.output(levelFatal, 3, &format, v...)
	os.Exit(1)
}

// Fatalln logs message as `FATAL` and call to os.Exit(1).
func Fatalln(format string, v ...interface{}) {
	std.output(levelFatal, 3, &format, v...)
	os.Exit(1)
}

// Panic logs message as `PANIC` and call to panic().
func Panic(v ...interface{}) {
	std.output(levelPanic, 3, nil, v...)
	panic("")
}

// Panicf logs message as `PANIC` and call to panic().
func Panicf(format string, v ...interface{}) {
	std.output(levelPanic, 3, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

// Panicln logs message as `PANIC` and call to panic().
func Panicln(format string, v ...interface{}) {
	std.output(levelPanic, 3, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

// SetDefaultLogger method sets the given logger instance as default logger.
func SetDefaultLogger(l *Logger) {
	std = l
}

// SetLevel method sets log level for default logger.
func SetLevel(level string) error {
	return std.SetLevel(level)
}

// SetPattern method sets the log format pattern for default logger.
func SetPattern(pattern string) error {
	return std.SetPattern(pattern)
}

// IsBufferEmpty returns true if logger buffer is empty otherwise false.
// This method can be used to ensure all the log entry is written successfully.
func IsBufferEmpty() bool {
	return std.IsBufferEmpty()
}

func init() {
	cfg, _ := config.ParseString("log { }")
	std, _ = New(cfg)
}
