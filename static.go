// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	sniffLen              = 512
	noCacheHdrValue       = "no-cache, no-store, must-revalidate"
	dirListDateTimeFormat = "2006-01-02 15:04:05"
)

var (
	staticDfltCacheHdr    string
	staticMimeCacheHdrMap = make(map[string]string)
	errSeeker             = errors.New("static: seeker can't seek")
)

type byName []os.FileInfo

// serveStatic method static file/directory delivery.
func (e *engine) serveStatic(ctx *Context) error {
	// Taking control over for static file delivery
	ctx.Reply().Done()

	// TODO static assets Dynamic minify for JS and CSS for non-dev profile

	// Determine route is file or directory as per user defined
	// static route config (refer to https://docs.aahframework.org/static-files.html#section-static).
	//   httpDir -> value is from routes config
	//   filePath -> value is from request
	httpDir, filePath := getHTTPDirAndFilePath(ctx)
	ctx.Log().Tracef("Dir: %s, Filepath: %s", httpDir, filePath)

	f, err := httpDir.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errFileNotFound
		}
		writeStaticFileError(ctx.Res, ctx.Req, err)
		return nil
	}

	defer ess.CloseQuietly(f)
	fi, err := f.Stat()
	if err != nil {
		writeStaticFileError(ctx.Res, ctx.Req, err)
		return nil
	}

	// Gzip, 1kb above, TODO make it configurable from aah.conf
	if fi.Size() > 1024 {
		ctx.Reply().gzip = checkGzipRequired(filePath)
		e.wrapGzipWriter(ctx)
	}
	e.writeHeaders(ctx)

	// Serve file
	if fi.Mode().IsRegular() {
		// `Cache-Control` header based on `cache.static.*`
		if contentType, err := detectFileContentType(filePath, f); err == nil {
			ctx.Res.Header().Set(ahttp.HeaderContentType, contentType)

			// apply cache header if environment profile is `prod`
			if appIsProfileProd {
				ctx.Res.Header().Set(ahttp.HeaderCacheControl, cacheHeader(contentType))
			} else { // for static files hot-reload
				ctx.Res.Header().Set(ahttp.HeaderExpires, "0")
				ctx.Res.Header().Set(ahttp.HeaderCacheControl, noCacheHdrValue)
			}
		}

		// 'OnPreReply' server extension point
		publishOnPreReplyEvent(ctx)

		http.ServeContent(ctx.Res, ctx.Req.Unwrap(), path.Base(filePath), fi.ModTime(), f)

		// 'OnAfterReply' server extension point
		publishOnAfterReplyEvent(ctx)

		// Send data to access log channel
		if e.isAccessLogEnabled && e.isStaticAccessLogEnabled {
			sendToAccessLog(ctx)
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
		publishOnPreReplyEvent(ctx)

		directoryList(ctx.Res, ctx.Req.Unwrap(), f)

		// 'OnAfterReply' server extension point
		publishOnAfterReplyEvent(ctx)

		// Send data to access log channel
		if e.isAccessLogEnabled && e.isStaticAccessLogEnabled {
			sendToAccessLog(ctx)
		}
		return nil
	}

	// Flow reached here it means directory listing is not allowed
	ctx.Log().Warnf("Directory listing not allowed: %s", ctx.Req.Path)
	ctx.Res.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(ctx.Res, "403 Directory listing not allowed")

	return nil
}

// directoryList method compose directory listing response
func directoryList(res http.ResponseWriter, req *http.Request, f http.File) {
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
			d.ModTime().Format(dirListDateTimeFormat),
		)
	}
	fmt.Fprintf(res, "</table></pre>\n")
	fmt.Fprintf(res, "<hr></body>\n")
	fmt.Fprintf(res, "</html>\n")
}

// checkGzipRequired method return for static which requires gzip response.
func checkGzipRequired(file string) bool {
	switch filepath.Ext(file) {
	case ".css", ".js", ".html", ".htm", ".json", ".xml",
		".txt", ".csv", ".ttf", ".otf", ".eot":
		return true
	default:
		return false
	}
}

// getHTTPDirAndFilePath method returns the `http.Dir` and requested file path.
// Note: `ctx.route.*` values come from application routes configuration.
func getHTTPDirAndFilePath(ctx *Context) (http.Dir, string) {
	if ctx.route.IsFile() { // this is configured value from routes.conf
		return http.Dir(filepath.Join(AppBaseDir(), ctx.route.Dir)),
			parseCacheBustPart(ctx.route.File, AppBuildInfo().Version)
	}
	return http.Dir(filepath.Join(AppBaseDir(), ctx.route.Dir)),
		parseCacheBustPart(ctx.Req.PathValue("filepath"), AppBuildInfo().Version)
}

// detectFileContentType method to identify the static file content-type.
func detectFileContentType(file string, content io.ReadSeeker) (string, error) {
	ctype := mime.TypeByExtension(filepath.Ext(file))
	if ctype == "" {
		// read a chunk to decide between utf-8 text and binary
		var buf [sniffLen]byte
		n, _ := io.ReadFull(content, buf[:])
		ctype = http.DetectContentType(buf[:n])

		// rewind to output whole file
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return "", errSeeker
		}
	}
	return ctype, nil
}

func cacheHeader(contentType string) string {
	if idx := strings.IndexByte(contentType, ';'); idx > 0 {
		contentType = contentType[:idx]
	}

	if hdrValue, found := staticMimeCacheHdrMap[contentType]; found {
		return hdrValue
	}
	return staticDfltCacheHdr
}

// Sort interface for Directory list
func (s byName) Len() int           { return len(s) }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// parseStaticMimeCacheMap method parses the static file cache configuration
// `cache.static.*`.
func parseStaticMimeCacheMap(e *Event) {
	cfg := AppConfig()
	staticDfltCacheHdr = cfg.StringDefault("cache.static.default_cache_control", "max-age=31536000, public")
	keyPrefix := "cache.static.mime_types"

	for _, k := range cfg.KeysByPath(keyPrefix) {
		mimes := strings.Split(cfg.StringDefault(keyPrefix+"."+k+".mime", ""), ",")
		for _, m := range mimes {
			if ess.IsStrEmpty(m) {
				continue
			}
			hdr := cfg.StringDefault(keyPrefix+"."+k+".cache_control", staticDfltCacheHdr)
			staticMimeCacheHdrMap[strings.TrimSpace(strings.ToLower(m))] = hdr
		}
	}
}

func parseCacheBustPart(name, part string) string {
	if strings.Contains(name, part) {
		name = strings.Replace(name, "-"+part, "", 1)
		name = strings.Replace(name, part+"-", "", 1)
		return name
	}
	return name
}

func writeStaticFileError(res ahttp.ResponseWriter, req *ahttp.Request, err error) {
	if os.IsPermission(err) {
		log.Warnf("Static file permission issue: %s", req.Path)
		res.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(res, "403 Forbidden")
	} else {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "500 Internal Server Error")
	}
}

func init() {
	OnStart(parseStaticMimeCacheMap)
}
