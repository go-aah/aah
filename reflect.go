// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// CallerInfo struct is store Go caller info
type CallerInfo struct {
	QualifiedName string
	FunctionName  string
	FileName      string
	File          string
	Line          int
}

// GetFunctionName method returns the function name for given interface value.
func GetFunctionName(f interface{}) string {
	if f == nil {
		return ""
	}

	defer func() {
		if r := recover(); r != nil {
			// recovered
		}
	}()

	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// GetCallerInfo method returns caller's QualifiedName, FunctionName, File,
// FileName, Line Number.
func GetCallerInfo() *CallerInfo {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	fn := runtime.FuncForPC(pc).Name()

	return &CallerInfo{
		QualifiedName: fn,
		FunctionName:  fn[strings.LastIndex(fn, ".")+1:],
		File:          file,
		FileName:      filepath.Base(file),
		Line:          line,
	}
}
