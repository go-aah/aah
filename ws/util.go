// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/ws source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ws

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"aahframe.work/aah/ahttp"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// IsDisconnected method is helper to identify error is disconnect related.
// If it is returns true otherwise false.
func IsDisconnected(err error) bool {
	switch err {
	case ErrConnectionClosed, ErrUseOfClosedConnection:
		return true
	}
	return false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package Unexported methods
//______________________________________________________________________________

// WriteHTTPError is to write WebSocket context error.
func writeHTTPError(w http.ResponseWriter, code int, body string) {
	w.Header().Set(ahttp.HeaderContentType, ahttp.ContentTypePlainText.String())
	w.Header().Set(ahttp.HeaderContentLength, strconv.Itoa(len(body)))
	w.WriteHeader(code)
	_, _ = w.Write([]byte(body))
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package Unexported methods
//______________________________________________________________________________

// createError method creates aah WebSocket error.
func createError(err error) error {
	if err == nil {
		return err
	}

	msg := err.Error()
	if strings.HasPrefix(msg, "ws closed") {
		return ErrConnectionClosed
	} else if err == io.EOF || strings.HasSuffix(msg, "use of closed network connection") {
		return ErrUseOfClosedConnection
	}
	return fmt.Errorf("aah%s", msg)
}
