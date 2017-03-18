// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"reflect"

	"aahframework.org/log.v0"
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
	mwStack = mwStack[:len(mwStack)-3]
	mwStack = append(mwStack, middlewares...)
	mwStack = append(
		mwStack,
		templateMiddleware,
		interceptorMiddleware,
		actionMiddleware,
	)

	invalidateMwChain()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Middleware methods
//___________________________________

// Next method calls next middleware in the chain if available.
func (mw *Middleware) Next(c *Controller) {
	if c.abort {
		// abort, not to proceed further
		return
	}

	if mw.next != nil {
		mw.next(c, mw.further)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Interceptor middleware
//___________________________________

// interceptorMiddleware calls pre-defined actions (Before, Before<ActionName>,
// After, After<ActionName>, Panic, Panic<ActionName>, Finally,
// Finally<ActionName>) from controller.
func interceptorMiddleware(c *Controller, m *Middleware) {
	target := reflect.ValueOf(c.target)

	// Finally action and method
	defer func() {
		if finallyActionMethod := target.MethodByName(incpFinallyActionName + c.action.Name); finallyActionMethod.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", c.controller, incpFinallyActionName+c.action.Name)
			finallyActionMethod.Call(emptyArg)
		}

		if finallyAction := target.MethodByName(incpFinallyActionName); finallyAction.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", c.controller, incpFinallyActionName)
			finallyAction.Call(emptyArg)
		}
	}()

	// Panic action and method
	defer func() {
		if r := recover(); r != nil {
			if panicActionMethod := target.MethodByName(incpPanicActionName + c.action.Name); panicActionMethod.IsValid() {
				log.Debugf("Calling interceptor: %s.%s", c.controller, incpPanicActionName+c.action.Name)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicActionMethod.Call(rv)
			}

			if panicAction := target.MethodByName(incpPanicActionName); panicAction.IsValid() {
				log.Debugf("Calling interceptor: %s.%s", c.controller, incpPanicActionName)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicAction.Call(rv)
			} else { // propagate it
				panic(r)
			}
		}
	}()

	// Before action
	if beforeAction := target.MethodByName(incpBeforeActionName); beforeAction.IsValid() {
		log.Debugf("Calling interceptor: %s.%s", c.controller, incpBeforeActionName)
		beforeAction.Call(emptyArg)
	}

	// Before action method
	if !c.abort {
		if beforeActionMethod := target.MethodByName(incpBeforeActionName + c.action.Name); beforeActionMethod.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", c.controller, incpBeforeActionName+c.action.Name)
			beforeActionMethod.Call(emptyArg)
		}
	}

	m.Next(c)

	// After action method
	if !c.abort {
		if afterActionMethod := target.MethodByName(incpAfterActionName + c.action.Name); afterActionMethod.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", c.controller, incpAfterActionName+c.action.Name)
			afterActionMethod.Call(emptyArg)
		}
	}

	// After action
	if !c.abort {
		if afterAction := target.MethodByName(incpAfterActionName); afterAction.IsValid() {
			log.Debugf("Calling interceptor: %s.%s", c.controller, incpAfterActionName)
			afterAction.Call(emptyArg)
		}
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Action middleware
//___________________________________

// ActionMiddleware calls the requested action on controller
func actionMiddleware(c *Controller, m *Middleware) {
	target := reflect.ValueOf(c.target)
	action := target.MethodByName(c.action.Name)

	if !action.IsValid() {
		return
	}

	actionArgs := make([]reflect.Value, len(c.action.Parameters))

	// TODO Auto Binder for arguments

	log.Debugf("Calling controller: %s.%s", c.controller, c.action.Name)
	if action.Type().IsVariadic() {
		action.CallSlice(actionArgs)
	} else {
		action.Call(actionArgs)
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
		templateMiddleware,
		interceptorMiddleware,
		actionMiddleware,
	)

	invalidateMwChain()
}
