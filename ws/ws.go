// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ws is a WebSocket library for aah framework (RFC 6455 compliant).
//
// aah ws internally it uses tiny, efficient WebSocket library
// (http://github.com/gobwas/ws) developed by Sergey Kamardin
// (https://github.com/gobwas).
package ws

import (
	"fmt"
	"net/url"

	"aahframe.work/aah/ainsp"
)

// New method creates aah WebSocket engine with given aah application instance :)
func New(app interface{}) (*Engine, error) {
	a, ok := app.(application)
	if !ok {
		return nil, fmt.Errorf("ws: not a valid aah application instance")
	}

	eng := &Engine{
		app: a,
		registry: &ainsp.TargetRegistry{
			Registry:   make(map[string]*ainsp.Target),
			SearchType: ctxPtrType,
		},
	}

	keyPrefix := "server.websocket"

	eng.checkOrigin = a.Config().BoolDefault(keyPrefix+".origin.check", false)

	// parse whitelist origin urls
	eng.originWhitelist = make([]*url.URL, 0)
	if originWhitelist, found := a.Config().StringList(keyPrefix + ".origin.whitelist"); found {
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
