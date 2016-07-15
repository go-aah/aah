// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-aah/essentials"
)

var (
	errPathNotFound = errors.New("path not found")
)

type (
	node struct {
		label  string
		prefix string
		value  interface{}
		edges  edges
	}

	edges []*node
)

func (e edges) Len() int {
	return len(e)
}

func (e edges) Less(i, j int) bool {
	return e[i].label < e[j].label
}

func (e edges) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e edges) Sort() {
	sort.Sort(e)
}

func (n *node) isLeaf() bool {
	return len(n.edges) == 0
}

func (n *node) addEdge(a *node) {
	n.edges = append(n.edges, a)
	n.edges.Sort()
}

func (n *node) newNode(prefix string, value interface{}) *node {
	return &node{
		label:  newLabel(prefix),
		prefix: prefix,
		value:  value,
		edges:  make(edges, 0),
	}
}

func (n *node) findEdge(label string) *node {
	cnt := len(n.edges)
	idx := sort.Search(cnt, func(i int) bool {
		return n.edges[i].label >= label
	})
	if idx < cnt && n.edges[idx].label == label {
		return n.edges[idx]
	}
	return nil
}

func (n *node) add(path string, value interface{}) error {
	if ess.StrIsEmpty(path) {
		return errors.New("path is required value")
	}

	if len(n.prefix) > 0 || len(n.edges) > 0 {
		var t *node
		search := path
		for {
			idx := lcp(search, n.prefix)

			// split the edge
			if idx < len(n.prefix) {
				e := n.newNode(n.prefix[idx:], n.value)

				// move n edges to new edge and clean n edges
				e.edges = append(e.edges, n.edges...)
				n.edges = make(edges, 0)

				// add new created edge
				n.addEdge(e)

				// update the current node
				n.label = newLabel(n.prefix[:idx])
				n.prefix = n.prefix[:idx]
				n.value = nil
			}

			if idx < len(search) {
				search = search[idx:]

				// look for edge if not found, add it
				t = n
				n = n.findEdge(newLabel(search))
				if n == nil {
					t.addEdge(t.newNode(search, value))
					return nil
				}
				continue
			}

			if idx == len(n.prefix) && n.prefix == search {
				if n.value != nil {
					return fmt.Errorf("path already exists: %v", path)
				}
				n.value = value
				return nil
			}
		}
	} else {
		n.label = newLabel(path)
		n.prefix = path
		n.value = value
	}

	return nil
}

func (n *node) lookup(path string) (interface{}, error) {
	if !ess.StrIsEmpty(path) {
		search := path
		for {
			if len(search) == 0 || n == nil {
				return nil, errPathNotFound
			}

			if strings.HasPrefix(search, n.prefix) {
				if n.prefix == search {
					return n.value, nil
				}

				search = search[len(n.prefix):]
				n = n.findEdge(newLabel(search))
			}
		}
	}

	return nil, errPathNotFound
}

// finds longest common prefix between two string
// and returns position value
func lcp(s1, s2 string) int {
	max := min(len(s1), len(s2))
	pos := 0
	for pos < max && s1[pos] == s2[pos] {
		pos++
	}
	return pos
}

// simple min value helper
func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// creates label for the tree node
func newLabel(str string) string {
	if len(str) == 0 {
		return ""
	}
	return string([]rune(str)[0])
}
