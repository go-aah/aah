// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ess provides simple & useful utils for Go. aah framework utilizes
// essentails library across.
package ess

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	filePermission = os.FileMode(0755)
)

// StrIsEmpty returns true if strings is empty otherwise false
func StrIsEmpty(v string) bool {
	return len(strings.TrimSpace(v)) == 0
}

// LineCnt counts no. of lines on file
func LineCnt(fileName string) int {
	f, err := os.Open(fileName)
	if err != nil {
		return 0
	}
	defer CloseQuietly(f)

	return LineCntr(f)
}

// LineCntr counts no. of lines for given reader
func LineCntr(r io.Reader) int {
	buf := make([]byte, 8196)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return count
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}

	return count
}

// CloseQuietly closes `io.Closer` quietly. Very handy and helpful for code
// quality too.
func CloseQuietly(v interface{}) {
	if d, ok := v.(io.Closer); ok {
		_ = d.Close()
	}
}

// MkDirAll method creates nested directories with permission 0755 if not exists
func MkDirAll(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, filePermission); err != nil {
				return fmt.Errorf("unable to create directory '%v': %v", path, err)
			}
		}
	}
	return nil
}
