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
	"sort"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0-unstable"
)

const (
	keyAahRequestDump      = "_aahRequestDump"
	keyAahRequestDumpBody  = "_aahRequestDumpBody"
	keyAahResponseDumpBody = "_aahResponseDumpBody"
)

var (
	appDumpLog       *log.Logger
	dumpRequestBody  bool
	dumpResponseBody bool
	isDumpLogEnabled bool
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func initDumpLog(logsDir string, appCfg *config.Config) error {
	isDumpLogEnabled = appCfg.BoolDefault("server.dump_log.enable", false)
	if !isDumpLogEnabled {
		return nil
	}

	// log file configuration
	cfg, _ := config.ParseString("")
	file := appCfg.StringDefault("server.dump_log.file", "")

	cfg.SetString("log.receiver", "file")
	if ess.IsStrEmpty(file) {
		cfg.SetString("log.file", filepath.Join(logsDir, getBinaryFileName()+"-dump.log"))
	} else {
		abspath, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		cfg.SetString("log.file", abspath)
	}

	cfg.SetString("log.pattern", "%message")

	dumpRequestBody = appCfg.BoolDefault("server.dump_log.request_body", false)
	dumpResponseBody = appCfg.BoolDefault("server.dump_log.response_body", false)

	var err error
	appDumpLog, err = log.New(cfg)
	return err
}

func composeRequestDump(ctx *Context) string {
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
	buf.WriteString(composeHeaders(ctx.Req.Header))

	if dumpRequestBody {
		if len(ctx.Req.Params.Form) > 0 {
			ctx.Set(keyAahRequestDumpBody, ctx.Req.Params.Form.Encode())
		} else if ahttp.ContentTypePlainText.IsEqual(ctx.Req.ContentType.Mime) ||
			ahttp.ContentTypeHTML.IsEqual(ctx.Req.ContentType.Mime) {
			if b, err := ioutil.ReadAll(ctx.Req.Body()); err == nil {
				ctx.Set(keyAahRequestDumpBody, string(b))
				ctx.Req.Unwrap().Body = ioutil.NopCloser(bytes.NewReader(b))
			}
		}
	}

	return buf.String()
}

func composeResponseDump(ctx *Context) string {
	buf := acquireBuffer()
	defer releaseBuffer(buf)

	buf.WriteString(fmt.Sprintf("STATUS: %d %s\n", ctx.Res.Status(), http.StatusText(ctx.Res.Status())))
	buf.WriteString(fmt.Sprintf("BYTES WRITTEN: %d\n", ctx.Res.BytesWritten()))
	buf.WriteString("HEADERS:\n")
	buf.WriteString(composeHeaders(ctx.Res.Header()))

	return buf.String()
}

func composeHeaders(hdrs http.Header) string {
	var str []string
	for _, k := range sortHeaderKeys(hdrs) {
		str = append(str, fmt.Sprintf("    %s: %s", k, strings.Join(hdrs[k], ", ")))
	}
	return strings.Join(str, "\n")
}

func sortHeaderKeys(hdrs http.Header) []string {
	keys := make([]string, 0, len(hdrs))
	for key := range hdrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func addReqBodyIntoCtx(ctx *Context, result reflect.Value) {
	switch ctx.Req.ContentType.Mime {
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

func addResBodyIntoCtx(ctx *Context) {
	ct := ctx.Reply().ContType
	if ahttp.ContentTypeHTML.IsEqual(ct) || ahttp.ContentTypeJSON.IsEqual(ct) || ahttp.ContentTypeJSONText.IsEqual(ct) || ahttp.ContentTypeXML.IsEqual(ct) ||
		ahttp.ContentTypeXMLText.IsEqual(ct) || ahttp.ContentTypePlainText.IsEqual(ct) {
		ctx.Set(keyAahResponseDumpBody, ctx.Reply().Body().String())
	}
}

func dump(ctx *Context) {
	appDumpLog.Print(ctx.Get(keyAahRequestDump))
	if dumpRequestBody && ctx.Get(keyAahRequestDumpBody) != nil {
		appDumpLog.Printf("BODY:\n%v\n", ctx.Get(keyAahRequestDumpBody))
	} else {
		appDumpLog.Println()
	}
	appDumpLog.Print("-----------------------------------------------------------------------\n")
	appDumpLog.Print(composeResponseDump(ctx))
	if dumpResponseBody && ctx.Get(keyAahResponseDumpBody) != nil {
		appDumpLog.Printf("BODY:\n%v\n", ctx.Get(keyAahResponseDumpBody))
	} else {
		appDumpLog.Println()
	}
	appDumpLog.Print("=======================================================================")
}
