// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/vfs source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package vfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/test.v0/assert"
)

func TestVFSMountAdd(t *testing.T) {
	fs := new(VFS)

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
			err:          nil,
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
			err:          nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			err := fs.AddMount(tc.mountpath, tc.physicalpath)
			fmt.Println("Err: ", err)
			assert.Equal(t, tc.err, err)
		})
	}
}

func testdataBaseDir() string {
	wd, _ := os.Getwd()
	if idx := strings.Index(wd, "testdata"); idx > 0 {
		wd = wd[:idx]
	}
	return filepath.Join(wd, "testdata")
}
