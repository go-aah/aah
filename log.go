// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package log implements a simple, flexible, non-blocking logger.
// It supports `console`, `file` (rotation by daily, size, lines).
// It also has a predefined 'standard' Logger accessible through helper
// functions `Error{f}`, `Warn{f}`, `Info{f}`, `Debug{f}`, `Trace{f}`,
// `Print{f,ln}`, `Fatal{f,ln}`, `Panic{f,ln}` which are easier to use than creating
// a Logger manually. Default logger writes to standard error and prints log
// `Entry` details as per `DefaultPattern`.
//
// aah log package can be used as drop-in replacement for standard go logger
// with features.
//
// 	log.Info("Welcome ", "to ", "aah ", "logger")
// 	log.Infof("%v, %v, %v", "simple", "flexible", "logger")
//
// 	// Output:
// 	2016-07-03 19:22:11.504 INFO  Welcome to aah logger
// 	2016-07-03 19:22:11.504 INFO  simple, flexible, logger
package log

import (
	"errors"
	"fmt"
	"io"
	slog "log"
	"os"
	"strings"
	"sync"

	"aahframework.org/config.v0"
)

// Level type definition
type level uint8

// HookFunc type is aah framework logger custom hook.
type HookFunc func(e Entry)

// Log Level definition
const (
	LevelFatal level = iota
	LevelPanic
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
	LevelUnknown
)

var (
	// ErrLogReceiverIsNil returned when suppiled receiver is nil.
	ErrLogReceiverIsNil = errors.New("log: receiver is nil")

	// ErrHookFuncIsNil is returned when hook function is nil.
	ErrHookFuncIsNil = errors.New("log: hook func is nil")

	filePermission = os.FileMode(0755)

	// abstract it, can be unit tested
	exit = os.Exit

	_ Loggerer = (*Logger)(nil)
)

