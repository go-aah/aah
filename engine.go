// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/aruntime"
	"aahframework.org/aah/router"
	"aahframework.org/essentials"
	"aahframework.org/log"
	"aahframework.org/pool"
)

var errFileNotFound = errors.New("file not found")

type (
	// Engine is the aah framework application server handler for request and response.
	// Implements `http.Handler` interface.
	engine struct {
		cPool *pool.Pool
		rPool *pool.Pool
		bPool *pool.Pool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Debugf("Request for %s", req.URL.Path)

	c, r := e.getController(), e.getRequest()
	defer e.putController(c)
	defer e.putRequest(r)

	c.Req = ahttp.ParseRequest(req, r)
	c.res = ahttp.WrapResponseWriter(w)
	c.reply = &Reply{}

	// panic handling
	defer e.handlePanic(c)

	// Middlewares
	mwChain[0].Next(c)

	// Write response
	e.writeResponse(c)
}

// handlePanic handles application panics and recovers from it. Panic gets
// translated into HTTP Internal Server Error (Status 500).
func (e *engine) handlePanic(c *Controller) {
	if r := recover(); r != nil {
		log.Errorf("Internal Server Error for %s", c.Req.Path)

		st := aruntime.NewStacktrace(r, AppConfig())
		buf := e.getBuffer()
		defer e.putBuffer(buf)

		st.Print(buf)

		log.Error("Recovered from panic:")
		log.Error(buf.String())

		if AppProfile() != "prod" { // detailed error info
			// TODO design server error page with stack trace info
			return
		}

		// For "prod", detailed information gets logged
		c.res.WriteHeader(http.StatusInternalServerError)
		_, _ = c.res.Write([]byte("Internal Server Error\n"))
	}
}

// writeResponse method writes the response on the wire based on `Reply` values.
func (e *engine) writeResponse(c *Controller) {
	defer c.res.(*ahttp.Response).Close()

	re := c.Reply()
	buf := e.getBuffer()
	defer e.putBuffer(buf)

	// Render and detect the errors earlier, framework can write error info
	// without messing with response.
	// HTTP Body
	if re.render != nil {
		if err := re.render.Render(buf); err != nil {
			log.Error("Render error:", err)
			c.res.WriteHeader(http.StatusInternalServerError)
			_, _ = c.res.Write([]byte("Render error: " + err.Error() + "\n"))
			return
		}
	}

	// HTTP headers
	for k, v := range re.header {
		for _, vv := range v {
			c.res.Header().Add(k, vv)
		}
	}

	// Content Type
	if !ess.IsStrEmpty(re.contentType) {
		c.res.Header().Set(ahttp.HeaderContentType, re.contentType)
	} else if !ess.IsStrEmpty(c.Req.AcceptContentType.Mime) {
		// based on 'Accept' Header
		c.res.Header().Set(ahttp.HeaderContentType, c.Req.AcceptContentType.Raw())
	} else {
		// default Content-Type defined 'render.default' in aah.conf
		c.res.Header().Set(ahttp.HeaderContentType, e.defaultContentType().Raw())
	}

	// HTTP status
	if re.IsStatusSet() {
		c.res.WriteHeader(re.status)
	} else {
		c.res.WriteHeader(http.StatusOK)
	}

	// Write it on the wire
	_, _ = buf.WriteTo(c.res)
}

// defaultContentType method returns the Content-Type based on 'render.default'
// config from aah.conf
func (e *engine) defaultContentType() *ahttp.ContentType {
	cfgValue := AppConfig().StringDefault("render.default", "")
	switch cfgValue {
	case "html":
		return ahttp.ContentTypeHTML
	case "json":
		return ahttp.ContentTypeJSON
	case "xml":
		return ahttp.ContentTypeXML
	case "text":
		return ahttp.ContentTypePlainText
	default:
		return ahttp.ContentTypeOctetStream
	}
}

// getController method gets controller from pool
func (e *engine) getController() *Controller {
	return e.cPool.Get().(*Controller)
}

// getRequest method gets request from pool
func (e *engine) getRequest() *ahttp.Request {
	return e.rPool.Get().(*ahttp.Request)
}

// putController method puts controller back to pool
func (e *engine) putController(c *Controller) {
	c.Reset()
	e.cPool.Put(c)
}

// putRequest method puts request back to pool
func (e *engine) putRequest(r *ahttp.Request) {
	r.Reset()
	e.rPool.Put(r)
}

// getBuffer method gets buffer from pool
func (e *engine) getBuffer() *bytes.Buffer {
	return e.bPool.Get().(*bytes.Buffer)
}

// putBPool puts buffer into pool
func (e *engine) putBuffer(b *bytes.Buffer) {
	b.Reset()
	e.bPool.Put(b)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// serveStatic method static file/directory delivery.
func serveStatic(c *Controller, route *router.Route, pathParams *router.PathParams) error {

	fmt.Println("Static route:", route, pathParams)

	// TODO static serve implementation

	return errFileNotFound
}

// handleNotFound method is used for 1. route action not found, 2. route is
// not found and 3. static file/directory.
func handleNotFound(c *Controller, domain *router.Domain, isStatic bool) {
	if domain.NotFoundRoute == nil {
		c.Reply().NotFound().Text("404 Not Found")
		return
	}

	if err := c.setTarget(domain.NotFoundRoute); err != errTargetNotFound {
		target := reflect.ValueOf(c.target)
		if notFoundAction := target.MethodByName(c.action.Name); notFoundAction.IsValid() {
			log.Debugf("Calling not-found on controller: %s.%s", c.controller, c.action.Name)
			notFoundAction.Call([]reflect.Value{reflect.ValueOf(isStatic)})
		}
	} // may be later on else part
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

func newEngine() *engine {
	// TODO provide config for pool size
	return &engine{
		cPool: pool.NewPool(150, func() interface{} {
			return &Controller{}
		}),
		rPool: pool.NewPool(150, func() interface{} {
			return &ahttp.Request{}
		}),
		bPool: pool.NewPool(50, func() interface{} {
			return &bytes.Buffer{}
		}),
	}
}
