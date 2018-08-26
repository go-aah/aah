// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"aahframe.work/aah/config"
	"aahframe.work/aah/essentials"
)

var (
	// ANSI color codes
	resetColor   = []byte("\033[0m")
	levelToColor = [][]byte{
		LevelFatal: []byte("\033[0;31m"), // red
		LevelPanic: []byte("\033[0;31m"), // red
		LevelError: []byte("\033[0;31m"), // red
		LevelWarn:  []byte("\033[0;33m"), // yellow
		LevelInfo:  []byte("\033[0;37m"), // white
		LevelDebug: []byte("\033[0;36m"), // cyan
		LevelTrace: []byte("\033[0;35m"), // magenta (purple)
	}

	_ Receiver = (*ConsoleReceiver)(nil)
)

// ConsoleReceiver writes the log entry into os.Stderr.
// For non-windows it  writes with color.
type ConsoleReceiver struct {
	out          io.Writer
	formatter    string
	flags        []ess.FmtFlagPart
	isCallerInfo bool
	isColor      bool
	mu           sync.Mutex
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ConsoleReceiver methods
//___________________________________

// Init method initializes the console logger.
func (c *ConsoleReceiver) Init(cfg *config.Config) error {
	c.out = os.Stderr
	c.isColor = runtime.GOOS != "windows"

	if v, found := cfg.Bool("log.color"); found {
		c.isColor = v
	}

	c.formatter = cfg.StringDefault("log.format", "text")
	if !(c.formatter == textFmt || c.formatter == jsonFmt) {
		return fmt.Errorf("log: unsupported format '%s'", c.formatter)
	}

	c.mu = sync.Mutex{}

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

// SetWriter method sets the given writer into console receiver.
func (c *ConsoleReceiver) SetWriter(w io.Writer) {
	c.out = w
}

// IsCallerInfo method returns true if log receiver is configured with caller info
// otherwise false.
func (c *ConsoleReceiver) IsCallerInfo() bool {
	return c.isCallerInfo
}

// Log method writes the log entry into os.Stderr.
func (c *ConsoleReceiver) Log(entry *Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isColor {
		_, _ = c.out.Write(levelToColor[entry.Level])
	}

	var msg []byte
	if c.formatter == textFmt {
		msg = textFormatter(c.flags, entry)
	} else {
		msg, _ = json.Marshal(entry)
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
