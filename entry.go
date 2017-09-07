// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

var (
	entryPool *sync.Pool
	bufPool   *sync.Pool
	_         loggerInterface = (*Entry)(nil)
)

// Fields type is used to log fields values in the logger.
type Fields map[string]interface{}

// Entry represents a log entry and contains the timestamp when the entry
// was created, level, etc.
type Entry struct {
	AppName   string    `json:"appname,omitempty"`
	RequestID string    `json:"reqid,omitempty"`
	Principal string    `json:"principal,omitempty"`
	Level     level     `json:"-"`
	Time      time.Time `json:"-"`
	Message   string    `json:"message,omitempty"`
	File      string    `json:"file,omitempty"`
	Line      int       `json:"line,omitempty"`
	Fields    Fields    `json:"fields,omitempty"`

	logger *Logger
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Entry methods
//___________________________________

// MarshalJSON method for formating entry to JSON.
func (e *Entry) MarshalJSON() ([]byte, error) {
	type alias Entry
	return json.Marshal(&struct {
		Level string `json:"level,omitempty"`
		Time  string `json:"timestamp,omitempty"`
		*alias
	}{
		Level: levelToLevelName[e.Level],
		Time:  formatTime(e.Time),
		alias: (*alias)(e),
	})
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Entry logger methods
//_______________________________________

// Error logs message as `ERROR`. Arguments handled in the mananer of `fmt.Print`.
func (e *Entry) Error(v ...interface{}) {
	if e.logger.level >= LevelError {
		e.output(LevelError, fmt.Sprint(v...))
	}
}

// Errorf logs message as `ERROR`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Errorf(format string, v ...interface{}) {
	if e.logger.level >= LevelError {
		e.output(LevelError, fmt.Sprintf(format, v...))
	}
}

// Warn logs message as `WARN`. Arguments handled in the mananer of `fmt.Print`.
func (e *Entry) Warn(v ...interface{}) {
	if e.logger.level >= LevelWarn {
		e.output(LevelWarn, fmt.Sprint(v...))
	}
}

// Warnf logs message as `WARN`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Warnf(format string, v ...interface{}) {
	if e.logger.level >= LevelWarn {
		e.output(LevelWarn, fmt.Sprintf(format, v...))
	}
}

// Info logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (e *Entry) Info(v ...interface{}) {
	if e.logger.level >= LevelInfo {
		e.output(LevelInfo, fmt.Sprint(v...))
	}
}

// Infof logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Infof(format string, v ...interface{}) {
	if e.logger.level >= LevelInfo {
		e.output(LevelInfo, fmt.Sprintf(format, v...))
	}
}

// Debug logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Print`.
func (e *Entry) Debug(v ...interface{}) {
	if e.logger.level >= LevelDebug {
		e.output(LevelDebug, fmt.Sprint(v...))
	}
}

// Debugf logs message as `DEBUG`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Debugf(format string, v ...interface{}) {
	if e.logger.level >= LevelDebug {
		e.output(LevelDebug, fmt.Sprintf(format, v...))
	}
}

// Trace logs message as `TRACE`. Arguments handled in the mananer of `fmt.Print`.
func (e *Entry) Trace(v ...interface{}) {
	if e.logger.level >= LevelTrace {
		e.output(LevelTrace, fmt.Sprint(v...))
	}
}

// Tracef logs message as `TRACE`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Tracef(format string, v ...interface{}) {
	if e.logger.level >= LevelTrace {
		e.output(LevelTrace, fmt.Sprintf(format, v...))
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Entry methods - Drop-in replacement
// for Go standard logger
//_______________________________________

// Print logs message as `INFO`. Arguments handled in the mananer of `fmt.Print`.
func (e *Entry) Print(v ...interface{}) {
	e.output(LevelInfo, fmt.Sprint(v...))
}

// Printf logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Printf(format string, v ...interface{}) {
	e.output(LevelInfo, fmt.Sprintf(format, v...))
}

// Println logs message as `INFO`. Arguments handled in the mananer of `fmt.Printf`.
func (e *Entry) Println(v ...interface{}) {
	e.output(LevelInfo, fmt.Sprint(v...))
}

