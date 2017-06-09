// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"
)

var (
	entryPool = &sync.Pool{New: func() interface{} { return &Entry{} }}
	bufPool   = &sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}
)

// Entry represents a log entry and contains the timestamp when the entry
// was created, level, etc.
type Entry struct {
	Level   level     `json:"level,omitempty"`
	Time    time.Time `json:"timestamp,omitempty"`
	Message string    `json:"message,omitempty"`
	File    string    `json:"file,omitempty"`
	Line    int       `json:"line,omitempty"`
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Entry methods
//___________________________________

// MarshalJSON method for formating entry to JSON.
func (e *Entry) MarshalJSON() ([]byte, error) {
	type Alias Entry
	return json.Marshal(&struct {
		Level string `json:"level,omitempty"`
		Time  string `json:"timestamp,omitempty"`
		*Alias
	}{
		Level: levelToLevelName[e.Level],
		Time:  formatTime(e.Time),
		Alias: (*Alias)(e),
	})
}

// Reset method resets the `Entry` values for reuse.
func (e *Entry) Reset() {
	e.Level = LevelUnknown
	e.Time = time.Time{}
	e.Message = ""
	e.File = ""
	e.Line = 0
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// getEntry gets entry object from pool.
func getEntry() *Entry {
	return entryPool.Get().(*Entry)
}

// putEntry reset the entry and puts it into Pool.
func putEntry(e *Entry) {
	e.Reset()
	entryPool.Put(e)
}

// getBuffer gets buffer object from pool.
func getBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

// putBuffer reset the buffer and puts it into Pool.
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}
