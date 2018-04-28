// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/ws source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ws is a WebSocket library for aah framework (RFC 6455 compliant).
//
// aah ws internally uses tiny, efficient WebSocket library
// http://github.com/gobwas/ws developed by Sergey Kamardin
// (https://github.com/gobwas).
package ws

import (
	"net/url"

	"aahframework.org/ainsp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
)

// New method creates aah WebSocket engine :)
func New(cfg *config.Config, logger log.Loggerer, router *router.Router) (*Engine, error) {
	eng := &Engine{
		cfg:    cfg,
		logger: logger,
		router: router,
		registry: &ainsp.TargetRegistry{
			Registry:   make(map[string]*ainsp.Target),
			SearchType: ctxPtrType,
		},
	}

	keyPrefix := "server.websocket"

	eng.checkOrigin = cfg.BoolDefault(keyPrefix+".origin.check", false)

	// parse whitelist origin urls
	eng.originWhitelist = make([]*url.URL, 0)
	if originWhitelist, found := cfg.StringList(keyPrefix + ".origin.whitelist"); found {
		for _, o := range originWhitelist {
			u, err := url.Parse(o)
			if err != nil {
				return nil, err
			}
			eng.originWhitelist = append(eng.originWhitelist, u)
		}
	}

	return eng, nil
}
