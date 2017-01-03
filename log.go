// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package log implements a simple, flexible & powerful logger. Supports
// console, file (rotation by size, daily, lines), logging receivers
// and Logging Stats. It also has a predefined 'standard' Logger accessible
// through helper functions Error{f}, Warn{f}, Info{f}, Debug{f}, Trace{f}
// which are easier to use than creating a Logger manually. That logger writes
// to standard error and prints log `Entry` details as per `DefaultPattern`.
// 	log.Info("Welcome ", "to ", "aah ", "logger")
// 	log.Infof("%v, %v, & %v", "simple", "flexible", "powerful logger")
//
// 	// Output:
// 	2016-07-03 19:22:11.504 INFO  - Welcome to aah logger
// 	2016-07-03 19:22:11.504 INFO  - simple, flexible, & powerful logger
package log

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"aahframework.org/config"
	"aahframework.org/essentials"
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
	// Version no. of go-aah/log library
	Version = "0.1"

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

	// DefaultPattern is default log entry pattern in aah/log
	// For e.g:
	//    2006-01-02 15:04:05.000 INFO  - This is my message
	DefaultPattern = "%time:2006-01-02 15:04:05.000 %level:-5 %custom:- %message"

	// BackupTimeFormat is used for timestamp with filename on rotation
	BackupTimeFormat = "2006-01-02-15-04-05.000"

	// ErrFormatStringEmpty returned when log format parameter is empty
	ErrFormatStringEmpty = errors.New("log format string is empty")

	// ErrWriterIsClosed returned when log writer is closed
	ErrWriterIsClosed = errors.New("log writer is closed")

	flagSeparator      = "%"
	flagValueSeparator = ":"
	defaultFormat      = "%v"
	filePermission     = os.FileMode(0755)

	levelNameToLevel = map[string]Level{
		"FATAL": levelFatal,
		"PANIC": levelPanic,
		"ERROR": LevelError,
		"WARN":  LevelWarn,
		"INFO":  LevelInfo,
		"DEBUG": LevelDebug,
		"TRACE": LevelTrace,
	}

	levelToLevelName = map[Level]string{
		levelFatal: "FATAL",
		levelPanic: "PANIC",
		LevelError: "ERROR",
		LevelWarn:  "WARN",
		LevelInfo:  "INFO",
		LevelDebug: "DEBUG",
		LevelTrace: "TRACE",
	}

	// ANSI color codes
	resetColor   = []byte("\033[0m")
	levelToColor = [][]byte{
		levelFatal: []byte("\033[0;31m"), // red
		levelPanic: []byte("\033[0;31m"), // red
		LevelError: []byte("\033[0;31m"), // red
		LevelWarn:  []byte("\033[0;33m"), // yellow
		LevelInfo:  []byte("\033[0;37m"), // white
		LevelDebug: []byte("\033[0;34m"), // blue
		LevelTrace: []byte("\033[0;35m"), // magenta (purple)
	}

	_ Logger = &Receiver{}
)

// Entry represents a log entry and contains the timestamp when the entry
// was created, level, etc.
type Entry struct {
	Level  Level
	Time   time.Time
	Format *string
	Values *[]interface{}
	File   string
	Line   int
}

// Logger is interface for `aah/log` package
type Logger interface {
	// Output writes the entry data into receiver
	Output(level Level, calldepth int, format *string, v ...interface{}) error

	// Close closes the log writer. It cannot be used after this operation
	Close()

	// Closed returns true if the logger was previously closed
	Closed() bool

	// Stats returns current logger statistics like number of lines written,
	// number of bytes written, etc.
	Stats() *ReceiverStats

	// SetPattern sets the log entry format
	SetPattern(pattern string) error

	// SetLevel allows to set log level dynamically
	SetLevel(level Level)

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})

	Panic(v ...interface{})
	Panicf(format string, v ...interface{})

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
}

