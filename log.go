// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
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
	"time"

	"github.com/go-aah/forge"
)

// Level type definition
type Level uint8

// Log Level definition
const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
	LevelUnknown
)

var (
	// ErrWriterIsClosed returned when log writer is closed
	ErrWriterIsClosed = errors.New("log writer is closed")

	levelNameToLevel = map[string]Level{
		"ERROR": LevelError,
		"WARN":  LevelWarn,
		"INFO":  LevelInfo,
		"DEBUG": LevelDebug,
		"TRACE": LevelTrace,
	}

	levelToLevelName = map[Level]string{
		LevelError: "ERROR",
		LevelWarn:  "WARN",
		LevelInfo:  "INFO",
		LevelDebug: "DEBUG",
		LevelTrace: "TRACE",
	}

	// ANSI color codes
	resetColor   = []byte("\033[0m")
	levelToColor = [][]byte{
		LevelError: []byte("\033[0;31m"),
		LevelWarn:  []byte("\033[0;33m"),
		LevelInfo:  []byte("\033[0;37m"),
		LevelDebug: []byte("\033[0;34m"),
		LevelTrace: []byte("\033[0;35m"),
	}
)

// Entry represents a log entry and contains the timestamp when the entry
// was created, level, etc.
type Entry struct {
	Level  Level
	Time   time.Time
	Format string
	Values []interface{}
	File   string
	Line   int
}

// Logger is interface for `aah/log` package
type Logger interface {
	// Output writes the entry data into receiver
	Output(entry *Entry) error

	// Close closes the log writer. It cannot be used after this operation
	Close()

	// Closed returns true if the logger was previously closed
	Closed() bool

	// Stats returns current logger statistics like number of lines written,
	// number of bytes written, etc.
	Stats() *ReceiverStats

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

// New creates the logger based config supplied
func New(config string) (Logger, error) {
	if strIsEmpty(config) {
		return nil, errors.New("logger config is empty")
	}

	cfg, err := forge.ParseString(config)
	if err != nil {
		return nil, err
	}

	receiverType, err := cfg.GetString("receiver")
	if err != nil {
		return nil, err
	}
	receiverType = strings.ToUpper(receiverType)

	levelName, err := cfg.GetString("level")
	if err != nil {
		levelName = "DEBUG"
	}

	level := levelByName(levelName)
	if level == LevelUnknown {
		return nil, fmt.Errorf("unrecognized log level: %v", levelName)
	}

	pattern, err := cfg.GetString("pattern")
	if err != nil {
		pattern = DefaultPattern
	}

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
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}

	return file, line
}

func newConsoleReceiver(cfg *forge.Section, receiverType string, level Level, flags *[]FlagPart) (*Receiver, error) {
	receiver := Receiver{
		Config:     cfg,
		Type:       receiverType,
		Flags:      flags,
		Format:     DefaultFormatter,
		level:      level,
		out:        os.Stderr,
		stats:      &ReceiverStats{},
		isFileInfo: isFileFlagExists(flags),
		isLineInfo: isFmtFlagExists(flags, FmtFlagLine),
		isColor:    runtime.GOOS != "windows",
	}

	return &receiver, nil
}

func newFileReceiver(cfg *forge.Section, receiverType string, level Level, flags *[]FlagPart) (*Receiver, error) {
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

	rotate, _ := cfg.GetSection("rotate")
	receiver.rotate, _ = rotate.GetString("mode")
	switch receiver.rotate {
	case "daily":
		receiver.setOpenDay()
	case "lines":
		receiver.maxLines, _ = rotate.GetInteger("lines")
	case "size":
		receiver.maxSize, _ = rotate.GetInteger("size")
		receiver.maxSize = receiver.maxSize * 1024 * 1024
	}

	return &receiver, nil
}
