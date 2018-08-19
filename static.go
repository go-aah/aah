// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"aahframework.org/ahttp"
	"aahframework.org/essentials"
	"aahframework.org/vfs"
)

var (
	errSeeker = errors.New("static: seeker can't seek")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initStatic() error {
	a.staticMgr = &staticManager{
		a:                     a,
		mimeCacheHdrMap:       make(map[string]string),
		noCacheHdrValue:       "no-cache, no-store, must-revalidate",
		dirListDateTimeFormat: "2006-01-02 15:04:05",
	}

	// default cache header
	a.staticMgr.defaultCacheHdr = a.Config().StringDefault("cache.static.default_cache_control", "max-age=31536000, public")

	// MIME cache headers
	// static file cache configuration is from `cache.static.*`
	keyPrefix := "cache.static.mime_types"
	for _, k := range a.Config().KeysByPath(keyPrefix) {
		mimes := strings.Split(a.Config().StringDefault(keyPrefix+"."+k+".mime", ""), ",")
		for _, m := range mimes {
			if !ess.IsStrEmpty(m) {
				hdr := a.Config().StringDefault(keyPrefix+"."+k+".cache_control", a.staticMgr.defaultCacheHdr)
				a.staticMgr.mimeCacheHdrMap[strings.TrimSpace(strings.ToLower(m))] = hdr
			}
		}
	}

	return nil
}

type staticManager struct {
	a                     *app
	defaultCacheHdr       string
	noCacheHdrValue       string
	dirListDateTimeFormat string
	mimeCacheHdrMap       map[string]string
}

func (s *staticManager) Serve(ctx *Context) error {
	// TODO static assets Dynamic minify for JS and CSS for non-dev profile

	// Determine route is file or directory as per user defined
	// static route config (refer to https://docs.aahframework.org/static-files.html#section-static).
	f, err := s.open(ctx)
	if err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		}
		s.writeError(ctx.Res, ctx.Req, err)
		return nil
	}
	defer ess.CloseQuietly(f)

	fi, err := f.Stat()
	if err != nil {
		s.writeError(ctx.Res, ctx.Req, err)
		return nil
	}

	gf, ok := f.(vfs.Gziper)
	var fr io.ReadSeeker = f
	if s.a.gzipEnabled && ctx.Req.IsGzipAccepted {
		if ok && gf.IsGzip() {
			ctx.Res.Header().Add(ahttp.HeaderVary, ahttp.HeaderAcceptEncoding)
			ctx.Res.Header().Add(ahttp.HeaderContentEncoding, gzipContentEncoding)
			fr = bytes.NewReader(gf.RawBytes())
		} else if fi.Size() > defaultGzipMinSize && gzipRequired(fi.Name()) {
			ctx.Res = wrapGzipWriter(ctx.Res)
		}
	}

	// write headers
	ctx.writeHeaders()

	// Serve file
	if fi.Mode().IsRegular() {
		// `Cache-Control` header based on `cache.static.*`
		if contentType, err := detectFileContentType(fi.Name(), f); err == nil {
			ctx.Res.Header().Set(ahttp.HeaderContentType, contentType)

			// apply cache header if environment profile is `prod`
			if s.a.IsProfileProd() {
				ctx.Res.Header().Set(ahttp.HeaderCacheControl, s.cacheHeader(contentType))
			} else { // for static files hot-reload
				ctx.Res.Header().Set(ahttp.HeaderExpires, "0")
				ctx.Res.Header().Set(ahttp.HeaderCacheControl, s.noCacheHdrValue)
			}
		}

		// 'OnPreReply' server extension point
		s.a.he.publishOnPreReplyEvent(ctx)

		// 'OnHeaderReply' HTTP event
		s.a.he.publishOnHeaderReplyEvent(ctx.Res.Header())

		http.ServeContent(ctx.Res, ctx.Req.Unwrap(), path.Base(fi.Name()), fi.ModTime(), fr)

		// 'OnAfterReply' server extension point
		s.a.he.publishOnPostReplyEvent(ctx)

		// Send data to access log channel
		if s.a.accessLogEnabled && s.a.staticAccessLogEnabled {
			s.a.accessLog.Log(ctx)
		}
		return nil
	}

	// Serve directory
	if fi.Mode().IsDir() && ctx.route.ListDir {
		// redirect if the directory name doesn't end in a slash
		if ctx.Req.Path[len(ctx.Req.Path)-1] != '/' {
			ctx.Log().Debugf("redirecting to dir: %s", ctx.Req.Path+"/")
			http.Redirect(ctx.Res, ctx.Req.Unwrap(), path.Base(ctx.Req.Path)+"/", http.StatusMovedPermanently)
			return nil
		}

		// 'OnPreReply' server extension point
		s.a.he.publishOnPreReplyEvent(ctx)

		s.listDirectory(ctx.Res, ctx.Req.Unwrap(), f)

		// 'OnAfterReply' server extension point
		s.a.he.publishOnPostReplyEvent(ctx)

		// Send data to access log channel
		if s.a.accessLogEnabled && s.a.staticAccessLogEnabled {
			s.a.accessLog.Log(ctx)
		}
		return nil
	}

	// Flow reached here it means directory listing is not allowed
	ctx.Log().Warnf("Directory listing not allowed: %s", ctx.Req.Path)
	ctx.Res.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(ctx.Res, "403 Directory listing not allowed")

	return nil
}

