// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0-unstable"
)

var (
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

	_ Receiver = &ConsoleReceiver{}
)

// ConsoleReceiver writes the log entry into os.Stderr.
// For non-windows it  writes with color.
type ConsoleReceiver struct {
	out          io.Writer
	formatter    string
	flags        []ess.FmtFlagPart
	isCallerInfo bool
	isColor      bool
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ConsoleReceiver methods
//___________________________________

// Init method initializes the console logger.
func (c *ConsoleReceiver) Init(cfg *config.Config) error {
	c.out = os.Stderr
	c.isColor = runtime.GOOS != "windows"

	c.formatter = cfg.StringDefault("log.format", "text")
	if !(c.formatter == textFmt || c.formatter == jsonFmt) {
		return fmt.Errorf("log: unsupported format '%s'", c.formatter)
	}

	return nil
}

// SetPattern method initializes the logger format pattern.
func (c *ConsoleReceiver) SetPattern(pattern string) error {
	flags, err := ess.ParseFmtFlag(pattern, FmtFlags)
	if err != nil {
		return err
	}
	c.flags = flags
	if c.formatter == textFmt {
		c.isCallerInfo = isCallerInfo(c.flags)
	}
	return nil
}

// IsCallerInfo method returns true if log receiver is configured with caller info
// otherwise false.
func (c *ConsoleReceiver) IsCallerInfo() bool {
	return c.isCallerInfo
}

// Log method writes the log entry into os.Stderr.
func (c *ConsoleReceiver) Log(entry *Entry) {
	if c.isColor {
		_, _ = c.out.Write(levelToColor[entry.Level])
	}

	msg := applyFormatter(c.formatter, c.flags, entry)
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg = append(msg, '\n')
	}
	_, _ = c.out.Write(msg)

	if c.isColor {
		_, _ = c.out.Write(resetColor)
	}
}

// Writer method returns the current log writer.
func (c *ConsoleReceiver) Writer() io.Writer {
	return c.out
}
