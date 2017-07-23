// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
)

var (
	// backupTimeFormat is used for timestamp with filename on rotation
	backupTimeFormat = "2006-01-02-15-04-05.000"

	_ Receiver = (*FileReceiver)(nil)
)

// FileReceiver writes the log entry into file.
type FileReceiver struct {
	filename     string
	out          io.Writer
	formatter    string
	flags        []ess.FmtFlagPart
	isCallerInfo bool
	stats        *receiverStats
	mu           *sync.Mutex
	isClosed     bool
	rotatePolicy string
	openDay      int
	isUTC        bool
	maxSize      int64
	maxLines     int64
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// FileReceiver methods
//___________________________________

// Init method initializes the file receiver instance.
func (f *FileReceiver) Init(cfg *config.Config) error {
	// File
	f.filename = cfg.StringDefault("log.file", "")
	if err := f.openFile(); err != nil {
		return err
	}

	f.formatter = cfg.StringDefault("log.format", "text")
	if !(f.formatter == textFmt || f.formatter == jsonFmt) {
		return fmt.Errorf("log: unsupported format '%s'", f.formatter)
	}

	f.rotatePolicy = cfg.StringDefault("log.rotate.policy", "daily")
	switch f.rotatePolicy {
	case "daily":
		f.openDay = f.getDay()
	case "lines":
		f.maxLines = int64(cfg.IntDefault("log.rotate.lines", 0))
	case "size":
		maxSize, err := ess.StrToBytes(cfg.StringDefault("log.rotate.size", "512mb"))
		if err != nil {
			return err
		}
		f.maxSize = maxSize
	}

	f.mu = &sync.Mutex{}

	return nil
}

// SetPattern method initializes the logger format pattern.
func (f *FileReceiver) SetPattern(pattern string) error {
	flags, err := ess.ParseFmtFlag(pattern, FmtFlags)
	if err != nil {
		return err
	}
	f.flags = flags
	if f.formatter == textFmt {
		f.isCallerInfo = isCallerInfo(f.flags)
	}
	f.isUTC = isFmtFlagExists(f.flags, FmtFlagUTCTime)
	f.openDay = f.getDay()
	return nil
}

// SetWriter method sets the given writer into file receiver.
func (f *FileReceiver) SetWriter(w io.Writer) {
	f.out = w
}

// IsCallerInfo method returns true if log receiver is configured with caller info
// otherwise false.
func (f *FileReceiver) IsCallerInfo() bool {
	return f.isCallerInfo
}

// Log method logs the given entry values into file.
func (f *FileReceiver) Log(entry *Entry) {
	f.mu.Lock()
	if f.isRotate() {
		_ = f.rotateFile()

		// reset rotation values
		f.openDay = f.getDay()
		f.stats.lines = 0
		f.stats.bytes = 0
	}
	f.mu.Unlock()

	msg := applyFormatter(f.formatter, f.flags, entry)
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		msg = append(msg, '\n')
	}

	size, _ := f.out.Write(msg)

	// calculate receiver stats
	f.stats.bytes += int64(size)
	f.stats.lines++
}

// Writer method returns the current log writer.
func (f *FileReceiver) Writer() io.Writer {
	return f.out
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// FileReceiver Unexported methods
//___________________________________

func (f *FileReceiver) isRotate() bool {
	switch f.rotatePolicy {
	case "daily":
		return f.openDay != f.getDay()
	case "lines":
		return f.maxLines != 0 && f.stats.lines >= f.maxLines
	case "size":
		return f.maxSize != 0 && f.stats.bytes >= f.maxSize
	default:
		return false
	}
}

func (f *FileReceiver) rotateFile() error {
	if _, err := os.Lstat(f.filename); err == nil {
		f.close()
		if err = os.Rename(f.filename, f.backupFileName()); err != nil {
			return err
		}
	}

	if err := f.openFile(); err != nil {
		return err
	}

	return nil
}

func (f *FileReceiver) openFile() error {
	dir := filepath.Dir(f.filename)
	_ = ess.MkDirAll(dir, filePermission)

	file, err := os.OpenFile(f.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePermission)
	if err != nil {
		return err
	}

	fileStat, err := file.Stat()
	if err != nil {
		return err
	}

	f.SetWriter(file)
	f.isClosed = false
	f.stats = &receiverStats{}
	f.stats.bytes = fileStat.Size()
	f.stats.lines = int64(ess.LineCntr(file))

	return nil
}

func (f *FileReceiver) close() {
	if !f.isClosed {
		ess.CloseQuietly(f.out)
		f.isClosed = true
	}
}

func (f *FileReceiver) backupFileName() string {
	dir := filepath.Dir(f.filename)
	fileName := filepath.Base(f.filename)
	ext := filepath.Ext(fileName)
	baseName := ess.StripExt(fileName)
	t := time.Now()
	if f.isUTC {
		t = t.UTC()
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", baseName, t.Format(backupTimeFormat), ext))
}

func (f *FileReceiver) getDay() int {
	if f.isUTC {
		return time.Now().UTC().Day()
	}
	return time.Now().Day()
}