type (
	// Logger is the object which logs the given message into recevier as per deifned
	// format flags. Logger can be used simultaneously from multiple goroutines;
	// it guarantees to serialize access to the Receivers.
	Logger struct {
		cfg      *config.Config
		m        *sync.RWMutex
		level    level
		receiver Receiver
		ctx      Fields
		hooks    map[string]HookFunc
	}

	// Receiver is the interface for pluggable log receiver.
	// For e.g: Console, File
	Receiver interface {
		Init(cfg *config.Config) error
		SetPattern(pattern string) error
		SetWriter(w io.Writer)
		IsCallerInfo() bool
		Writer() io.Writer
		Log(e *Entry)
	}

	// Loggerer interface is for Logger and Entry log method implementation.
	Loggerer interface {
		Error(v ...interface{})
		Errorf(format string, v ...interface{})
		Warn(v ...interface{})
		Warnf(format string, v ...interface{})
		Info(v ...interface{})
		Infof(format string, v ...interface{})
		Debug(v ...interface{})
		Debugf(format string, v ...interface{})
		Trace(v ...interface{})
		Tracef(format string, v ...interface{})

		// Context/Field methods
		WithFields(fields Fields) Loggerer
		WithField(key string, value interface{}) Loggerer

		// Level Info
		IsLevelInfo() bool
		IsLevelError() bool
		IsLevelWarn() bool
		IsLevelDebug() bool
		IsLevelTrace() bool
		IsLevelFatal() bool
		IsLevelPanic() bool

		// For standard logger drop-in replacement
		ToGoLogger() *slog.Logger
		Print(v ...interface{})
		Printf(format string, v ...interface{})
		Println(v ...interface{})
		Fatal(v ...interface{})
		Fatalf(format string, v ...interface{})
		Fatalln(v ...interface{})
		Panic(v ...interface{})
		Panicf(format string, v ...interface{})
		Panicln(v ...interface{})
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// New method creates the aah logger based on supplied `config.Config`.
func New(cfg *config.Config) (*Logger, error) {
	if cfg == nil {
		return nil, errors.New("log: config is nil")
	}

	logger := &Logger{m: &sync.RWMutex{}, cfg: cfg}

	// Receiver
	receiverType := strings.ToUpper(cfg.StringDefault("log.receiver", "CONSOLE"))
	if err := logger.SetReceiver(getReceiverByName(receiverType)); err != nil {
		return nil, err
	}

	// Pattern
	if err := logger.SetPattern(cfg.StringDefault("log.pattern", DefaultPattern)); err != nil {
		return nil, err
	}

	// Level
	if err := logger.SetLevel(cfg.StringDefault("log.level", "DEBUG")); err != nil {
		return nil, err
	}

	logger.ctx = make(Fields)
	logger.hooks = make(map[string]HookFunc)

	return logger, nil
}

// NewWithContext method creates the aah logger based on supplied `config.Config`.
func NewWithContext(cfg *config.Config, ctx Fields) (*Logger, error) {
	l, err := New(cfg)
	if err != nil {
		return nil, err
	}
	l.AddContext(ctx)
	return l, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods
//___________________________________

// New method creates a child logger and adds structured context to it. Child
// logger inherits parent logger context value on creation. Fields added
// to the child don't affect the parent logger and vice versa. These context
// values gets logged with each log entry.
//
// Also you can use method `AddContext` to context to the current logger.
func (l *Logger) New(fields Fields) *Logger {
	nl := *l
	nl.ctx = make(Fields)
	nl.AddContext(l.ctx)
	nl.AddContext(fields)
	return &nl
}

// AddContext method to add context values into current logger.
func (l *Logger) AddContext(fields Fields) {
	for k, v := range fields {
		l.ctx[k] = v
	}
}

// AddHook method is to add logger hook function.
func (l *Logger) AddHook(name string, hook HookFunc) error {
	if hook == nil {
		return ErrHookFuncIsNil
	}

	l.m.Lock()
	defer l.m.Unlock()
	if _, found := l.hooks[name]; found {
		return fmt.Errorf("log: hook name '%v' is already added, skip it", name)
	}

	l.hooks[name] = hook
	return nil
}

// Level method returns currently enabled logging level.
func (l *Logger) Level() string {
	return levelToLevelName[l.level]
}

// SetLevel method sets the given logging level for the logger.
// For e.g.: INFO, WARN, DEBUG, etc. Case-insensitive.
func (l *Logger) SetLevel(level string) error {
	l.m.Lock()
	defer l.m.Unlock()
	levelFlag := levelByName(level)
	if levelFlag == LevelUnknown {
		return fmt.Errorf("log: unknown log level '%s'", level)
	}
	l.level = levelFlag
	return nil
}

// SetPattern method sets the log format pattern.
func (l *Logger) SetPattern(pattern string) error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.receiver == nil {
		return ErrLogReceiverIsNil
	}
	return l.receiver.SetPattern(pattern)
}

// SetReceiver method sets the given receiver into logger instance.
func (l *Logger) SetReceiver(receiver Receiver) error {
	l.m.Lock()
	defer l.m.Unlock()

	if receiver == nil {
		return ErrLogReceiverIsNil
	}

	l.receiver = receiver
	return l.receiver.Init(l.cfg)
}

// SetWriter method sets the given writer into logger instance.
func (l *Logger) SetWriter(w io.Writer) {
	l.m.Lock()
	defer l.m.Unlock()
	l.receiver.SetWriter(w)
}

// ToGoLogger method wraps the current log writer into Go Logger instance.
func (l *Logger) ToGoLogger() *slog.Logger {
	return slog.New(l.receiver.Writer(), "", slog.LstdFlags)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger logging methods
//_______________________________________

// Error logs message as `ERROR`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Error(v ...interface{}) {
	if l.level >= LevelError {
		e := acquireEntry(l)
		e.Error(v...)
		releaseEntry(e)
	}
}

// Errorf logs message as `ERROR`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level >= LevelError {
		e := acquireEntry(l)
		e.Errorf(format, v...)
		releaseEntry(e)
	}
}

// Warn logs message as `WARN`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Warn(v ...interface{}) {
	if l.level >= LevelWarn {
		e := acquireEntry(l)
		e.Warn(v...)
		releaseEntry(e)
	}
}

// Warnf logs message as `WARN`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level >= LevelWarn {
		e := acquireEntry(l)
		e.Warnf(format, v...)
		releaseEntry(e)
	}
}

// Info logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Info(v ...interface{}) {
	if l.level >= LevelInfo {
		e := acquireEntry(l)
		e.Info(v...)
		releaseEntry(e)
	}
}

// Infof logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level >= LevelInfo {
		e := acquireEntry(l)
		e.Infof(format, v...)
		releaseEntry(e)
	}
}

