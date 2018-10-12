// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"reflect"

	"aahframe.work/essentials"
	"aahframe.work/log"
)

const (
	// Interceptor Action Name
	incpBeforeActionName  = "Before"
	incpAfterActionName   = "After"
	incpPanicActionName   = "Panic"
	incpFinallyActionName = "Finally"
)

// MiddlewareFunc func type is aah framework middleware signature.
type MiddlewareFunc func(ctx *Context, m *Middleware)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// ToMiddleware method expands the possibilities. It helps aah users to
// register the third-party or your own net/http middleware into `aah.MiddlewareFunc`.
//
// It is highly recommended to refactored to `aah.MiddlewareFunc`.
//
//    You can register below handler types:
//
//      1) aah.ToMiddleware(h http.Handler)
//
//      2) aah.ToMiddleware(h http.HandlerFunc)
//
//      3) aah.ToMiddleware(func(w http.ResponseWriter, r *http.Request))
func ToMiddleware(handler interface{}) MiddlewareFunc {
	switch handler.(type) {
	case MiddlewareFunc:
		return handler.(MiddlewareFunc)
	case http.Handler:
		h := handler.(http.Handler)
		return func(ctx *Context, m *Middleware) {
			h.ServeHTTP(ctx.Res, ctx.Req.Unwrap())
			m.Next(ctx)
		}
	case func(http.ResponseWriter, *http.Request):
		return ToMiddleware(http.HandlerFunc(handler.(func(http.ResponseWriter, *http.Request))))
	default:
		log.Errorf("Not a vaild handler: %s", ess.GetFunctionInfo(handler).QualifiedName)
		return nil
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Middleware methods
//______________________________________________________________________________

// Middleware struct is to implement aah framework middleware chain.
type Middleware struct {
	next    MiddlewareFunc
	further *Middleware
}

// Next method calls next middleware in the chain if available.
func (mw *Middleware) Next(ctx *Context) {
	if ctx.abort {
		// abort, not to proceed further
		return
	}

	if mw.next != nil {
		mw.next(ctx, mw.further)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// engine - Middleware
//______________________________________________________________________________

// Middlewares method adds given middleware into middleware stack
func (e *HTTPEngine) Middlewares(middlewares ...MiddlewareFunc) {
	e.mwStack = append(e.mwStack, middlewares...)

	e.invalidateMwChain()
}

func (e *HTTPEngine) invalidateMwChain() {
	e.mwChain = nil
	cnt := len(e.mwStack)
	e.mwChain = make([]*Middleware, cnt)

	for idx := 0; idx < cnt; idx++ {
		e.mwChain[idx] = &Middleware{next: e.mwStack[idx]}
	}

	for idx := cnt - 1; idx > 0; idx-- {
		e.mwChain[idx-1].further = e.mwChain[idx]
	}

	e.mwChain[cnt-1].further = &Middleware{}
}

type beforeInterceptor interface {
	Before()
}

type afterInterceptor interface {
	After()
}

type finallyInterceptor interface {
	Finally()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Action middleware
//______________________________________________________________________________

// ActionMiddleware performs
//	- Executes Interceptors (Before, Before<ActionName>, After, After<ActionName>,
//				Panic, Panic<ActionName>, Finally, Finally<ActionName>)
// 	- Invokes Controller Action
func ActionMiddleware(ctx *Context, m *Middleware) {
	if err := ctx.setTarget(ctx.route); err == errTargetNotFound {
		// No controller or action found for the route
		ctx.Reply().NotFound().Error(newError(ErrControllerOrActionNotFound, http.StatusNotFound))
		return
	}

	// Finally action and method. Always executed if present
	defer func() {
		if finallyActionMethod := ctx.targetrv.MethodByName(incpFinallyActionName + ctx.action.Name); finallyActionMethod.IsValid() {
			ctx.Log().Debugf("Calling interceptor: %s.%s", ctx.controller.FqName, incpFinallyActionName+ctx.action.Name)
			finallyActionMethod.Call(emptyArg)
		}

		// Finally: executes always if its implemented. Its applicable for every action in the controller
		if cntrl, ok := ctx.target.(finallyInterceptor); ok {
			ctx.Log().Debugf("Calling interceptor: %s.Finally", ctx.controller.FqName)
			cntrl.Finally()
		}
	}()

	// Panic action and method
	defer func() {
		if r := recover(); r != nil {
			// if ctx.abort {
			// 	return
			// }

			if panicActionMethod := ctx.targetrv.MethodByName(incpPanicActionName + ctx.action.Name); panicActionMethod.IsValid() {
				ctx.Log().Debugf("Calling interceptor: %s.%s", ctx.controller.FqName, incpPanicActionName+ctx.action.Name)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicActionMethod.Call(rv)
			} else if panicAction := ctx.targetrv.MethodByName(incpPanicActionName); panicAction.IsValid() {
				ctx.Log().Debugf("Calling interceptor: %s.%s", ctx.controller.FqName, incpPanicActionName)
				rv := append([]reflect.Value{}, reflect.ValueOf(r))
				panicAction.Call(rv)
			} else { // propagate it
				panic(r)
			}
		}
	}()

	// Before: executes before every action in the controller
	if cntrl, ok := ctx.target.(beforeInterceptor); ok {
		ctx.Log().Debugf("Calling interceptor: %s.Before", ctx.controller.FqName)
		cntrl.Before()
	}

	// Before action method
	if !ctx.abort {
		if beforeActionMethod := ctx.targetrv.MethodByName(incpBeforeActionName + ctx.action.Name); beforeActionMethod.IsValid() {
			ctx.Log().Debugf("Calling interceptor: %s.%s", ctx.controller.FqName, incpBeforeActionName+ctx.action.Name)
			beforeActionMethod.Call(emptyArg)
		}
	}

	if !ctx.abort {
		// Parse Action Parameters
		actionArgs, err := ctx.parseParameters()
		if err != nil { // Any error of parameter parsing result in 400 Bad Request
			ctx.Reply().BadRequest().Error(err)
			return
		}

		ctx.Log().Debugf("Calling action: %s.%s", ctx.controller.FqName, ctx.action.Name)
		ctx.actionrv.Call(actionArgs)
	}

	// After action method
	if !ctx.abort {
		if afterActionMethod := ctx.targetrv.MethodByName(incpAfterActionName + ctx.action.Name); afterActionMethod.IsValid() {
			ctx.Log().Debugf("Calling interceptor: %s.%s", ctx.controller.FqName, incpAfterActionName+ctx.action.Name)
			afterActionMethod.Call(emptyArg)
		}
	}

	// After: executes after every action in the controller
	if !ctx.abort {
		if cntrl, ok := ctx.target.(afterInterceptor); ok {
			ctx.Log().Debugf("Calling interceptor: %s.After", ctx.controller.FqName)
			cntrl.After()
		}
	}
}
