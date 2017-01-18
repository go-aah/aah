// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"reflect"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/router"
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
		fmt.Println("domain not found") // TODO no domain mapping
	}

	route, pathParams, rts := domain.Lookup(c.Req)
	if route != nil {
		if err := c.setTarget(route); err == errTargetNotFound {
			// TODO Action Not found
			fmt.Println("Action not found")
			return
		}

		c.pathParams = pathParams
		m.Next(c)

		return
	}

	if c.Req.Method != ahttp.MethodConnect && c.Req.Path != router.SlashString {
		if rts && domain.RedirectTrailingSlash {
			redirectTrailingSlash(c)
			return
		}
	}

	if domain.MethodNotAllowed {
		allowed := domain.Allowed(c.Req.Method, c.Req.Path)
		log.Debugf("Allowed HTTP Methods: %s", allowed)
		c.res.Header().Set(ahttp.HeaderAllow, allowed) // TODO change it after Reply module is donw
		return
	}

	// TODO 404 not found
}

// Redirect method redirects request to given URL.
func redirectTrailingSlash(c *Controller) {
	code := http.StatusMovedPermanently
	if c.Req.Method != ahttp.MethodGet {
		code = http.StatusTemporaryRedirect
	}

	path := c.Req.Path
	req := c.Req.Raw
	if len(path) > 1 && path[len(path)-1] == '/' {
		req.URL.Path = path[:len(path)-1]
	} else {
		req.URL.Path = path + "/"
	}

	log.Debugf("RedirectTrailingSlash: %d, %s ==> %s", code, path, req.URL.String())
	http.Redirect(c.res, req, req.URL.String(), code)
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
			rv := append([]reflect.Value{}, reflect.ValueOf(r))
			if panicAction := target.MethodByName(incpPanicActionName); panicAction.IsValid() {
				log.Debugf("Calling panic interceptor on controller: %s", c.controller)
				panicAction.Call(rv)
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
