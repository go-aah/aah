// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"os"

	"aahframework.org/aah/aruntime"
	"aahframework.org/log"
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
// Middleware methods
//___________________________________

// Next method calls next middleware in the chain if available.
func (mw *Middleware) Next(c *Controller) {
	if mw.next != nil {
		mw.next(c, mw.further)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Painc middleware
//___________________________________

// PanicMiddleware handles panic calls and recovers from it then converts
// panic into HTTP Internal Server Error (Status 500).
func panicMiddleware(c *Controller, m *Middleware) {
	log.Info("panicMiddleware before")
	defer func() {
		if r := recover(); r != nil {
			log.Error("Internal server error occurred")

			st := aruntime.NewStacktrace(r, AppConfig())
			st.Print(os.Stdout)

			// TODO HTTP error handling
		}
	}()

	m.Next(c)

	log.Info("panicMiddleware after")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Router middleware
//___________________________________

// RouterMiddleware finds the route of incoming request and moves forward.
// If routes not found it does appropriate response for the request.
func routerMiddleware(c *Controller, m *Middleware) {

	log.Info("routerMiddleware before")

	m.Next(c)

	log.Info("routerMiddleware after")

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
// Dispatch middleware
//___________________________________

// DispatchMiddleware calls the requested action on controller and calls
// pre-defined actions (Before, After, Error, Finally) from controller.
// It is Last middleware in the aah framework middleware chain.
func dispatchMiddleware(c *Controller, m *Middleware) {

	log.Info("dispatchMiddleware")
	c.res.WriteHeader(200)
	_, _ = c.res.Write([]byte("OK"))

}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________
