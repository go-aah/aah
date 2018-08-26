// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// CallerInfo struct stores Go caller info
type CallerInfo struct {
	QualifiedName string
	FunctionName  string
	FileName      string
	File          string
	Line          int
}

// FunctionInfo structs Go function info
type FunctionInfo struct {
	Name          string
	Package       string
	QualifiedName string
}

// GetFunctionInfo method returns the function name for given interface value.
func GetFunctionInfo(f interface{}) (fi *FunctionInfo) {
	if f == nil {
		fi = &FunctionInfo{}
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// recovered
			fi = &FunctionInfo{}
		}
	}()

	info := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	info = strings.Replace(info, "%2e", ".", -1)
	idx := strings.LastIndexByte(info, '.')
	fi = &FunctionInfo{
		Name:          info[idx+1:],
		Package:       info[:idx-1],
		QualifiedName: info,
	}

	return
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
	fn = strings.Replace(fn, "%2e", ".", -1)

	return &CallerInfo{
		QualifiedName: fn,
		FunctionName:  fn[strings.LastIndex(fn, ".")+1:],
		File:          file,
		FileName:      filepath.Base(file),
		Line:          line,
	}
}
