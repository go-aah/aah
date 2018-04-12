// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	keyAahRequestDump      = "_aahRequestDump"
	keyAahRequestDumpBody  = "_aahRequestDumpBody"
	keyAahResponseDumpBody = "_aahResponseDumpBody"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app Unexported methods
//______________________________________________________________________________

func (a *app) initDumpLog() error {
	// log file configuration
	cfg, _ := config.ParseString("")
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
		a:                a,
		e:                a.engine,
		logger:           adLog,
		dumpRequestBody:  a.Config().BoolDefault("server.dump_log.request_body", false),
		dumpResponseBody: a.Config().BoolDefault("server.dump_log.response_body", false),
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// dumpLogger
//______________________________________________________________________________

type dumpLogger struct {
	a                *app
	e                *engine
	logger           *log.Logger
	dumpRequestBody  bool
	dumpResponseBody bool
}

func (d *dumpLogger) composeRequestDump(ctx *Context) string {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	uri := fmt.Sprintf("%s://%s%s", ctx.Req.Scheme, ctx.Req.Host, ctx.Req.Path)
	queryStr := ctx.Req.Params.Query.Encode()
	if !ess.IsStrEmpty(queryStr) {
		uri += "?" + queryStr
	}

	buf.WriteString(fmt.Sprintf("\nURI: %s\n", uri))
	buf.WriteString(fmt.Sprintf("METHOD: %s\n", ctx.Req.Method))
	buf.WriteString(fmt.Sprintf("PROTO: %s\n", ctx.Req.Proto))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(d.composeHeaders(ctx.Req.Header))

	if ctx.a.dumpLog.dumpRequestBody {
		if len(ctx.Req.Params.Form) > 0 {
			ctx.Set(keyAahRequestDumpBody, ctx.Req.Params.Form.Encode())
		} else if ahttp.ContentTypePlainText.IsEqual(ctx.Req.ContentType().Mime) ||
			ahttp.ContentTypeHTML.IsEqual(ctx.Req.ContentType().Mime) {
			if b, err := ioutil.ReadAll(ctx.Req.Body()); err == nil {
				ctx.Set(keyAahRequestDumpBody, string(b))
				ctx.Req.Unwrap().Body = ioutil.NopCloser(bytes.NewReader(b))
			}
		}
	}

	return buf.String()
}

func (d *dumpLogger) composeResponseDump(ctx *Context) string {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	buf.WriteString(fmt.Sprintf("STATUS: %d %s\n", ctx.Res.Status(), http.StatusText(ctx.Res.Status())))
	buf.WriteString(fmt.Sprintf("BYTES WRITTEN: %d\n", ctx.Res.BytesWritten()))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(d.composeHeaders(ctx.Res.Header()))

	return buf.String()
}

func (d *dumpLogger) composeHeaders(hdrs http.Header) string {
	var str []string
	for _, k := range sortHeaderKeys(hdrs) {
		str = append(str, fmt.Sprintf("    %s: %s", k, strings.Join(hdrs[k], ", ")))
	}
	return strings.Join(str, "\n")
}

func (d *dumpLogger) addReqBodyIntoCtx(ctx *Context, result reflect.Value) {
	switch ctx.Req.ContentType().Mime {
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		if b, err := json.MarshalIndent(result.Interface(), "", "    "); err == nil {
			ctx.Set(keyAahRequestDumpBody, string(b))
		}
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		if b, err := xml.MarshalIndent(result.Interface(), "", "    "); err == nil {
			ctx.Set(keyAahRequestDumpBody, string(b))
		}
	}
}

func (d *dumpLogger) addResBodyIntoCtx(ctx *Context) {
	ct := ctx.Reply().ContType
	if ahttp.ContentTypeHTML.IsEqual(ct) ||
		ahttp.ContentTypeJSON.IsEqual(ct) || ahttp.ContentTypeJSONText.IsEqual(ct) ||
		ahttp.ContentTypeXML.IsEqual(ct) || ahttp.ContentTypeXMLText.IsEqual(ct) ||
		ahttp.ContentTypePlainText.IsEqual(ct) {
		ctx.Set(keyAahResponseDumpBody, ctx.Reply().Body().String())
	}
}

func (e *engine) dump(ctx *Context) {
	dumpStr := fmt.Sprint(ctx.Get(keyAahRequestDump)) + "\n"
	if e.a.dumpLog.dumpRequestBody && ctx.Get(keyAahRequestDumpBody) != nil {
		dumpStr += "BODY:\n" + fmt.Sprint(ctx.Get(keyAahRequestDumpBody)) + "\n"
	}

	dumpStr += "\n-----------------------------------------------------------------------\n\n"

	dumpStr += ctx.a.dumpLog.composeResponseDump(ctx) + "\n"
	if e.a.dumpLog.dumpResponseBody && ctx.Get(keyAahResponseDumpBody) != nil {
		dumpStr += "BODY:\n" + fmt.Sprint(ctx.Get(keyAahResponseDumpBody)) + "\n"
	}
	dumpStr += "\n=======================================================================\n"

	e.a.dumpLog.logger.Print(dumpStr)
}
