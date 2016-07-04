// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

func strIsEmpty(v string) bool {
	return len(strings.TrimSpace(v)) == 0
}

func closeSilently(v interface{}) {
	if d, ok := v.(io.Closer); ok {
		_ = d.Close()
	}
}

func lines(fileName string) int {
	f, err := os.Open(fileName)
	if err != nil {
		return 0
	}
	defer closeSilently(f)

	buf := make([]byte, 8196)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := f.Read(buf)
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

// createDir method creates nested directories if not exists
func mkDirAll(path string) error {
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, filePermission); err != nil {
				return fmt.Errorf("unable to create directory '%v': %v", path, err)
			}
		} else {
			return fmt.Errorf("unable to create directory '%v': %v", path, err)
		}
	}
	return nil
}
