// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"

	"aahframework.org/ahttp.v0"
	"aahframework.org/aruntime.v0"
	"aahframework.org/atemplate.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/pool.v0"
	"aahframework.org/router.v0"
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

	byName []os.FileInfo
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Engine methods
//___________________________________

// ServeHTTP method implementation of http.Handler interface.
func (e *engine) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	w := ahttp.WrapResponseWriter(rw)

	// Recovery handling, capture every possible panic(s) from application.
	defer e.handleRecovery(w, req)

	domain := AppRouter().FindDomain(req)
	if domain == nil {
		e.writeReply(w, req, NewReply().NotFound().Text("404 Route Not Exists"))
		return
	}

	route, pathParams, rts := domain.Lookup(req)
	if route == nil { // route not found
		if reply := handleRtsOptionsMna(domain, req, rts); reply != nil {
			e.writeReply(w, req, reply)
			return
		}

		handleRouteNotFound(w, req, domain, domain.NotFoundRoute)
		return
	}

	// Serving static file
	if route.IsStatic {
		if err := serveStatic(w, req, route, pathParams); err == errFileNotFound {
			handleRouteNotFound(w, req, domain, route)
		}
		return
	}

	// preparing targeted context
	ctx := e.prepareContext(w, req, route)
	if ctx == nil { // no controller or action found for the route
		handleRouteNotFound(w, req, domain, route)
		return
	}

	defer e.putContext(ctx)

	// Path parameters
	if pathParams.Len() > 0 {
		ctx.Req.Params.Path = make(map[string]string, pathParams.Len())
		for _, v := range *pathParams {
			ctx.Req.Params.Path[v.Key] = v.Value
		}
	}

	ctx.domain = domain
	ctx.viewArgs = make(map[string]interface{})

	// set defaults when actual value not found
	e.setDefaults(ctx)

	// Middlewares
	e.executeMiddlewares(ctx)

	// Write Reply on the wire
	e.writeReply(w, req, ctx.reply)
}

// handleRecovery method handles application panics and recovers from it.
// Panic gets translated into HTTP Internal Server Error (Status 500).
func (e *engine) handleRecovery(w http.ResponseWriter, req *http.Request) {
	if r := recover(); r != nil {
		log.Errorf("Internal Server Error on %s", req.URL.Path)

		st := aruntime.NewStacktrace(r, AppConfig())
		buf := e.getBuffer()
		defer e.putBuffer(buf)

		st.Print(buf)
		log.Error(buf.String())

		if AppProfile() != "prod" { // detailed error info
			// TODO design server error page with stack trace info
			e.writeReply(w, req, NewReply().InternalServerError().Text("Internal Server Error: %s", buf.String()))
			return
		}

		e.writeReply(w, req, NewReply().InternalServerError().Text("Internal Server Error"))
	}
}

// prepareContext method gets controller, request from pool, set the targeted
// controller, parses the request and returns the controller.
func (e *engine) prepareContext(w ahttp.ResponseWriter, req *http.Request, route *router.Route) *Context {
	ctx := e.getContext()
	if err := ctx.setTarget(route); err == errTargetNotFound {
		e.putContext(ctx)
		return nil
	}

	r := e.getRequest()
	ctx.Req = ahttp.ParseRequest(req, r)
	ctx.Res = w
	ctx.reply = NewReply()

	return ctx
}

// setDefaults method sets default value based on aah app configuration
// when actual value is not found.
func (e *engine) setDefaults(ctx *Context) {
	if ctx.Req.Locale == nil {
		ctx.Req.Locale = ahttp.NewLocale(appConfig.StringDefault("i18n.default", "en"))
	}
}

// executeMiddlewares method executes the configured middlewares.
func (e *engine) executeMiddlewares(ctx *Context) {
	mwChain[0].Next(ctx)
}

