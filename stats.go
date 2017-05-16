// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

// receiverStats tracks the number of output lines and bytes written.
type receiverStats struct {
	lines int64
	bytes int64
}

// Lines returns the number of lines written.
func (s *receiverStats) Lines() int64 {
	return s.lines
}

// Bytes returns the number of bytes written.
func (s *receiverStats) Bytes() int64 {
	return s.bytes
}
