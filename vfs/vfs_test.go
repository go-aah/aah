// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVFSMountAdd(t *testing.T) {
	fs := new(VFS)
	assert.False(t, fs.IsEmbeddedMode())

	testcases := []struct {
		label        string
		mountpath    string
		physicalpath string
		err          error
	}{
		{
			label:        "adding mount config directory",
			mountpath:    "/vfstest/config",
			physicalpath: filepath.Join(testdataBaseDir(), "vfstest", "config"),
			// err:          nil,
		},
		{
			label:        "adding same mount config directory",
			mountpath:    "/vfstest/config",
			physicalpath: filepath.Join(testdataBaseDir(), "vfstest", "config"),
			err:          &os.PathError{Op: "addmount", Path: "/vfstest/config", Err: ErrMountExists},
		},
		{
			label:        "adding mount view directory",
			mountpath:    "/vfstest/views",
			physicalpath: filepath.Join(testdataBaseDir(), "vfstest", "views"),
			// err:          nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			err := fs.AddMount(tc.mountpath, tc.physicalpath)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestVFSWalk(t *testing.T) {
	fs := createVFS(t)

	var walkList []string
	err := fs.Walk("/app", func(fpath string, info os.FileInfo, err error) error {
		walkList = append(walkList, fpath)
		return nil
	})
	assert.Nil(t, err)

	for _, p := range []string{
		"/app/views/common",
		"/app/views/pages/app/index.html",
		"/app/i18n/messages.en",
		"/app/config/env/prod.conf",
		"/app/static/js/aah.js",
		"/app/.gitignore",
	} {
		assert.Contains(t, walkList, p)
	}
}

func TestVFSOpenAndReadFile(t *testing.T) {
	fs := createVFS(t)

	testcases := []struct {
		fpath    string
		dir      bool
		size     int64
		contains []string
		gzip     bool
		mode     string
	}{
		{
			fpath:    "/app/views/pages/app/index.html",
			size:     335,
			contains: []string{".Greet.Message", "label.pages.app.index.title"},
			mode:     "-r--r--r--",
		},
		{
			fpath:    "/app/config/aah.conf",
			size:     16637,
			contains: []string{`name = "vfstest"`, `engine = "go"`, `default = "html"`},
			gzip:     true,
			mode:     "-r--r--r--",
		},
		{
			fpath: "/app/config/security.conf",
			size:  9352,
			contains: []string{"Anti-CSRF Protection",
				`sign_key = "1e2d3bdf896eb3df1fdac69dc8d90fc2b2345cd65e03a5c6dc916d9624d4d12c"`,
				`#secret_length = 32`},
			gzip: true,
			mode: "-r--r--r--",
		},
		{
			fpath: "/app/static/robots.txt",
			size:  68,
			contains: []string{"User-agent: *", "# Prevents all robots visiting your site.",
				"Disallow: /"},
			mode: "-r--r--r--",
		},
		{
			fpath: "/app/views/common",
			dir:   true,
			mode:  "drwxr-xr-x",
		},
		{
			fpath: "/app/views/pages/app",
			dir:   true,
			mode:  "drwxr-xr-x",
		},
	}

	for _, tc := range testcases {
		t.Run("open and readfile "+tc.fpath, func(t *testing.T) {
			f, err := Open(fs, tc.fpath)
			assert.Nil(t, err)

			// stats
			s, err := f.Stat()
			assert.Nil(t, err)
			assert.True(t, s.Size() >= tc.size)
			assert.Equal(t, tc.dir, s.IsDir())
			assert.Nil(t, s.Sys())
			assert.Equal(t, tc.mode, fmt.Sprintf("%s", s.Mode()))

			// gzip
			ff, ok := f.(Gziper)
			assert.True(t, ok)
			assert.Equal(t, tc.gzip, ff.IsGzip())

			// content
			if len(tc.contains) > 0 {
				assert.True(t, len(ff.RawBytes()) > 0)

				data, err := ReadFile(fs, tc.fpath)
				assert.Nil(t, err)

				sdata := string(data)
				for _, sv := range tc.contains {
					assert.Contains(t, sdata, sv)
				}
			}
		})
	}

	t.Log("not exists /app/views/not-exists")
	f, err := Open(fs, "/app/views/not-exists")
	assert.NotNil(t, err)
	assert.True(t, os.IsNotExist(err))
	assert.Contains(t, err.Error(), filepath.Join("views", "not-exists"))
	assert.Nil(t, f)

	t.Log("File string")
	s, err := Lstat(fs, "/app/views/errors/404.html")
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(fmt.Sprintf("%s", s), "file(name=404.html dir=false gzip=false"))

	s, err = Stat(fs, "/app/views/errors")
	assert.Nil(t, err)
	assert.True(t, strings.HasPrefix(fmt.Sprintf("%s", s), "file(name=errors dir=true gzip=false size=0,"))
}

