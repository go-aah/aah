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
// 	log.Infof("%v, %v, %v", "simple", "flexible", "non-blocking logger")
//
// 	// Output:
// 	2016-07-03 19:22:11.504 INFO  Welcome to aah logger
// 	2016-07-03 19:22:11.504 INFO  simple, flexible, non-blocking logger
package log

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"aahframework.org/config.v0"
)

// Level type definition
type Level uint8

// FmtFlag type definition
type FmtFlag uint8

// Log Level definition
const (
	levelFatal Level = iota
	levelPanic
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
	LevelUnknown
)

// Format flags used to define log message format for each log entry
const (
	FmtFlagLevel FmtFlag = iota
	FmtFlagTime
	FmtFlagUTCTime
	FmtFlagLongfile
	FmtFlagShortfile
	FmtFlagLine
	FmtFlagMessage
	FmtFlagCustom
	FmtFlagUnknown
)

var (
	// Version no. of aahframework.org/log library
	Version = "0.3.2"

	// FmtFlags is the list of log format flags supported by aah/log library
	// Usage of flag order is up to format composition.
	//    level     - outputs INFO, DEBUG, ERROR, so on
	//    time      - outputs local time as per format supplied
	//    utctime   - outputs UTC time as per format supplied
	//    longfile  - outputs full file name: /a/b/c/d.go
	//    shortfile - outputs final file name element: d.go
	//    line      - outputs file line number: L23
	//    message   - outputs given message along supplied arguments if they present
	//    custom    - outputs string as-is into log entry
	FmtFlags = map[string]FmtFlag{
		"level":     FmtFlagLevel,
		"time":      FmtFlagTime,
		"utctime":   FmtFlagUTCTime,
		"longfile":  FmtFlagLongfile,
		"shortfile": FmtFlagShortfile,
		"line":      FmtFlagLine,
		"message":   FmtFlagMessage,
		"custom":    FmtFlagCustom,
	}

	// DefaultPattern is default log entry pattern in aah/log. Only applicable to
	// text formatter.
	// For e.g:
	//    2006-01-02 15:04:05.000 INFO  This is my message
	DefaultPattern = "%time:2006-01-02 15:04:05.000 %level:-5 %message"

	// BackupTimeFormat is used for timestamp with filename on rotation
	BackupTimeFormat = "2006-01-02-15-04-05.000"

	// ErrLogReceiverIsNil returned when suppiled receiver is nil.
	ErrLogReceiverIsNil = errors.New("log: receiver is nil")

	flagSeparator      = "%"
	flagValueSeparator = ":"
	defaultFormat      = "%v"
	filePermission     = os.FileMode(0755)
)

type (
	// Receiver is the interface for pluggable log receiver.
	// For e.g: Console, File, HTTP, etc
	Receiver interface {
		Init(cfg *config.Config) error
		SetPattern(pattern string) error
		IsCallerInfo() bool
		Log(e *Entry)
	}

	// Logger is the object which logs the given message into recevier as per deifned
	// format flags. Logger can be used simultaneously from multiple goroutines;
	// it guarantees to serialize access to the Receivers.
	Logger struct {
		cfg      *config.Config
		m        *sync.Mutex
		level    Level
		receiver Receiver
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// New method creates the aah logger based on supplied `config.Config`.
func New(cfg *config.Config) (*Logger, error) {
	if cfg == nil {
		return nil, errors.New("log: config is nil")
	}

	logger := &Logger{m: &sync.Mutex{}, cfg: cfg}

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

	return logger, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods
//___________________________________

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

// SetPattern methods sets the log format pattern.
func (l *Logger) SetPattern(pattern string) error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.receiver == nil {
		return ErrLogReceiverIsNil
	}
	return l.receiver.SetPattern(pattern)
}

// SetReceiver sets the given receiver into logger instance.
func (l *Logger) SetReceiver(receiver Receiver) error {
	l.m.Lock()
	defer l.m.Unlock()

	if receiver == nil {
		return ErrLogReceiverIsNil
	}

	l.receiver = receiver
	return l.receiver.Init(l.cfg)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger logging methods
//_______________________________________

// Error logs message as `ERROR`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Error(v ...interface{}) {
	l.output(LevelError, 3, nil, v...)
}

// Errorf logs message as `ERROR`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.output(LevelError, 3, &format, v...)
}

// Warn logs message as `WARN`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Warn(v ...interface{}) {
	l.output(LevelWarn, 3, nil, v...)
}

// Warnf logs message as `WARN`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.output(LevelWarn, 3, &format, v...)
}

// Info logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Info(v ...interface{}) {
	l.output(LevelInfo, 3, nil, v...)
}

// Infof logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.output(LevelInfo, 3, &format, v...)
}

// Debug logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Debug(v ...interface{}) {
	l.output(LevelDebug, 3, nil, v...)
}

// Debugf logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.output(LevelDebug, 3, &format, v...)
}

// Trace logs message as `TRACE`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Trace(v ...interface{}) {
	l.output(LevelTrace, 3, nil, v...)
}

// Tracef logs message as `TRACE`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Tracef(format string, v ...interface{}) {
	l.output(LevelTrace, 3, &format, v...)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Logger methods - Drop-in replacement
// for Go standard logger
//_______________________________________

// Print logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (l *Logger) Print(v ...interface{}) {
	l.output(LevelInfo, 3, nil, v...)
}

// Printf logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.output(LevelInfo, 3, &format, v...)
}

// Println logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (l *Logger) Println(format string, v ...interface{}) {
	l.output(LevelInfo, 3, &format, v...)
}

// Fatal logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	l.output(levelFatal, 3, nil, v...)
	os.Exit(1)
}

// Fatalf logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.output(levelFatal, 3, &format, v...)
	os.Exit(1)
}

// Fatalln logs message as `FATAL` and call to os.Exit(1).
func (l *Logger) Fatalln(format string, v ...interface{}) {
	l.output(levelFatal, 3, &format, v...)
	os.Exit(1)
}

// Panic logs message as `PANIC` and call to panic().
func (l *Logger) Panic(v ...interface{}) {
	l.output(levelPanic, 3, nil, v...)
	panic("")
}

// Panicf logs message as `PANIC` and call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	l.output(levelPanic, 3, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

// Panicln logs message as `PANIC` and call to panic().
func (l *Logger) Panicln(format string, v ...interface{}) {
	l.output(levelPanic, 3, &format, v...)
	panic(fmt.Sprintf(format, v...))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// output method checks the level, formats the arguments and call to configured
// Log receivers.
func (l *Logger) output(level Level, calldepth int, format *string, v ...interface{}) {
	if level > l.level {
		return
	}

	entry := getEntry()
	defer putEntry(entry)
	entry.Time = time.Now()
	entry.Level = level
	if format == nil {
		entry.Message = fmt.Sprint(v...)
	} else {
		entry.Message = fmt.Sprintf(*format, v...)
	}

	if l.receiver.IsCallerInfo() {
		entry.File, entry.Line = fetchCallerInfo(calldepth)
	}

	l.receiver.Log(entry)
}
