// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"sync"

	"aahframework.org/aah/ahttp"
	"aahframework.org/log"
)

var (
	mwStack []MiddlewareType
	mwChain []*Middleware
)

type (
	// Engine is the aah framework application server handler for request and response.
	// Implements `http.Handler` interface.
	engine struct {
		cPool sync.Pool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// Middlewares method adds given middleware into middleware stack
func Middlewares(middlewares ...MiddlewareType) {
	mwStack = mwStack[:len(mwStack)-1]
	mwStack = append(mwStack, middlewares...)
	mwStack = append(mwStack, dispatchMiddleware)

	invalidateMwChain()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := e.cPool.Get().(*Controller)
	defer e.cPool.Put(c)
	c.reset()

	c.Req = ahttp.ParseRequest(r)
	c.res = ahttp.WrapResponseWriter(w)

	log.Info("ServeHTTP called:", c.Req.Path)

	mwChain[0].Next(c)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func invalidateMwChain() {
	mwChain = nil
	cnt := len(mwStack)
	mwChain = make([]*Middleware, cnt, cnt)

	for idx := 0; idx < cnt; idx++ {
		mwChain[idx] = &Middleware{next: mwStack[idx]}
	}

	for idx := cnt - 1; idx > 0; idx-- {
		mwChain[idx-1].further = mwChain[idx]
	}

	mwChain[cnt-1].further = &Middleware{}
}

func init() {
	mwStack = append(mwStack,
		panicMiddleware,
		routerMiddleware,
		paramsMiddleware,
		dispatchMiddleware,
	)

	invalidateMwChain()
}
