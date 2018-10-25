// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// NOTE: pprof idea and most of code snippet borrowed/referred from
// https://github.com/golang/go/blob/master/src/net/http/pprof/pprof.go
// and customized for aah framework.

package diagnosis

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
)

var profileDescriptions = map[string]string{
	"allocs":       "A sampling of all past memory allocations",
	"block":        "Stack traces that led to blocking on synchronization primitives",
	"cmdline":      "The command line invocation of the current program",
	"goroutine":    "Stack traces of all current goroutines",
	"heap":         "A sampling of memory allocations of live objects. You can specify the gc GET parameter to run GC before taking the heap sample.",
	"mutex":        "Stack traces of holders of contended mutexes",
	"profile":      "CPU profile. You can specify the duration in the seconds GET parameter. After you get the profile file, use the go tool pprof command to investigate the profile.",
	"threadcreate": "Stack traces that led to the creation of new OS threads",
	"trace":        "A trace of execution of the current program. You can specify the duration in the seconds GET parameter. After you get the trace file, use the go tool trace command to investigate.",
	"symbol":       "Symbol looks up the program counters listed in the request, responding with a table mapping program counters to function names.",
}

type profile struct {
	Name  string
	Href  string
	Desc  string
	Count int
}

// IndexHandler responds with the pprof-formatted profile named by the request.
// For example, "/diagnosis/pprof/heap" serves the "heap" profile.
func (d *Diagnosis) indexHandler(w http.ResponseWriter, r *http.Request) {
	var profiles []profile
	for _, p := range pprof.Profiles() {
		profiles = append(profiles, profile{
			Name:  p.Name(),
			Href:  p.Name() + "?debug=1",
			Desc:  profileDescriptions[p.Name()],
			Count: p.Count(),
		})
	}

	// Adding other profiles exposed from within this package
	for _, p := range []string{"cmdline", "profile", "trace", "symbol"} {
		profiles = append(profiles, profile{
			Name: p,
			Href: p,
			Desc: profileDescriptions[p],
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	if err := indexTmpl.Execute(w, map[string]interface{}{
		"AppName":    d.appName,
		"PathPrefix": d.pathPrefix,
		"Profiles":   profiles,
	}); err != nil {
		log.Print(err)
	}
}

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8" />
<meta http-equiv="X-UA-Compatible" content="IE=edge" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>{{ .AppName }} - Diagnosis</title>
<style>
html {-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}
html, body {
	margin: 0;
	background-color: #fff;
	color: #636b6f;
	font-family: Georgia, serif;
	font-weight: 100;
	height: 80vh;
	padding: 25px;
}
.profile-list {
	width: 100%;
	border: 0;
}
.profile-list thead {
	line-height: 26px;
	background-color: #efefef;
	font-weight: bold;
}
.profile-list thead td {
	padding-left: 15px;
	padding-right: 15px;
}
.profile-list td {
	text-align: left;
}
.profile-list tbody td {	
	padding-top: 5px;
	padding-bottom: 5px;
}
.profile-list td.count {
	text-align: center;
}
</style>
</head>
<body>
<center><h2>Diagnosis: {{ .AppName }}</h2></center><br>
<br>
<center>
<table class="profile-list">
<thead>
	<td>Count</td>
	<td>Profile</td>
	<td>Description</td>
</thead>
<tbody>
{{ range .Profiles }}
	<tr>
		<td class="count">{{ .Count }}</td>
		<td><a href={{ $.PathPrefix }}/pprof/{{.Href}}>{{ .Name }}</a></td>
		<td>{{ .Desc }}</td>
	</tr>
{{ end }}
</tbody>
</table>
</center>
</body>
</html>
`))

// DynamicProfileHandler serves the profile info for "allocs, block, goroutine, heap and mutex".
func (d *Diagnosis) dynamicProfileHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, d.pathPrefix+"/pprof/") {
		name := strings.TrimPrefix(r.URL.Path, d.pathPrefix+"/pprof/")
		if name != "" {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			p := pprof.Lookup(string(name))
			if p == nil {
				serveError(w, http.StatusNotFound, "Unknown profile")
				return
			}
			gc, _ := strconv.Atoi(r.FormValue("gc"))
			debug, _ := strconv.Atoi(r.FormValue("debug"))
			if debug != 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
			}
			_ = d.doProfileByName(w, name, gc > 0, debug)
			return
		}
	}
	w.Write([]byte("Unknown profile"))
}

// CmdlineHandler responds with the running program's
// command line, with arguments separated by NUL bytes.
func (d *Diagnosis) cmdlineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, strings.Join(os.Args, "\x00"))
}

// ProfileHandler responds with the pprof-formatted cpu profile.
// Profiling lasts for duration specified in seconds GET parameter, or for 30 seconds if not specified.
func (d *Diagnosis) profileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	sec, err := strconv.ParseInt(r.FormValue("seconds"), 10, 64)
	if sec <= 0 || err != nil {
		sec = 30
	}
	if durationExceedsWriteTimeout(r, float64(sec)) {
		serveError(w, http.StatusBadRequest, fmt.Sprintf("cpu profile duration exceeds diagnosis server's WriteTimeout: %v", d.serverWriteTimeout))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="profile"`)
	if err := d.cpuProfile(w, time.Duration(sec)*time.Second); err != nil {
		serveError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// TraceHandler responds with the execution trace in binary form.
// Tracing lasts for duration specified in seconds GET parameter, or for 1 second if not specified.
func (d *Diagnosis) traceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	sec, err := strconv.ParseFloat(r.FormValue("seconds"), 64)
	if sec <= 0 || err != nil {
		sec = 1
	}
	if durationExceedsWriteTimeout(r, sec) {
		serveError(w, http.StatusBadRequest, fmt.Sprintf("trace profile duration exceeds diagnosis server's WriteTimeout: %v", d.serverWriteTimeout))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="trace"`)
	if err := d.trace(w, time.Duration(sec*float64(time.Second))); err != nil {
		serveError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// SymbolHandler looks up the program counters listed in the request,
// responding with a table mapping program counters to function names.
func (d *Diagnosis) symbolHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// We have to read the whole POST body before
	// writing any output. Buffer the output here.
	var buf bytes.Buffer

	// We don't know how many symbols we have, but we
	// do have symbol information. Pprof only cares whether
	// this number is 0 (no symbols available) or > 0.
	fmt.Fprintf(&buf, "num_symbols: 1\n")

	var b *bufio.Reader
	if r.Method == "POST" {
		b = bufio.NewReader(r.Body)
	} else {
		b = bufio.NewReader(strings.NewReader(r.URL.RawQuery))
	}

	for {
		word, err := b.ReadSlice('+')
		if err == nil {
			word = word[0 : len(word)-1] // trim +
		}
		pc, _ := strconv.ParseUint(string(word), 0, 64)
		if pc != 0 {
			f := runtime.FuncForPC(uintptr(pc))
			if f != nil {
				fmt.Fprintf(&buf, "%#x %s\n", pc, f.Name())
			}
		}

		// Wait until here to check for err; the last
		// symbol will have an err because it doesn't end in +.
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(&buf, "reading request: %v\n", err)
			}
			break
		}
	}

	w.Write(buf.Bytes())
}

func durationExceedsWriteTimeout(r *http.Request, seconds float64) bool {
	srv, ok := r.Context().Value(http.ServerContextKey).(*http.Server)
	return ok && srv.WriteTimeout != 0 && seconds >= srv.WriteTimeout.Seconds()
}

func serveError(w http.ResponseWriter, status int, txt string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Go-Pprof", "1")
	w.Header().Set("X-Aah-Diagnosis-Pprof", "1")
	w.Header().Del("Content-Disposition")
	w.WriteHeader(status)
	fmt.Fprintln(w, txt)
}
