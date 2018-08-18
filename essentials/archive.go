// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Zip method creates zip archive for given file or directory.
func Zip(dest, src string) error {
	if !IsFileExists(src) {
		return fmt.Errorf("source does not exists: %v", src)
	}

	if IsFileExists(dest) {
		return fmt.Errorf("destination archive already exists: %v", dest)
	}

	zf, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer CloseQuietly(zf)

	archive := zip.NewWriter(zf)
	defer CloseQuietly(archive)

	sinfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	var baseDir string
	if sinfo.IsDir() {
		baseDir = filepath.Base(src)
	}

	err = Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if !IsStrEmpty(baseDir) {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, src))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer CloseQuietly(file)

		_, err = io.Copy(writer, file)
		return err
	})

	return err
}
