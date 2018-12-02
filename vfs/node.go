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
	"path"
	"strings"
	"time"
)

var _ os.FileInfo = (*NodeInfo)(nil)
var _ os.FileInfo = (*node)(nil)

// Gzip Member header
// RFC 1952 section 2.3 and 2.3.1
var gzipMemberHeader = []byte("\x1F\x8B\x08")

// NodeInfo is used to collect `os.FileInfo` values during binary generation.
type NodeInfo struct {
	Dir      bool
	DataSize int64
	Path     string
	Time     time.Time
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// os.FileInfo interface
//______________________________________________________________________________

// Name method returns base name of the file/directory.
func (n NodeInfo) Name() string {
	return path.Base(n.Path)
}

// Size method returns length in bytes for regular files;
// system-dependent for others
func (n NodeInfo) Size() int64 {
	if n.IsDir() {
		return 0
	}
	return n.DataSize
}

// Mode method returns file mode bits.
func (n NodeInfo) Mode() os.FileMode {
	if n.IsDir() {
		return 0755 | os.ModeDir // drwxr-xr-x
	}
	return 0444 // -r--r--r--
}

// ModTime method returns modification time.
func (n NodeInfo) ModTime() time.Time {
	return n.Time
}

// IsDir method returns whether node is dierctory or not.
func (n NodeInfo) IsDir() bool {
	return n.Dir
}

// Sys method returns nil.
func (n NodeInfo) Sys() interface{} {
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Node and its methods
//______________________________________________________________________________

// Node represents the virtual Node of file/directory on mounted VFS.
//
// Implements interfaces `os.FileInfo` and `vfs.Gziper`.
type node struct {
	*NodeInfo
	data       []byte
	childInfos []os.FileInfo
	childs     map[string]*node
}

// String method Stringer interface.
func (n node) String() string {
	return fmt.Sprintf(`node(name=%s dir=%v gzip=%v size=%v, modtime=%v)`,
		n.Name(), n.IsDir(), n.IsGzip(), n.Size(), n.ModTime())
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Gziper interface methods
//______________________________________________________________________________

// IsGzip method returns true if its statisfies Gzip Member header
// RFC 1952 section 2.3 and 2.3.1 otherwise false.
func (n node) IsGzip() bool {
	return bytes.HasPrefix(n.data, gzipMemberHeader)
}

func (n node) RawBytes() []byte {
	return n.data
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Node unexported methods
//______________________________________________________________________________

func (n *node) find(name string) (*file, error) {
	tn, err := n.findNode(name)
	if err != nil {
		return nil, err
	}

	if tn.match(name) {
		return newFile(tn), nil
	}

	return nil, os.ErrNotExist
}

func (n *node) findNode(name string) (*node, error) {
	switch name {
	case ".":
		return nil, nil
	case "/":
		return n, nil
	}

	search := strings.Split(strings.TrimLeft(name, "/"), "/")
	if len(search) == 0 {
		return nil, os.ErrNotExist
	}

	tn := n
	for _, s := range search {
		if t, found := tn.childs[s]; found {
			tn = t
		} else {
			break
		}
	}

	return tn, nil
}

func (n *node) match(name string) bool {
	return strings.EqualFold(n.Name(), path.Base(name))
}

func (n *node) addChild(child *node) {
	n.childInfos = append(n.childInfos, child)
	n.childs[child.Name()] = child
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// GzipData type and methods
//______________________________________________________________________________

var _ io.Reader = (*gzipData)(nil)
var _ io.Seeker = (*gzipData)(nil)
var _ io.Closer = (*gzipData)(nil)

// GzipData my goal is to expose transparent behavior for regular and gzip
// data bytes. So I have designed gzip data handing.
type gzipData struct {
	n    *node
	r    *gzip.Reader
	rpos int64
	spos int64
}

// Imitate regular read in gzip reader
// https://github.com/shurcooL/vfsgen/blob/master/generator.go
func (g *gzipData) Read(b []byte) (int, error) {
	if g.rpos > g.spos { // to the beginning
		if err := g.r.Reset(bytes.NewReader(g.n.data)); err != nil {
			return 0, err
		}
		g.rpos = 0
	}

	if g.rpos < g.spos { // move forward
		if _, err := io.CopyN(ioutil.Discard, g.r, g.spos-g.rpos); err != nil {
			return 0, err
		}
		g.rpos = g.spos
	}

	size, err := g.r.Read(b)
	g.rpos += int64(size)
	g.spos = g.rpos

	return size, err
}

// Imitate regular seek in gzip reader
// https://github.com/shurcooL/vfsgen/blob/master/generator.go
func (g *gzipData) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		g.spos = 0 + offset
	case io.SeekCurrent:
		g.spos += offset
	case io.SeekEnd:
		g.spos = g.n.Size() + offset
	default:
		return 0, fmt.Errorf("invalid whence: %v", whence)
	}
	return g.spos, nil
}

func (g *gzipData) Close() error {
	return g.r.Close()
}
