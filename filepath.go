// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/essentails source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Excludes is handly filepath match manipulation
type Excludes []string

// Validate helps to evalute the pattern are valid
// `Match` method is from error and focus on match
func (e *Excludes) Validate() error {
	for _, pattern := range *e {
		if _, err := filepath.Match(pattern, "abc/b/c"); err != nil {
			return fmt.Errorf("unable to evalute pattern: %v", pattern)
		}
	}
	return nil
}

// Match evalutes given file with available patterns returns true if matches
// otherwise false. `Match` internally uses the `filepath.Match`
//
// Note: `Match` ignore pattern errors, use `Validate` method to ensure
// you have correct exclude patterns
func (e *Excludes) Match(file string) bool {
	for _, pattern := range *e {
		if match, _ := filepath.Match(pattern, file); match {
			return match
		}
	}
	return false
}

// IsFileExists return true is file or directory is exists, otherwise returns
// false. It also take cares of symlink path as well
func IsFileExists(filename string) bool {
	_, err := os.Lstat(filename)
	return err == nil
}

// IsDirEmpty returns true if the given directory is empty also returns true if
// directory not exists. Otherwise returns false
func IsDirEmpty(path string) bool {
	if !IsFileExists(path) {
		// directory not exists
		return true
	}
	dir, _ := os.Open(path)
	defer CloseQuietly(dir)
	results, _ := dir.Readdir(1)
	return len(results) == 0
}

// IsDir returns true if the given `path` is directory otherwise returns false.
// Also returns false if path is not exists
func IsDir(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ApplyFileMode applies the given file mode to the target{file|directory}
func ApplyFileMode(target string, mode os.FileMode) error {
	err := os.Chmod(target, mode)
	if err != nil {
		return fmt.Errorf("unable to apply mode: %v, to given file or directory: %v", mode, target)
	}
	return nil
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

// Walk method extends filepath.Walk to also follows symlinks.
// Always returns the path of the file or directory also path
// is inline to name of symlink
func Walk(srcDir string, walkFn filepath.WalkFunc) error {
	return doWalk(srcDir, srcDir, walkFn)
}

func doWalk(fname string, linkName string, walkFn filepath.WalkFunc) error {
	fsWalkFn := func(path string, info os.FileInfo, err error) error {
		var name string
		name, err = filepath.Rel(fname, path)
		if err != nil {
			return err
		}

		path = filepath.Join(linkName, name)

		if err == nil && info.Mode()&os.ModeSymlink == os.ModeSymlink {
			var symlinkPath string
			symlinkPath, err = filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			// https://github.com/golang/go/blob/master/src/path/filepath/path.go#L392
			info, err = os.Lstat(symlinkPath)
			if err != nil {
				return walkFn(path, info, err)
			}

			if info.IsDir() {
				return doWalk(symlinkPath, path, walkFn)
			}
		}

		return walkFn(path, info, err)
	}

	return filepath.Walk(fname, fsWalkFn)
}

// CopyFile copies the given source file into destination
func CopyFile(dest, src string) (int64, error) {
	if !IsFileExists(src) {
		return 0, fmt.Errorf("source file is not exists: %v", src)
	}

	if IsFileExists(dest) {
		return 0, fmt.Errorf("destination file already exists: %v", dest)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("unable to create dest file: %v", dest)
	}
	defer CloseQuietly(destFile)

	srcFile, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("unable to open source file: %v", src)
	}
	defer CloseQuietly(srcFile)

	copiedBytes, err := io.Copy(destFile, srcFile)
	if err != nil {
		return 0, fmt.Errorf("unable to copy file from %v to %v (%v)",
			src, dest, err)
	}

	return copiedBytes, nil
}

// CopyDir copies entire directory, sub directories and files into destination
// and it excludes give file matches
func CopyDir(dest, src string, excludes Excludes) error {
	if !IsFileExists(src) {
		return fmt.Errorf("source dir is not exists: %v", src)
	}

	src = filepath.Clean(src)
	srcInfo, _ := os.Lstat(src)
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not directory: %v", src)
	}

	if IsFileExists(dest) {
		return fmt.Errorf("destination dir already exists: %v", dest)
	}

	if err := excludes.Validate(); err != nil {
		return err
	}

	return Walk(src, func(srcPath string, info os.FileInfo, err error) error {
		if excludes.Match(filepath.Base(srcPath)) {
			if info.IsDir() {
				// excluding directory
				return filepath.SkipDir
			}
			// excluding file
			return nil
		}

		relativeSrcPath := strings.TrimLeft(srcPath[len(src):], string(filepath.Separator))
		destPath := filepath.Join(dest, relativeSrcPath)

		if info.IsDir() {
			// directory permisions is not preserved from source
			return MkDirAll(destPath, 0755)
		}

		// copy source into destination
		if _, err = CopyFile(destPath, srcPath); err != nil {
			return err
		}

		// Apply source permision into target as well
		// so file permissions are preserved
		return ApplyFileMode(destPath, info.Mode())
	})
}
