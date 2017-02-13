// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/render"
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
		c.Reply().NotFound().Text("404 Route Not Exists")
		return
	}

	route, pathParams, rts := domain.Lookup(c.Req)
	log.Tracef("Route: %#v, Path Params: %v, rts: %v ", route, pathParams, rts)

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

		// Path parameters
		if pathParams.Len() > 0 {
			c.Req.Params.Path = make(map[string]string, pathParams.Len())
			for _, v := range *pathParams {
				c.Req.Params.Path[v.Key] = v.Value
			}
		}

		c.domain = domain

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
				allowed += ", " + ahttp.MethodOptions
				log.Debugf("Auto 'OPTIONS' allowed HTTP Methods: %s", allowed)
				c.Reply().Header(ahttp.HeaderAllow, allowed)
				return
			}
		}
	}

	// 405 Method Not Allowed
	if domain.MethodNotAllowed {
		if allowed := domain.Allowed(c.Req.Method, c.Req.Path); !ess.IsStrEmpty(allowed) {
			allowed += ", " + ahttp.MethodOptions
			log.Debugf("Allowed HTTP Methods for 405 response: %s", allowed)
			c.Reply().
				Status(http.StatusMethodNotAllowed).
				Header(ahttp.HeaderAllow, allowed).
				Text("405 Method Not Allowed")
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
	req := c.Req.Raw

	if c.Req.Method != ahttp.MethodGet {
		contentType := c.Req.ContentType.Mime
		log.Debugf("request content type: %s", contentType)

		switch contentType {
		case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeXML.Mime:
			if payloadBytes, err := ioutil.ReadAll(req.Body); err == nil {
				c.Req.Payload = string(payloadBytes)
			} else {
				log.Errorf("unable to read request body for '%s': %s", contentType, err)
			}
		case ahttp.ContentTypeForm.Mime:
			if err := req.ParseForm(); err == nil {
				c.Req.Params.Form = req.Form
			} else {
				log.Errorf("unable to parse form: %s", err)
			}
		case ahttp.ContentTypeMultipartForm.Mime:
			if isMultipartEnabled {
				if err := req.ParseMultipartForm(appMultipartMaxMemory); err == nil {
					c.Req.Params.Form = req.MultipartForm.Value
					c.Req.Params.File = req.MultipartForm.File
				} else {
					log.Errorf("unable to parse multipart form: %s", err)
				}
			} else {
				log.Warn("multipart processing is disabled in aah.conf")
			}
		} // switch end

		// clean up
		defer func(r *http.Request) {
			if r.MultipartForm != nil {
				log.Debug("multipart form file clean up")
				if err := r.MultipartForm.RemoveAll(); err != nil {
					log.Error(err)
				}
			}
		}(req)
	}

	m.Next(c)

}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Template middleware
//___________________________________

// TemplateMiddleware finds appropriate template based request and route
// information.
func templateMiddleware(c *Controller, m *Middleware) {
	m.Next(c)

	reply := c.Reply()

	// ContentType
	if ess.IsStrEmpty(reply.ContType) {
		if !ess.IsStrEmpty(c.Req.AcceptContentType.Mime) &&
			c.Req.AcceptContentType.Mime != "*/*" { // based on 'Accept' Header
			reply.ContentType(c.Req.AcceptContentType.Raw())
		} else { // default Content-Type defined 'render.default' in aah.conf
			reply.ContentType(defaultContentType().Raw())
		}
	}

	// HTML response
	if AppMode() == appModeWeb && ahttp.ContentTypeHTML.IsEqual(reply.ContType) {
		if reply.Rdr == nil {
			reply.Rdr = &render.HTML{}
		}

		htmlRdr := reply.Rdr.(*render.HTML)

		if ess.IsStrEmpty(htmlRdr.Layout) {
			htmlRdr.Layout = appDefaultTmplLayout
		}

		if htmlRdr.ViewArgs == nil {
			htmlRdr.ViewArgs = make(map[string]interface{})
		}

		for k, v := range c.ViewArgs() {
			htmlRdr.ViewArgs[k] = v
		}

		// ViewArgs values from framework
		htmlRdr.ViewArgs["Host"] = c.Req.Host
		htmlRdr.ViewArgs["HTTPMethod"] = c.Req.Method
		htmlRdr.ViewArgs["Locale"] = c.Req.Locale
		htmlRdr.ViewArgs["ClientIP"] = c.Req.ClientIP
		htmlRdr.ViewArgs["RequestPath"] = c.Req.Path
		htmlRdr.ViewArgs["IsJSONP"] = c.Req.IsJSONP
		htmlRdr.ViewArgs["HTTPReferer"] = c.Req.Referer
		htmlRdr.ViewArgs["AahVersion"] = Version

		controllerName := c.controller
		if strings.HasSuffix(controllerName, controllerNameSuffix) {
			controllerName = controllerName[:len(controllerName)-controllerNameSuffixLen]
		}

		tmplPath := filepath.Join("pages", controllerName)
		tmplName := c.action.Name + appTemplateExt

		htmlRdr.Template = appTemplateEngine.Get(htmlRdr.Layout, tmplPath, tmplName)
		if htmlRdr.Template == nil {
			tmplFile := filepath.Join("views", "pages", controllerName, tmplName)
			if !appTemplateCaseSensitive {
				tmplFile = strings.ToLower(tmplFile)
			}

			log.Errorf("template not found: %s", tmplFile)
		}
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Interceptor middleware
//___________________________________

// interceptorMiddleware calls pre-defined actions (Before, After, Panic,
// Finally) from controller.
func interceptorMiddleware(c *Controller, m *Middleware) {
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

	m.Next(c)

	// After action
	if afterAction := target.MethodByName(incpAfterActionName); afterAction.IsValid() {
		log.Debugf("Calling after interceptor on controller: %s", c.controller)
		afterAction.Call(emptyArg)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Action middleware
//___________________________________

// ActionMiddleware calls the requested action on controller
func actionMiddleware(c *Controller, m *Middleware) {
	target := reflect.ValueOf(c.target)
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