func (s *staticManager) open(ctx *Context) (vfs.File, error) {
	var filePath string
	if ctx.route.IsFile() { // this is configured value from routes.conf
		filePath = parseCacheBustPart(ctx.route.File, s.a.BuildInfo().Version)
	} else {
		filePath = parseCacheBustPart(ctx.Req.PathValue("filepath"), s.a.BuildInfo().Version)
	}

	resource := filepath.ToSlash(path.Join(s.a.VirtualBaseDir(), ctx.route.Dir, filePath))
	ctx.Log().Tracef("Static resource: %s", resource)

	return s.a.VFS().Open(resource)
}

func (s *staticManager) cacheHeader(contentType string) string {
	if hdrValue, found := s.mimeCacheHdrMap[stripCharset(contentType)]; found {
		return hdrValue
	}
	return s.defaultCacheHdr
}

// listDirectory method compose directory listing response
func (s *staticManager) listDirectory(res http.ResponseWriter, req *http.Request, f http.File) {
	dirs, err := f.Readdir(-1)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Error reading directory")
		return
	}
	sort.Sort(byName(dirs))

	res.Header().Set(ahttp.HeaderContentType, ahttp.ContentTypeHTML.String())
	fmt.Fprintf(res, "<html>\n")
	fmt.Fprintf(res, "<head><title>Listing of %s</title></head>\n", req.URL.Path)
	fmt.Fprintf(res, "<body bgcolor=\"white\">\n")
	fmt.Fprintf(res, "<h1>Listing of %s</h1><hr>\n", req.URL.Path)
	fmt.Fprintf(res, "<pre><table border=\"0\">\n")
	fmt.Fprintf(res, "<tr><td collapse=\"2\"><a href=\"../\">../</a></td></tr>\n")
	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		// name may contain '?' or '#', which must be escaped to remain
		// part of the URL path, and not indicate the start of a query
		// string or fragment.
		url := url.URL{Path: name}
		fmt.Fprintf(res, "<tr><td><a href=\"%s\">%s</a></td><td width=\"200px\" align=\"right\">%s</td></tr>\n",
			url.String(),
			template.HTMLEscapeString(name),
			d.ModTime().Format(s.dirListDateTimeFormat),
		)
	}
	fmt.Fprintf(res, "</table></pre>\n")
	fmt.Fprintf(res, "<hr></body>\n")
	fmt.Fprintf(res, "</html>\n")
}

func (s *staticManager) writeError(res ahttp.ResponseWriter, req *ahttp.Request, err error) {
	if os.IsPermission(err) {
		s.a.Log().Warnf("Static file permission issue: %s", req.Path)
		res.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(res, "403 Forbidden")
	} else {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "500 Internal Server Error")
	}
}

// Sort interface for Directory list
type byName []os.FileInfo

func (s byName) Len() int           { return len(s) }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
