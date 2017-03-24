// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"

	"aahframework.org/ahttp.v0"
	"aahframework.org/atemplate.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/router.v0"
)

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

// Sort interface for Directory list
func (s byName) Len() int           { return len(s) }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
