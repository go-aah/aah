// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"net/http"
	"sync"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/aruntime"
	"aahframework.org/log"
)

type (
	// Engine is the aah framework application server handler for request and response.
	// Implements `http.Handler` interface.
	engine struct {
		cPool sync.Pool
		rPool sync.Pool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := e.cPool.Get().(*Controller)
	r := e.rPool.Get().(*ahttp.Request)
	defer e.put(c, r)

	e.reset(c, r)
	c.Req = ahttp.ParseRequest(req, r)
	c.res = ahttp.WrapResponseWriter(w)

	// panic handling
	defer e.handlePanic(c)

	log.Debugf("Request for %s", c.Req.Path)

	// Middlewares
	mwChain[0].Next(c)

	// Write response
	e.writeResponse(c)
}

// handlePanic handles application panics and recovers from it. Panic gets
// translated into HTTP Internal Server Error (Status 500).
func (e *engine) handlePanic(c *Controller) {
	if r := recover(); r != nil {
		log.Errorf("Internal server error occurred: %s", c.Req.Path)

		st := aruntime.NewStacktrace(r, AppConfig())
		buf := &bytes.Buffer{}
		st.Print(buf)

		log.Error("Recovered from panic:")
		log.Error(buf.String())

		c.res.WriteHeader(http.StatusInternalServerError)
		c.res.WriteHeaderNow()

		_, _ = c.res.Write([]byte("Internal server error occurred\n"))

		// TODO HTTP error handling for panic
	}
}

// writeResponse method writes response.
func (e *engine) writeResponse(c *Controller) {
	c.res.WriteHeaderNow()
	c.res.Write([]byte("from writeResponse method\n"))
}

// put method puts object back to pool
func (e *engine) put(c *Controller, r *ahttp.Request) {
	e.cPool.Put(c)
	e.rPool.Put(r)
}

// reset method resets a obj for next use
func (e *engine) reset(c *Controller, r *ahttp.Request) {
	c.Reset()
	r.Reset()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func newCPool() sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return &Controller{}
		},
	}
}

func newRPool() sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return &ahttp.Request{}
		},
	}
}
