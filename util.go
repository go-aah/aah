// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"runtime"
	"strings"
	"time"

	"aahframework.org/essentials.v0"
)

var (
	levelNameToLevel = map[string]level{
		"FATAL": levelFatal,
		"PANIC": levelPanic,
		"ERROR": LevelError,
		"WARN":  LevelWarn,
		"INFO":  LevelInfo,
		"DEBUG": LevelDebug,
		"TRACE": LevelTrace,
	}

	levelToLevelName = map[level]string{
		levelFatal: "FATAL",
		levelPanic: "PANIC",
		LevelError: "ERROR",
		LevelWarn:  "WARN",
		LevelInfo:  "INFO",
		LevelDebug: "DEBUG",
		LevelTrace: "TRACE",
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func levelByName(name string) level {
	if level, ok := levelNameToLevel[strings.ToUpper(name)]; ok {
		return level
	}

	return LevelUnknown
}

func isFmtFlagExists(flags []ess.FmtFlagPart, flag ess.FmtFlag) bool {
	for _, f := range flags {
		if f.Flag == flag {
			return true
		}
	}
	return false
}

func fetchCallerInfo(calldepth int) (string, int) {
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	return file, line
}

// isCallerInfo method to identify to fetch caller or not.
func isCallerInfo(flags []ess.FmtFlagPart) bool {
	return (isFmtFlagExists(flags, FmtFlagShortfile) ||
		isFmtFlagExists(flags, FmtFlagLongfile) ||
		isFmtFlagExists(flags, FmtFlagLine))
}

func getReceiverByName(name string) Receiver {
	switch name {
	case "FILE":
		return &FileReceiver{}
	case "CONSOLE":
		return &ConsoleReceiver{}
	default:
		return nil
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
