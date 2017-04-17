// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"os"
)

var stdLogger Logger

// Fatal logs message as `FATAL` and calls os.Exit(1)
func Fatal(v ...interface{}) {
	_ = stdLogger.Output(levelFatal, 2, nil, v...)
	os.Exit(1)
}

// Fatalf logs message as `FATAL` and calls os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	_ = stdLogger.Output(levelFatal, 2, &format, v...)
	os.Exit(1)
}

// Panic logs message as `PANIC` and calls panic()
func Panic(v ...interface{}) {
	_ = stdLogger.Output(levelPanic, 2, nil, v...)
	panic("")
}

// Panicf logs message as `PANIC` and calls panic()
func Panicf(format string, v ...interface{}) {
	_ = stdLogger.Output(levelPanic, 2, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

// Error logs message as `LevelError`
func Error(v ...interface{}) {
	_ = stdLogger.Output(LevelError, 2, nil, v...)
}

// Errorf logs message as `LevelError`
func Errorf(format string, v ...interface{}) {
	_ = stdLogger.Output(LevelError, 2, &format, v...)
}

// Warn logs message as `LevelWarn`
func Warn(v ...interface{}) {
	_ = stdLogger.Output(LevelWarn, 2, nil, v...)
}

// Warnf logs message as `LevelWarn`
func Warnf(format string, v ...interface{}) {
	_ = stdLogger.Output(LevelWarn, 2, &format, v...)
}

// Info logs message as `LevelInfo`
func Info(v ...interface{}) {
	_ = stdLogger.Output(LevelInfo, 2, nil, v...)
}

// Infof logs message as `LevelInfo`
func Infof(format string, v ...interface{}) {
	_ = stdLogger.Output(LevelInfo, 2, &format, v...)
}

// Debug logs message as `LevelDebug`
func Debug(v ...interface{}) {
	_ = stdLogger.Output(LevelDebug, 2, nil, v...)
}

// Debugf logs message as `LevelDebug`
func Debugf(format string, v ...interface{}) {
	_ = stdLogger.Output(LevelDebug, 2, &format, v...)
}

// Trace logs message as `LevelTrace`
func Trace(v ...interface{}) {
	_ = stdLogger.Output(LevelTrace, 2, nil, v...)
}

// Tracef logs message as `LevelTrace`
func Tracef(format string, v ...interface{}) {
	_ = stdLogger.Output(LevelTrace, 2, &format, v...)
}

// Stats returns current logger statistics like number of lines written,
// number of bytes written, etc.
func Stats() *ReceiverStats {
	return stdLogger.Stats()
}

// SetPattern sets the log entry format
func SetPattern(pattern string) error {
	return stdLogger.SetPattern(pattern)
}

// SetLevel allows to set log level dynamically
func SetLevel(level Level) {
	stdLogger.SetLevel(level)
}

// SetOutput allows to set standard logger implementation
// which statisfies `Logger` interface
func SetOutput(logger Logger) {
	stdLogger = logger
}

func init() {
	stdLogger, _ = New(`receiver = "CONSOLE"; level = "DEBUG";`)
}
