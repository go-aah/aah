// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	keyAahRequestBodyBuf  = "_aahRequestBodyBuf"
	keyAahResponseBodyBuf = "_aahResponseBodyBuf"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initDumpLog() error {
	// log file configuration
	cfg := config.NewEmpty()
	file := a.Config().StringDefault("server.dump_log.file", "")

	cfg.SetString("log.receiver", "file")
	if ess.IsStrEmpty(file) {
		cfg.SetString("log.file", filepath.Join(a.logsDir(), a.binaryFilename()+"-dump.log"))
	} else {
		abspath, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		cfg.SetString("log.file", abspath)
	}

	cfg.SetString("log.pattern", "%message")

	adLog, err := log.New(cfg)
	if err != nil {
		return err
	}

	a.dumpLog = &dumpLogger{
		a:               a,
		logger:          adLog,
		logRequestBody:  a.Config().BoolDefault("server.dump_log.request_body", false),
		logResponseBody: a.Config().BoolDefault("server.dump_log.response_body", false),
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// dumpLogger
//______________________________________________________________________________

type dumpLogger struct {
	a               *app
	logger          *log.Logger
	logRequestBody  bool
	logResponseBody bool
}

func (d *dumpLogger) Dump(ctx *Context) {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	// Request
	uri := fmt.Sprintf("%s://%s%s", ctx.Req.Scheme, ctx.Req.Host, ctx.Req.Path)
	if qs := ctx.Req.URL().RawQuery; len(qs) > 0 {
		uri += "?" + qs
	}

	buf.WriteString(fmt.Sprintf("\nURI: %s\n", uri))
	buf.WriteString(fmt.Sprintf("METHOD: %s\n", ctx.Req.Method))
	buf.WriteString(fmt.Sprintf("PROTO: %s\n", ctx.Req.Proto))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(d.composeHeaders(ctx.Req.Header) + "\n")
	if d.logRequestBody {
		buf.WriteString("BODY:\n")
		d.writeBody(keyAahRequestBodyBuf, ctx.Req.ContentType().Mime, buf, ctx)
	}

	buf.WriteString("\n\n-----------------------------------------------------------------------\n\n")

	// Response
	buf.WriteString(fmt.Sprintf("STATUS: %d %s\n", ctx.Res.Status(), http.StatusText(ctx.Res.Status())))
	buf.WriteString(fmt.Sprintf("BYTES WRITTEN: %d\n", ctx.Res.BytesWritten()))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(d.composeHeaders(ctx.Res.Header()) + "\n")
	if d.logResponseBody {
		buf.WriteString("BODY:\n")
		d.writeBody(keyAahResponseBodyBuf, ctx.Reply().ContType, buf, ctx)
	}

	buf.WriteString("\n\n=======================================================================")

	d.logger.Print(buf.String())
}

func (d *dumpLogger) writeBody(key, ct string, w *bytes.Buffer, ctx *Context) {
	cbuf := ctx.Get(key)
	if cbuf == nil {
		w.WriteString("    ***** NO CONTENT *****")
		return
	}

	b := cbuf.(*bytes.Buffer)
	switch stripCharset(ct) {
	case ahttp.ContentTypeHTML.Mime, ahttp.ContentTypeForm.Mime,
		ahttp.ContentTypeMultipartForm.Mime, ahttp.ContentTypePlainText.Mime:
		_, _ = b.WriteTo(w)
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		_ = json.Indent(w, b.Bytes(), "", "    ")
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		// TODO XML formatting
		_, _ = b.WriteTo(w)
	}
	releaseBuffer(b)
}

func (d *dumpLogger) composeHeaders(hdrs http.Header) string {
	var str []string
	for _, k := range sortHeaderKeys(hdrs) {
		str = append(str, fmt.Sprintf("    %s: %s", k, strings.Join(hdrs[k], ", ")))
	}
	return strings.Join(str, "\n")
}
