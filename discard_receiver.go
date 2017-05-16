// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/log source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import "aahframework.org/config.v0"

var _ Receiver = &DiscardReceiver{}

// DiscardReceiver is to throw the log entry.
type DiscardReceiver struct {
}

// Init method initializes the console logger.
func (d *DiscardReceiver) Init(_ *config.Config) error {
	return nil
}

// SetPattern method initializes the logger format pattern.
func (d *DiscardReceiver) SetPattern(_ string) error {
	return nil
}

// IsCallerInfo method returns true if log receiver is configured with caller info
// otherwise false.
func (d *DiscardReceiver) IsCallerInfo() bool {
	return false
}

// Log method writes the buf to
func (d *DiscardReceiver) Log(_ *Entry) {
}
