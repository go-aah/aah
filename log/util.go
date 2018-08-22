// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"runtime"
	"strings"
	"time"

	"aahframe.work/aah/essentials"
)

var (
	levelNameToLevel = map[string]level{
		"FATAL": LevelFatal,
		"PANIC": LevelPanic,
		"ERROR": LevelError,
		"WARN":  LevelWarn,
		"INFO":  LevelInfo,
		"DEBUG": LevelDebug,
		"TRACE": LevelTrace,
	}

	levelToLevelName = map[level]string{
		LevelFatal: "FATAL",
		LevelPanic: "PANIC",
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

// String level string interface.
func (l level) String() string {
	return levelToLevelName[l]
}

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

func fetchCallerInfo() (string, int) {
	// dynamic call depth calculation; skip 3 known path and get
	// maximum 5 which would cover log package.
	pc := make([]uintptr, 5)
	n := runtime.Callers(3, pc)
	if n == 0 {
		// No pcs available. Stop now.
		// This can happen if the first argument to runtime.Callers is large.
		return "???", 0
	}

	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)

	// Loop to get frames.
	// A fixed number of pcs can expand to an indefinite number of Frames.
	for {
		frame, _ := frames.Next()

		// Unwinding for aah log pkg otherwise stop.
		if strings.Contains(frame.File, "aahframe.work/aah/log") {
			continue
		}

		return frame.File, frame.Line
	}
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