func TestVFSReadDir(t *testing.T) {
	fs := createVFS(t)

	infos, err := fs.ReadDir("/app/config")
	assert.Nil(t, err)
	assert.True(t, len(infos) == 4)
	assert.True(t, strings.HasPrefix(fmt.Sprintf("%s", infos[1]), "node(name=env dir=true gzip=false size=0, modtime="))
	fi3 := infos[3]
	assert.Equal(t, "security.conf", fi3.Name())
	assert.False(t, fi3.IsDir())
	assert.True(t, fi3.(Gziper).IsGzip())
	assert.True(t, fi3.Size() >= 9352)
}

func TestVFSGlobAndIsExists(t *testing.T) {
	fs := createVFS(t)

	names, err := fs.Glob("/app/config/*")
	assert.Nil(t, err)
	assert.True(t, len(names) == 4)

	for _, p := range []string{
		"/app/config/routes.conf",
		"/app/config/security.conf",
		"/app/config/aah.conf",
		"/app/config/env",
	} {
		assert.True(t, fs.IsExists(p))
		assert.Contains(t, names, p)
	}
}

func TestVFSDirsAndFiles(t *testing.T) {
	fs := createVFS(t)

	dirs, err := fs.Dirs("/app/config")
	assert.Nil(t, err)
	assert.True(t, len(dirs) == 2)
	assert.Equal(t, "/app/config/env", dirs[1])

	files, err := fs.Files("/app/config")
	assert.Nil(t, err)
	assert.True(t, len(files) == 5)

	for _, p := range []string{
		"/app/config/routes.conf",
		"/app/config/env/prod.conf",
		"/app/config/aah.conf",
	} {
		assert.Contains(t, files, p)
	}
}

func createVFS(t *testing.T) *VFS {
	mountDir := filepath.Join(testdataBaseDir(), "vfstest")

	fs := new(VFS)
	err := fs.AddMount("/app", mountDir)
	assert.Nil(t, err)

	m, err := fs.FindMount("/app")
	assert.Nil(t, err)
	assert.NotNil(t, m)

	err = filepath.Walk(mountDir, func(fpath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err = m.AddDir(&NodeInfo{
				Dir:  info.IsDir(),
				Path: m.toVirtualPath(fpath),
				Time: info.ModTime(),
			})
			assert.Nil(t, err)
		} else {
			data, er := ioutil.ReadFile(fpath)
			assert.Nil(t, er)

			if info.Name() == "aah.conf" || info.Name() == "security.conf" {
				buf := new(bytes.Buffer)
				gw := gzip.NewWriter(buf)
				_, err = io.Copy(gw, bytes.NewReader(data))
				assert.Nil(t, err)
				err = gw.Close()
				assert.Nil(t, err)
				data = buf.Bytes()
			}

			err = m.AddFile(&NodeInfo{
				DataSize: info.Size(),
				Path:     m.toVirtualPath(fpath),
				Time:     info.ModTime(),
			}, data)
			assert.Nil(t, err)
		}
		return nil
	})
	assert.Nil(t, err)

	return fs
}

func testdataBaseDir() string {
	wd, _ := os.Getwd()
	if idx := strings.Index(wd, "testdata"); idx > 0 {
		wd = wd[:idx]
	}
	return filepath.Join(wd, "testdata")
}
