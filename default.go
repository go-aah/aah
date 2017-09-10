// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"io"
	slog "log"

	"aahframework.org/config.v0"
)

var dl *Logger

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods
//_______________________________________

// Error logs message as `ERROR`. Arguments handled in the mananer of `fmt.Print`.
func Error(v ...interface{}) {
	dl.Error(v...)
}

// Errorf logs message as `ERROR`. Arguments handled in the mananer of `fmt.Printf`.
func Errorf(format string, v ...interface{}) {
	dl.Errorf(format, v...)
}

// Warn logs message as `WARN`. Arguments handled in the mananer of `fmt.Print`.
func Warn(v ...interface{}) {
	dl.Warn(v...)
}

// Warnf logs message as `WARN`. Arguments handled in the mananer of `fmt.Printf`.
func Warnf(format string, v ...interface{}) {
	dl.Warnf(format, v...)
}

// Info logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func Info(v ...interface{}) {
	dl.Info(v...)
}

// Infof logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func Infof(format string, v ...interface{}) {
	dl.Infof(format, v...)
}

// Debug logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Print`.
func Debug(v ...interface{}) {
	dl.Debug(v...)
}

// Debugf logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Printf`.
func Debugf(format string, v ...interface{}) {
	dl.Debugf(format, v...)
}

// Trace logs message as `TRACE`. Arguments handled in the mananer of `fmt.Print`.
func Trace(v ...interface{}) {
	dl.Trace(v...)
}

// Tracef logs message as `TRACE`. Arguments handled in the mananer of `fmt.Printf`.
func Tracef(format string, v ...interface{}) {
	dl.Tracef(format, v...)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods - Drop-in replacement
// for Go standard logger
//_______________________________________

// Print logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func Print(v ...interface{}) {
	dl.Print(v...)
}

// Printf logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func Printf(format string, v ...interface{}) {
	dl.Printf(format, v...)
}

// Println logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func Println(v ...interface{}) {
	dl.Println(v...)
}

// Fatal logs message as `FATAL` and call to os.Exit(1).
func Fatal(v ...interface{}) {
	dl.Fatal(v...)
}

// Fatalf logs message as `FATAL` and call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	dl.Fatalf(format, v...)
}

// Fatalln logs message as `FATAL` and call to os.Exit(1).
func Fatalln(v ...interface{}) {
	dl.Fatalln(v...)
}

// Panic logs message as `PANIC` and call to panic().
func Panic(v ...interface{}) {
	dl.Panic(v...)
}

// Panicf logs message as `PANIC` and call to panic().
func Panicf(format string, v ...interface{}) {
	dl.Panicf(format, v...)
}

// Panicln logs message as `PANIC` and call to panic().
func Panicln(v ...interface{}) {
	dl.Panicln(v...)
}

// AddContext method to add context values into default logger.
// These context values gets logged with each log entry.
func AddContext(fields Fields) {
	dl.AddContext(fields)
}

// AddHook method is to add logger hook function.
func AddHook(name string, hook HookFunc) error {
	return dl.AddHook(name, hook)
}

// WithFields method to add multiple key-value pairs into log.
func WithFields(fields Fields) Loggerer {
	return dl.WithFields(fields)
}

// WithField method to add single key-value into log
func WithField(key string, value interface{}) Loggerer {
	return dl.WithField(key, value)
}

// Writer method returns the writer of default logger.
func Writer() io.Writer {
	return dl.receiver.Writer()
}

// SetWriter method sets the given writer into logger instance.
func SetWriter(w io.Writer) {
	dl.SetWriter(w)
}

// ToGoLogger method wraps the current log writer into Go Logger instance.
func ToGoLogger() *slog.Logger {
	return dl.ToGoLogger()
}

// SetDefaultLogger method sets the given logger instance as default logger.
func SetDefaultLogger(l *Logger) {
	dl = l
}

// SetLevel method sets log level for default logger.
func SetLevel(level string) error {
	return dl.SetLevel(level)
}

// SetPattern method sets the log format pattern for default logger.
func SetPattern(pattern string) error {
	return dl.SetPattern(pattern)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger level and assertion methods
//___________________________________

// Level method returns currently enabled logging level.
func Level() string {
	return dl.Level()
}

// IsLevelInfo method returns true if log level is INFO otherwise false.
func IsLevelInfo() bool {
	return dl.IsLevelInfo()
}

// IsLevelError method returns true if log level is ERROR otherwise false.
func IsLevelError() bool {
	return dl.IsLevelError()
}

// IsLevelWarn method returns true if log level is WARN otherwise false.
func IsLevelWarn() bool {
	return dl.IsLevelWarn()
}

// IsLevelDebug method returns true if log level is DEBUG otherwise false.
func IsLevelDebug() bool {
	return dl.IsLevelDebug()
}

// IsLevelTrace method returns true if log level is TRACE otherwise false.
func IsLevelTrace() bool {
	return dl.IsLevelTrace()
}

func init() {
	cfg, _ := config.ParseString("log { }")
	dl, _ = New(cfg)
}
