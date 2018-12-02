// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// +build windows

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
	return syscall.SIGHUP
}
