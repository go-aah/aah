// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-aah/config"
	"github.com/go-aah/essentials"
)

// Receiver represents aah logger object that statisfy console, file, logging
// and Logging Stats. Each logging operation makes a single call to the
// Writer's Write method. `Receiver` guarantees serialize access and
// it can be used from multiple goroutines like Go standard logger.
type Receiver struct {
	Config *config.Config
	Type   string
	Flags  *[]FlagPart
	Format FormatterFunc

	m          sync.Mutex
	level      Level
	out        io.Writer
	stats      *ReceiverStats
	isFileInfo bool
	isLineInfo bool
	isUTC      bool

	// Console Receiver
	isColor bool

	// File Receiver
	isClosed bool
	rotate   string
	openDay  int
	maxSize  int64
	maxLines int64
}

// Output formats the give log inputs, caller info and writes to console
func (r *Receiver) Output(entry *Entry) error {
	if r.Closed() {
		return ErrWriterIsClosed
	}

	if r.isFileInfo || r.isLineInfo {
		entry.File, entry.Line = fetchCallerInfo(3)
	}

	r.m.Lock()
	defer r.m.Unlock()

	// Check log rotation is required
	if r.isRotateRequired() {
		if err := r.rotateFile(); err != nil {
			return err
		}
	}

	// format the log entry message as per pattern
	buf, err := r.Format(r.Flags, entry, r.isColor)
	if err != nil {
		return err
	}

	// writes bytes into writer
	size, err := r.out.Write(*buf)
	if err != nil {
		return err
	}

	// calculate receiver stats
	if r.stats != nil {
		r.stats.bytes += int64(size)
		r.stats.lines++
	}

	return nil
}

// Stats returns current logger statistics like number of lines written,
// number of bytes written, etc.
func (r *Receiver) Stats() *ReceiverStats {
	return r.stats
}

// Close closes the log writer. It cannot be used after this operation
func (r *Receiver) Close() {
	if r.isClosed {
		return
	}

	if out, ok := r.out.(io.Closer); ok {
		r.isClosed = true
		_ = out.Close()
	}
}

// Closed returns true if the logger was previously closed
func (r *Receiver) Closed() bool {
	return r.isClosed
}

// SetPattern sets the pattern to log entry format
func (r *Receiver) SetPattern(pattern string) error {
	r.m.Lock()
	defer r.m.Unlock()
	flags, err := parseFlag(pattern)
	if err != nil {
		return err
	}
	r.Flags = flags

	return nil
}

func (r *Receiver) isFileReceiver() bool {
	return r.Type == "FILE"
}

// Error logs message as `LevelError`
func (r *Receiver) Error(v ...interface{}) {
	if r.level >= LevelError {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelError,
			Values: v,
		})
	}
}

// Errorf logs message as `LevelError`
func (r *Receiver) Errorf(format string, v ...interface{}) {
	if r.level >= LevelError {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelError,
			Format: format,
			Values: v,
		})
	}
}

// Warn logs message as `LevelWarn`
func (r *Receiver) Warn(v ...interface{}) {
	if r.level >= LevelWarn {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelWarn,
			Values: v,
		})
	}
}

// Warnf logs message as `LevelWarn`
func (r *Receiver) Warnf(format string, v ...interface{}) {
	if r.level >= LevelWarn {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelWarn,
			Format: format,
			Values: v,
		})
	}
}

// Info logs message as `LevelInfo`
func (r *Receiver) Info(v ...interface{}) {
	if r.level >= LevelInfo {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelInfo,
			Values: v,
		})
	}
}

// Infof logs message as `LevelInfo`
func (r *Receiver) Infof(format string, v ...interface{}) {
	if r.level >= LevelInfo {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelInfo,
			Format: format,
			Values: v,
		})
	}
}

// Debug logs message as `LevelDebug`
func (r *Receiver) Debug(v ...interface{}) {
	if r.level >= LevelDebug {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelDebug,
			Values: v,
		})
	}
}

// Debugf logs message as `LevelDebug`
func (r *Receiver) Debugf(format string, v ...interface{}) {
	if r.level >= LevelDebug {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelDebug,
			Format: format,
			Values: v,
		})
	}
}

// Trace logs message as `LevelTrace`
func (r *Receiver) Trace(v ...interface{}) {
	if r.level >= LevelTrace {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelTrace,
			Values: v,
		})
	}
}

// Tracef logs message as `LevelTrace`
func (r *Receiver) Tracef(format string, v ...interface{}) {
	if r.level >= LevelTrace {
		_ = r.Output(&Entry{
			Time:   time.Now(),
			Level:  LevelTrace,
			Format: format,
			Values: v,
		})
	}
}

// unexported methods

func (r *Receiver) openFile() error {
	if !r.isFileReceiver() {
		return nil
	}

	name := r.fileName()
	dir := filepath.Dir(name)
	_ = ess.MkDirAll(dir, 0755)

	file, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePermission)
	if err != nil {
		return err
	}

	fileStat, err := file.Stat()
	if err != nil {
		return err

	}

	r.isClosed = false
	r.setOpenDay()
	r.stats.bytes = fileStat.Size()
	r.stats.lines = int64(ess.LineCntr(file))
	r.out = file

	return nil
}

func (r *Receiver) fileName() string {
	return r.Config.StringDefault("file", "aah-log-file.log")
}

func (r *Receiver) backupFileName() string {
	name := r.fileName()
	dir := filepath.Dir(name)
	fileName := filepath.Base(name)
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)

	t := time.Now()
	if r.isUTC {
		t = t.UTC()
	}

	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", baseName, t.Format(BackupTimeFormat), ext))
}

func (r *Receiver) setOpenDay() {
	if r.isUTC {
		r.openDay = time.Now().UTC().Day()
	} else {
		r.openDay = time.Now().Day()
	}
}

func (r *Receiver) isRotateRequired() bool {
	if !r.isFileReceiver() {
		return false
	}

	switch r.rotate {
	case "daily":
		if r.isUTC {
			return time.Now().UTC().Day() != r.openDay
		}
		return time.Now().Day() != r.openDay
	case "lines":
		return r.maxLines != 0 && r.stats.lines >= r.maxLines
	case "size":
		return r.maxSize != 0 && r.stats.bytes >= r.maxSize
	}

	return false
}

func (r *Receiver) rotateFile() error {
	if !r.isFileReceiver() {
		return nil
	}

	fileName := r.fileName()
	if _, err := os.Lstat(fileName); err == nil {
		r.Close()
		if err = os.Rename(fileName, r.backupFileName()); err != nil {
			return err
		}
	}

	if err := r.openFile(); err != nil {
		return err
	}

	return nil
}