// New creates the aah logger based on supplied config string
func New(configStr string) (Logger, error) {
	if ess.IsStrEmpty(configStr) {
		return nil, errors.New("logger config is empty")
	}

	cfg, err := config.ParseString(configStr)
	if err != nil {
		return nil, err
	}

	return Newc(cfg)
}

// Newc creates the aah logger based on supplied `config.Config`
func Newc(cfg *config.Config) (Logger, error) {
	if cfg == nil {
		return nil, errors.New("logger config is nil")
	}

	receiverType, found := cfg.String("receiver")
	if !found {
		return nil, errors.New("receiver configuration is required")
	}
	receiverType = strings.ToUpper(receiverType)

	levelName := cfg.StringDefault("level", "DEBUG")
	level := levelByName(levelName)
	if level == LevelUnknown {
		return nil, fmt.Errorf("unrecognized log level: %v", levelName)
	}

	pattern := cfg.StringDefault("pattern", DefaultPattern)
	flags, err := parseFlag(pattern)
	if err != nil {
		return nil, err
	}

	var alogger interface{}
	switch receiverType {
	case "CONSOLE":
		alogger, err = newConsoleReceiver(cfg, receiverType, level, flags)
	case "FILE":
		alogger, err = newFileReceiver(cfg, receiverType, level, flags)
	default:
		return nil, errors.New("unsupported receiver")
	}

	if err != nil {
		return nil, err
	} else if logger, ok := alogger.(Logger); ok {
		return logger, nil
	}

	return nil, errors.New("unable to create logger")
}

func (level Level) String() string {
	if name, ok := levelToLevelName[level]; ok {
		return name
	}

	return "Unknown"
}

// unexported methods

func levelByName(name string) Level {
	if level, ok := levelNameToLevel[strings.ToUpper(name)]; ok {
		return level
	}

	return LevelUnknown
}

func fetchCallerInfo(calldepth int) (string, int) {
	_, file, line, ok := runtime.Caller(calldepth + 1)
	if !ok {
		file = "???"
		line = 0
	}

	return file, line
}

func newConsoleReceiver(cfg *config.Config, receiverType string, level Level, flags *[]FlagPart) (*Receiver, error) {
	receiver := Receiver{
		Config:     cfg,
		Type:       receiverType,
		Flags:      flags,
		Format:     DefaultFormatter,
		m:          sync.Mutex{},
		level:      level,
		out:        os.Stderr,
		stats:      &ReceiverStats{},
		isFileInfo: isFileFlagExists(flags),
		isLineInfo: isFmtFlagExists(flags, FmtFlagLine),
		isColor:    runtime.GOOS != "windows",
	}

	return &receiver, nil
}

func newFileReceiver(cfg *config.Config, receiverType string, level Level, flags *[]FlagPart) (*Receiver, error) {
	maxSize := cfg.IntDefault("rotate.size", 100)
	if maxSize > 2048 { // maximum 2GB file size
		return nil, errors.New("max size > 2GB, please set it to 2048 for size rotation")
	}

	receiver := Receiver{
		Config:     cfg,
		Type:       receiverType,
		Flags:      flags,
		Format:     DefaultFormatter,
		level:      level,
		stats:      &ReceiverStats{},
		isFileInfo: isFileFlagExists(flags),
		isLineInfo: isFmtFlagExists(flags, FmtFlagLine),
		isUTC:      isFmtFlagExists(flags, FmtFlagUTCTime),
	}

	err := receiver.openFile()
	if err != nil {
		return nil, err
	}

	receiver.rotate = cfg.StringDefault("rotate.mode", "daily")
	switch receiver.rotate {
	case "daily":
		receiver.setOpenDay()
	case "lines":
		receiver.maxLines = int64(cfg.IntDefault("rotate.lines", 0))
	case "size":
		receiver.maxSize = int64(maxSize * 1024 * 1024)
	}

	return &receiver, nil
}
