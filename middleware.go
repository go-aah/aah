// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"reflect"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/router"
	"aahframework.org/essentials"
	"aahframework.org/log"
)

var (
	mwStack []MiddlewareType
	mwChain []*Middleware
)

type (
	// MiddlewareType func type is aah framework middleware signature.
	MiddlewareType func(c *Controller, m *Middleware)

	// Middleware struct is to implement aah framework middleware chain.
	Middleware struct {
		next    MiddlewareType
		further *Middleware
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// Middlewares method adds given middleware into middleware stack
func Middlewares(middlewares ...MiddlewareType) {
	mwStack = mwStack[:len(mwStack)-1]
	mwStack = append(mwStack, middlewares...)
	mwStack = append(mwStack, actionMiddleware)

	invalidateMwChain()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Middleware methods
//___________________________________

// Next method calls next middleware in the chain if available.
func (mw *Middleware) Next(c *Controller) {
	if mw.next != nil {
		mw.next(c, mw.further)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router middleware
//___________________________________

// RouterMiddleware finds the route of incoming request and moves forward.
// If routes not found it does appropriate response for the request.
func routerMiddleware(c *Controller, m *Middleware) {
	domain := router.FindDomain(c.Req)
	if domain == nil {
		c.res.WriteHeader(http.StatusNotFound) // TODO change it after Reply module is done
		_, _ = c.res.Write([]byte("404 Route Not Exists\n"))
		return
	}

	route, pathParams, rts := domain.Lookup(c.Req)
	if route != nil { // route found
		if route.IsStatic {
			if err := serveStatic(c, route, pathParams); err == errFileNotFound {
				handleNotFound(c, domain, route.IsStatic)
			}
			return
		}

		if err := c.setTarget(route); err == errTargetNotFound {
			handleNotFound(c, domain, false)
			return
		}

		c.pathParams = pathParams
		m.Next(c)

		return
	}

	// Redirect Trailing Slash
	if c.Req.Method != ahttp.MethodConnect && c.Req.Path != router.SlashString {
		if rts && domain.RedirectTrailingSlash {
			redirectTrailingSlash(c)
			return
		}
	}

	// HTTP: OPTIONS
	if c.Req.Method == ahttp.MethodOptions {
		if domain.AutoOptions {
			if allowed := domain.Allowed(c.Req.Method, c.Req.Path); !ess.IsStrEmpty(allowed) {
				log.Debugf("Auto 'OPTIONS' allowed HTTP Methods: %s", allowed)
				c.res.Header().Set(ahttp.HeaderAllow, allowed) // TODO change it after Reply module is done
				return
			}
		}
	}

	// 405 Method Not Allowed
	if domain.MethodNotAllowed {
		if allowed := domain.Allowed(c.Req.Method, c.Req.Path); !ess.IsStrEmpty(allowed) {
			allowed += ", " + ahttp.MethodOptions
			log.Debugf("Allowed HTTP Methods: %s", allowed)

			c.res.Header().Set(ahttp.HeaderAllow, allowed) // TODO change it after Reply module is done
			c.res.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = c.res.Write([]byte("405 Method Not Allowed\n"))

			return
		}
	}

	// 404 not found
	handleNotFound(c, domain, false)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Params middleware
//___________________________________

// ParamsMiddleware parses the incoming HTTP request to collects request
// parameters (query string and payload) stores into controller. Query string
// parameters made available in render context.
func paramsMiddleware(c *Controller, m *Middleware) {

	log.Info("paramsMiddleware before")

	m.Next(c)

	log.Info("paramsMiddleware after")

}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Action middleware
//___________________________________

// ActionMiddleware calls the requested action on controller and calls
// pre-defined actions (Before, After, Panic, Finally) from controller.
func actionMiddleware(c *Controller, m *Middleware) {
	target := reflect.ValueOf(c.target)

	// Finally action
	defer func() {
		if finallyAction := target.MethodByName(incpFinallyActionName); finallyAction.IsValid() {
			log.Debugf("Calling finally interceptor on controller: %s", c.controller)
			finallyAction.Call(emptyArg)
		}
	}()

	// Panic action
	defer func() {
		if r := recover(); r != nil {
			if panicAction := target.MethodByName(incpPanicActionName); panicAction.IsValid() {
				log.Debugf("Calling panic interceptor on controller: %s", c.controller)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicAction.Call(rv)
			} else { // propagate it
				panic(r)
			}
		}
	}()

	// Before action
	if beforeAction := target.MethodByName(incpBeforeActionName); beforeAction.IsValid() {
		log.Debugf("Calling before interceptor on controller: %s", c.controller)
		beforeAction.Call(emptyArg)
	}

	action := target.MethodByName(c.action.Name)
	if action.IsValid() {
		actionArgs := make([]reflect.Value, len(c.action.Parameters))

		// TODO Auto Binder for arguments

		log.Debugf("Calling controller: %s, action: %s", c.controller, c.action.Name)
		if action.Type().IsVariadic() {
			action.CallSlice(actionArgs)
		} else {
			action.Call(actionArgs)
		}
	}

	// After action
	if afterAction := target.MethodByName(incpAfterActionName); afterAction.IsValid() {
		log.Debugf("Calling after interceptor on controller: %s", c.controller)
		afterAction.Call(emptyArg)
	}
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
		routerMiddleware,
		paramsMiddleware,
		actionMiddleware,
	)

	invalidateMwChain()
}
