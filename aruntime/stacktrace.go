// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/aruntime source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aruntime

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"aahframework.org/config"
	"aahframework.org/essentials"
	"aahframework.org/log"
)

const (
	createdByPrefix = "created by "
	panicPrefix     = "panic("
	basePathPrefix  = "..."
)

type (
	// Stacktrace holds the parse information of `debug.Stack()`. It's easier to
	// debug and understand.
	Stacktrace struct {
		Raw          string
		Recover      interface{}
		IsParsed     bool
		StripSrcBase bool
		GoRoutines   []*GoRoutine
	}

	// GoRoutine holds information of single Go routine stack trace.
	GoRoutine struct {
		Header     string
		MaxFuncLen int
		MaxPkgLen  int
		HasPanic   bool
		PanicIndex int
		Packages   []string
		Functions  []string
		LineNo     []string
	}
)

// NewStacktrace method collects debug stack information and parsing them into
// easy understanding and returns the instance.
func NewStacktrace(r interface{}, appCfg *config.Config) *Stacktrace {
	strace := &Stacktrace{
		Recover: r,
	}

	if appCfg.BoolDefault("runtime.debug.all_goroutines", false) {
		bufCfgSize := appCfg.StringDefault("runtime.debug.stack_buffer_size", "2mb")
		bufSize, err := ess.StrToBytes(bufCfgSize)
		if err != nil {
			log.Errorf("unable to parse 'runtime.debug.stack_buffer_size' value: %s, "+
				"fallback to default value", bufCfgSize)
			bufSize = 2 << 20 // default fallback size is 2mb
		}

		buf := make([]byte, bufSize)
		length := runtime.Stack(buf, true)
		if length < len(buf) {
			buf = buf[:length]
		}

		strace.Raw = string(buf)
	} else {
		strace.Raw = string(debug.Stack())
	}

	strace.StripSrcBase = appCfg.BoolDefault("runtime.debug.strip_src_base", false)

	return strace
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Stacktrace methods
//___________________________________

// Parse method parses the go debug stacktrace into easy to understand.
func (st *Stacktrace) Parse() {
	var sections [][]string
	var section []string

	scanner := bufio.NewScanner(strings.NewReader(st.Raw))
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			sections = append(sections, section)
			section = make([]string, 0)
			continue
		}
		section = append(section, line)
	}

	// Only one go routine section found
	if len(sections) == 0 && len(section) > 0 {
		sections = append(sections, section)
	}

	for _, s := range sections {
		gr := &GoRoutine{Header: s[0]}
		lnCnt := 1

		for _, ln := range s[1:] {
			ln = strings.Replace(strings.TrimSpace(ln), "%2e", ".", -1)
			if lnCnt%2 == 0 { // File Path
				// Strip hexa chars (+0x15c, etc)
				if idx := strings.IndexByte(ln, ' '); idx > 0 {
					ln = ln[:idx]
				}

				// Separate the path and line no
				if idx := strings.LastIndexByte(ln, ':'); idx > 0 {
					gr.LineNo = append(gr.LineNo, ln[idx+1:])
					ln = ln[:idx]
				}

				// Strip base path i.e. before `.../src/`
				if st.StripSrcBase {
					if idx := strings.Index(ln, "src"); idx > 0 {
						ln = basePathPrefix + ln[idx+3:]
					}
				}

				// Find max len
				if l := len(ln); l > gr.MaxPkgLen {
					gr.MaxPkgLen = l
				}

				gr.Packages = append(gr.Packages, ln)
			} else { // Function Info
				// Strip parameters hexa values
				if !strings.HasPrefix(ln, createdByPrefix) {
					if rparen := strings.LastIndex(ln, "("); rparen != -1 {
						if comma := strings.IndexByte(ln[rparen:], ','); comma == -1 {
							ln = ln[:rparen+1] + ")"
						} else {
							ln = ln[:rparen+1] + "...)"
						}
					}
				}

				ln = filepath.Base(ln)

				// Find func max len
				if l := len(ln); l > gr.MaxFuncLen {
					gr.MaxFuncLen = l
				}

				// Check this goroutine has `panic(...)`
				if yes := strings.HasPrefix(ln, panicPrefix); yes || !gr.HasPanic {
					gr.HasPanic = yes

					// Capture panic index
					if gr.HasPanic {
						gr.PanicIndex = len(gr.Functions)
					}
				}

				gr.Functions = append(gr.Functions, ln)
			}

			lnCnt++
		}

		st.GoRoutines = append(st.GoRoutines, gr)
	}

	st.IsParsed = true
}

// Print method prints the stack trace info to io.Writer.
func (st *Stacktrace) Print(w io.Writer) {
	if w == nil {
		return
	}

	if !st.IsParsed {
		st.Parse()
	}

	fmt.Fprintf(w, "STACKTRACE:\n%v\n", st.Recover)
	for _, gr := range st.GoRoutines {
		fmt.Fprint(w, "\n"+gr.Header+"\n")
		hdrStr := fmt.Sprintf("    %-"+strconv.Itoa(gr.MaxPkgLen+1)+"s   %-"+strconv.Itoa(gr.MaxFuncLen)+"s   %s\n",
			"FILE", "FUNCTION", "LINE NO")
		fmt.Fprint(w, hdrStr)
		fmt.Fprint(w, "    ")
		for idx := 1; idx < len(hdrStr)-4; idx++ {
			fmt.Fprint(w, "-")
		}
		fmt.Fprint(w, "\n")

		printFmt := "    %-" + strconv.Itoa(gr.MaxPkgLen+1) + "s   %-" + strconv.Itoa(gr.MaxFuncLen) + "s   #%s\n"
		for idx, f := range gr.Packages[gr.PanicIndex:] {
			idx += gr.PanicIndex
			fmt.Fprintf(w, printFmt, f, gr.Functions[idx], gr.LineNo[idx])
		}
	}
}
