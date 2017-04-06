// +build go1.8

// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"context"
	"os"
	"strings"
	"time"

	"aahframework.org/log.v0"
)

// Shutdown method allows aah server to shutdown gracefully with given timeoout
// in seconds. It's invoked on OS signal `SIGINT` and `SIGTERM`.
//
// Method performs:
//    - Graceful server shutdown with timeout by `server.timeout.grace_shutdown`
//    - Publishes `OnShutdown` event
//    - Exits program with code 0
//
// Note: This applicable only to go1.8 and above.
func Shutdown() {
	graceTime := AppConfig().StringDefault("server.timeout.grace_shutdown", "60s")
	if !(strings.HasSuffix(graceTime, "s") || strings.HasSuffix(graceTime, "m")) {
		log.Warn("'server.timeout.grace_shutdown' value is not a valid time unit, assigning default")
		graceTime = "60s"
	}

	graceTimeout, _ = time.ParseDuration(graceTime)
	ctx, cancel := context.WithTimeout(context.Background(), graceTimeout)
	if err := aahServer.Shutdown(ctx); err != nil {
		log.Error(err)
	}

	// Publish `OnShutdown` event
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnShutdown})

	// Exit normally
	log.Infof("'%v' application stopped", AppName())
	cancel()
	os.Exit(0)
}