// Fatal logs message as `FATAL` and call to os.Exit(1).
func (e *Entry) Fatal(v ...interface{}) {
	e.output(levelFatal, fmt.Sprint(v...))
	exit(1)
}

// Fatalf logs message as `FATAL` and call to os.Exit(1).
func (e *Entry) Fatalf(format string, v ...interface{}) {
	e.output(levelFatal, fmt.Sprintf(format, v...))
	exit(1)
}

// Fatalln logs message as `FATAL` and call to os.Exit(1).
func (e *Entry) Fatalln(v ...interface{}) {
	e.output(levelFatal, fmt.Sprint(v...))
	exit(1)
}

// Panic logs message as `PANIC` and call to panic().
func (e *Entry) Panic(v ...interface{}) {
	e.output(levelPanic, fmt.Sprint(v...))
	panic(e)
}

// Panicf logs message as `PANIC` and call to panic().
func (e *Entry) Panicf(format string, v ...interface{}) {
	e.output(levelPanic, fmt.Sprintf(format, v...))
	panic(e)
}

// Panicln logs message as `PANIC` and call to panic().
func (e *Entry) Panicln(v ...interface{}) {
	e.output(levelPanic, fmt.Sprint(v...))
	panic(e)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Entry context/field methods
//_______________________________________

// WithFields method to add multiple key-value pairs into log.
func (e *Entry) WithFields(fields Fields) *Entry {
	ne := acquireEntry(e.logger)
	for k, v := range e.Fields {
		ne.Fields[k] = v
	}
	for k, v := range fields {
		if k == "appname" {
			ne.AppName = fmt.Sprint(v)
			continue
		}
		if k == "reqid" {
			ne.RequestID = fmt.Sprint(v)
			continue
		}
		if k == "principal" {
			ne.Principal = fmt.Sprint(v)
			continue
		}

		ne.Fields[k] = v
	}
	return ne
}

// WithField method to add single key-value into log
func (e *Entry) WithField(key string, value interface{}) *Entry {
	return e.WithFields(Fields{key: value})
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Entry level methods
//___________________________________

// IsLevelInfo method returns true if log level is INFO otherwise false.
func (e *Entry) IsLevelInfo() bool {
	return e.logger.level == LevelInfo
}

// IsLevelError method returns true if log level is ERROR otherwise false.
func (e *Entry) IsLevelError() bool {
	return e.logger.level == LevelError
}

// IsLevelWarn method returns true if log level is WARN otherwise false.
func (e *Entry) IsLevelWarn() bool {
	return e.logger.level == LevelWarn
}

// IsLevelDebug method returns true if log level is DEBUG otherwise false.
func (e *Entry) IsLevelDebug() bool {
	return e.logger.level == LevelDebug
}

// IsLevelTrace method returns true if log level is TRACE otherwise false.
func (e *Entry) IsLevelTrace() bool {
	return e.logger.level == LevelTrace
}

// Reset method resets the `Entry` values for reuse.
func (e *Entry) Reset() {
	e.AppName = ""
	e.RequestID = ""
	e.Principal = ""
	e.Level = LevelUnknown
	e.Time = time.Time{}
	e.Message = ""
	e.File = ""
	e.Line = 0
	e.Fields = make(map[string]interface{})
	e.logger = nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func (e *Entry) output(lvl level, msg string) {
	e.Time = time.Now()
	e.Level = lvl
	e.Message = msg
	e.logger.output(e)
}

func newEntry() *Entry {
	return &Entry{
		Fields: make(map[string]interface{}),
	}
}

func acquireEntry(logger *Logger) *Entry {
	e := entryPool.Get().(*Entry)
	e.logger = logger
	return e
}

func releaseEntry(e *Entry) {
	e.Reset()
	entryPool.Put(e)
}

func acquireBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func releaseBuffer(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bufPool.Put(buf)
	}
}

func init() {
	bufPool = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}
	entryPool = &sync.Pool{New: func() interface{} { return newEntry() }}
}