// Debug logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Debug(v ...interface{}) {
	if l.level >= LevelDebug {
		e := acquireEntry(l)
		e.Debug(v...)
		releaseEntry(e)
	}
}

// Debugf logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level >= LevelDebug {
		e := acquireEntry(l)
		e.Debugf(format, v...)
		releaseEntry(e)
	}
}

// Trace logs message as `TRACE`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Trace(v ...interface{}) {
	if l.level >= LevelTrace {
		e := acquireEntry(l)
		e.Trace(v...)
		releaseEntry(e)
	}
}

// Tracef logs message as `TRACE`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Tracef(format string, v ...interface{}) {
	if l.level >= LevelTrace {
		e := acquireEntry(l)
		e.Tracef(format, v...)
		releaseEntry(e)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger context/field methods
//_______________________________________

// WithFields method to add multiple key-value pairs into log entry.
func (l *Logger) WithFields(fields Fields) Loggerer {
	e := acquireEntry(l)
	defer releaseEntry(e)
	return e.WithFields(fields)
}

// WithField method to add single key-value into log entry.
func (l *Logger) WithField(key string, value interface{}) Loggerer {
	e := acquireEntry(l)
	defer releaseEntry(e)
	return e.WithField(key, value)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods - Drop-in replacement
// for Go standard logger
//_______________________________________

// Print logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Print(v ...interface{}) {
	e := acquireEntry(l)
	e.Print(v...)
	releaseEntry(e)
}

// Printf logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Printf(format string, v ...interface{}) {
	e := acquireEntry(l)
	e.Printf(format, v...)
	releaseEntry(e)
}

// Println logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Println(v ...interface{}) {
	e := acquireEntry(l)
	e.Println(v...)
	releaseEntry(e)
}

// Fatal logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	e := acquireEntry(l)
	e.Fatal(v...)
	releaseEntry(e)
}

// Fatalf logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	e := acquireEntry(l)
	e.Fatalf(format, v...)
	releaseEntry(e)
}

// Fatalln logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatalln(v ...interface{}) {
	e := acquireEntry(l)
	e.Fatalln(v...)
	releaseEntry(e)
}

// Panic logs message as `PANIC` and call to panic().
func (l *Logger) Panic(v ...interface{}) {
	e := acquireEntry(l)
	e.Panic(v...)
	releaseEntry(e)
}

// Panicf logs message as `PANIC` and call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	e := acquireEntry(l)
	e.Panicf(format, v...)
	releaseEntry(e)
}

// Panicln logs message as `PANIC` and call to panic().
func (l *Logger) Panicln(v ...interface{}) {
	e := acquireEntry(l)
	e.Panicln(v...)
	releaseEntry(e)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger level methods
//___________________________________

// IsLevelInfo method returns true if log level is INFO otherwise false.
func (l *Logger) IsLevelInfo() bool {
	return l.level == LevelInfo
}

// IsLevelError method returns true if log level is ERROR otherwise false.
func (l *Logger) IsLevelError() bool {
	return l.level == LevelError
}

// IsLevelWarn method returns true if log level is WARN otherwise false.
func (l *Logger) IsLevelWarn() bool {
	return l.level == LevelWarn
}

// IsLevelDebug method returns true if log level is DEBUG otherwise false.
func (l *Logger) IsLevelDebug() bool {
	return l.level == LevelDebug
}

// IsLevelTrace method returns true if log level is TRACE otherwise false.
func (l *Logger) IsLevelTrace() bool {
	return l.level == LevelTrace
}

// IsLevelFatal method returns true if log level is FATAL otherwise false.
func (l *Logger) IsLevelFatal() bool {
	return l.level == LevelFatal
}

// IsLevelPanic method returns true if log level is PANIC otherwise false.
func (l *Logger) IsLevelPanic() bool {
	return l.level == LevelPanic
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func (l *Logger) output(e *Entry) {
	if l.receiver.IsCallerInfo() {
		e.File, e.Line = fetchCallerInfo()
	}
	l.receiver.Log(e)

	// Execute logger hooks
	go l.executeHooks(*e)
}

func (l *Logger) executeHooks(e Entry) {
	l.m.RLock()
	defer l.m.RUnlock()
	for _, fn := range l.hooks {
		go fn(e)
	}
}
