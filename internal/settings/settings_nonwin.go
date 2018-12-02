// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// +build !windows

package settings

import (
	"os"
	"syscall"
)

// HotReloadSignal method returns type `os.Signal` based on config
// `runtime.config_hotreload.signal`. Default signal value is `SIGHUP`.
//
// Note: `SIGUSR1`, `SIGUSR2` is not applicable to Windows OS.
func (s *Settings) HotReloadSignal() os.Signal {
	if s.HotReloadSignalStr == "SIGUSR1" {
		return syscall.SIGUSR1
	}
	if s.HotReloadSignalStr == "SIGUSR2" {
		return syscall.SIGUSR2
	}
	return syscall.SIGHUP
}