// writeReply method writes the response on the wire based on `Reply` instance.
func (e *engine) writeReply(w http.ResponseWriter, req *http.Request, reply *Reply) {
	// handle redirects
	if reply.redirect {
		log.Debugf("Redirecting to '%s' with status '%d'", reply.redirectURL, reply.Code)
		http.Redirect(w, req, reply.redirectURL, reply.Code)
		return
	}

	// Response already written on the wire, don't go forward.
	if reply.done {
		return
	}

	buf := e.getBuffer()
	defer e.putBuffer(buf)

	// Render and detect the errors earlier, framework can write error info
	// without messing with response.
	// HTTP Body
	if reply.Rdr != nil {
		if err := reply.Rdr.Render(buf); err != nil {
			log.Error("Render error: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Render error: " + err.Error() + "\n"))
			return
		}
	}

	// HTTP headers
	for k, v := range reply.Hdr {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	// ContentType
	w.Header().Set(ahttp.HeaderContentType, reply.ContType)

	// HTTP status
	if reply.IsStatusSet() {
		w.WriteHeader(reply.Code)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Write it on the wire
	_, _ = buf.WriteTo(w)
}

// defaultContentType method returns the Content-Type based on 'render.default'
// config from aah.conf
func defaultContentType() *ahttp.ContentType {
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

// getContext method gets context from pool
func (e *engine) getContext() *Context {
	return e.cPool.Get().(*Context)
}

// getRequest method gets request from pool
func (e *engine) getRequest() *ahttp.Request {
	return e.rPool.Get().(*ahttp.Request)
}

// putContext method puts context back to pool
func (e *engine) putContext(ctx *Context) {
	// Try to close if `io.Closer` interface satisfies.
	if ctx.Res != nil {
		ctx.Res.(*ahttp.Response).Close()
	}

	// clear and put `ahttp.Request` into pool
	if ctx.Req != nil {
		ctx.Req.Reset()
		e.rPool.Put(ctx.Req)
	}

	// clear and put `aah.Context` into pool
	if ctx != nil {
		ctx.Reset()
		e.cPool.Put(ctx)
	}
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
func serveStatic(w http.ResponseWriter, req *http.Request, route *router.Route, pathParams *router.PathParams) error {
	var fileabs string
	if route.IsDir() {
		fileabs = filepath.Join(AppBaseDir(), route.Dir, filepath.FromSlash(pathParams.Get("filepath")))
	} else {
		fileabs = filepath.Join(AppBaseDir(), "static", filepath.FromSlash(route.File))
	}

	dir, file := filepath.Split(fileabs)
	log.Tracef("Dir: %s, File: %s", dir, file)

	fs := ahttp.Dir(dir, route.ListDir)
	f, err := fs.Open(file)
	if err != nil {
		if err == ahttp.ErrDirListNotAllowed {
			log.Warnf("directory listing not allowed: %s", req.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("403 Directory listing not allowed"))
			return nil
		} else if os.IsNotExist(err) {
			log.Errorf("file not found: %s", req.URL.Path)
			return errFileNotFound
		} else if os.IsPermission(err) {
			log.Warnf("permission issue: %s", req.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("403 Forbidden"))
			return nil
		}

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("500 Internal Server Error"))
		return nil
	}
	defer ess.CloseQuietly(f)

	fi, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("500 Internal Server Error"))
		return nil
	}

	if fi.IsDir() {
		// redirect if the directory name doesn't end in a slash
		if req.URL.Path[len(req.URL.Path)-1] != '/' {
			log.Debugf("redirecting to dir: %s", req.URL.Path+"/")
			http.Redirect(w, req, path.Base(req.URL.Path)+"/", http.StatusFound)
			return nil
		}

		directoryList(w, req, f)
		return nil
	}

	http.ServeContent(w, req, file, fi.ModTime(), f)
	return nil
}

func directoryList(w http.ResponseWriter, req *http.Request, f http.File) {
	dirs, err := f.Readdir(-1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Error reading directory"))
		return
	}
	sort.Sort(byName(dirs))

	w.Header().Set(ahttp.HeaderContentType, ahttp.ContentTypeHTML.Raw())
	reqPath := req.URL.Path
	fmt.Fprintf(w, "<html>\n")
	fmt.Fprintf(w, "<head><title>Listing of %s</title></head>\n", reqPath)
	fmt.Fprintf(w, "<body bgcolor=\"white\">\n")
	fmt.Fprintf(w, "<h1>Listing of %s</h1><hr>\n", reqPath)
	fmt.Fprintf(w, "<pre><table border=\"0\">\n")
	fmt.Fprintf(w, "<tr><td collapse=\"2\"><a href=\"../\">../</a></td></tr>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		url := url.URL{Path: name}
		fmt.Fprintf(w, "<tr><td><a href=\"%s\">%s</a></td><td width=\"200px\" align=\"right\">%s</td></tr>\n",
			url.String(),
			atemplate.HTMLEscape(name),
			d.ModTime().Format(appDefaultDateTimeFormat),
		)
	}
	fmt.Fprintf(w, "</table></pre>\n")
	fmt.Fprintf(w, "<hr></body>\n")
	fmt.Fprintf(w, "</html>\n")
}

func newEngine() *engine {
	// TODO provide config for pool size
	return &engine{
		cPool: pool.NewPool(150, func() interface{} {
			return &Context{}
		}),
		rPool: pool.NewPool(150, func() interface{} {
			return &ahttp.Request{}
		}),
		bPool: pool.NewPool(60, func() interface{} {
			return &bytes.Buffer{}
		}),
	}
}

// Sort interface

func (s byName) Len() int           { return len(s) }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
