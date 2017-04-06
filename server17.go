// +build !go1.8

// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"os"

	"aahframework.org/log.v0"
)

// Shutdown method causes aah application to stop. IT DOES NOT SUPPORT GRACEFUL.
// It's invoked on OS signal `SIGINT` and `SIGTERM`.
//
// Method performs:
//    - Publishes `OnShutdown` event
//    - Exits program with code 0
//
// Note: This applicable only to go1.7
func Shutdown() {
	// Publish `OnShutdown` event
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnShutdown})

	// Exit normally
	log.Infof("'%v' application stopped", AppName())
	os.Exit(0)
}
