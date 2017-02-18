// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// Copyright (c) 2013 Julien Schmidt (https://github.com/julienschmidt)
// All rights reserved.
// Use of this radix_tree.go source code is governed by a BSD-style license that can be found
// in the LICENSE file at https://raw.githubusercontent.com/julienschmidt/httprouter/master/LICENSE.
//
// Customized and improved for aah framework purpose.
// From upstream updated as of last commit date Dec 03, 2016 git#d35c3c3.

package router

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// SlashString const for comparison use
	SlashString = "/"

	dotByte   = '.'
	slashByte = '/'
	paramByte = ':'
	wildByte  = '*'
)

const (
	static nodeType = iota // default
	root
	param
	catchAll
)

var (
	errInvalidNodeType = errors.New("invalid node type")
)

type nodeType uint8

type node struct {
	path      string
	wildChild bool
	nType     nodeType
	maxParams uint8
	indices   string
	priority  uint32
	edges     []*node
	value     interface{}
}

// increments priority of the given edge and reorders if necessary
func (n *node) incrementEdgePriority(pos int) int {
	n.edges[pos].priority++
	prio := n.edges[pos].priority

	// adjust position (move to front)
	newPos := pos
	for newPos > 0 && n.edges[newPos-1].priority < prio {
		// swap node positions
		n.edges[newPos-1], n.edges[newPos] = n.edges[newPos], n.edges[newPos-1]
		newPos--
	}

	// build new index char string
	if newPos != pos {
		n.indices = n.indices[:newPos] + // unchanged prefix, might be empty
			n.indices[pos:pos+1] + // the index char we move
			n.indices[newPos:pos] + n.indices[pos+1:] // rest without char at 'pos'
	}

	return newPos
}

// add adds a node with the given value againsts given path.
// Not concurrency-safe!
func (n *node) add(path string, value interface{}) error {
	fullPath := path
	n.priority++
	numParams := countParams(path)

	// non-empty tree
	if len(n.path) > 0 || len(n.edges) > 0 {
	walk:
		for {
			// Update maxParams of the current node
			if numParams > n.maxParams {
				n.maxParams = numParams
			}

			// Find the longest common prefix.
			// This also implies that the common prefix contains no ':' or '*'
			// since the existing key can't contain those chars.
			i := 0
			max := min(len(path), len(n.path))
			for i < max && path[i] == n.path[i] {
				i++
			}

			// Split edge
			if i < len(n.path) {
				edge := node{
					path:      n.path[i:],
					wildChild: n.wildChild,
					nType:     static,
					indices:   n.indices,
					priority:  n.priority - 1,
					edges:     n.edges,
					value:     n.value,
				}

				// Update maxParams (max of all edges)
				for i := range edge.edges {
					if edge.edges[i].maxParams > edge.maxParams {
						edge.maxParams = edge.edges[i].maxParams
					}
				}

				n.edges = []*node{&edge}
				// []byte for proper unicode char conversion
				n.indices = string([]byte{n.path[i]})
				n.path = path[:i]
				n.value = nil
				n.wildChild = false
			}

			// Make new node a edge of this node
			if i < len(path) {
				path = path[i:]

				if n.wildChild {
					n = n.edges[0]
					n.priority++

					// Update maxParams of the edge node
					if numParams > n.maxParams {
						n.maxParams = numParams
					}
					numParams--

					// Check if the wildcard matches
					if len(path) >= len(n.path) && n.path == path[:len(n.path)] &&
						// Check for longer wildcard, e.g. :name and :names
						(len(n.path) >= len(path) || path[len(n.path)] == slashByte) {
						continue walk
					} else {
						// Wildcard conflict
						pathSeg := strings.SplitN(path, SlashString, 2)[0]
						prefix := fullPath[:strings.Index(fullPath, pathSeg)] + n.path

						return fmt.Errorf("'%s' in new path '%s' conflicts with existing "+
							"wildcard '%s' in existing prefix '%s'", pathSeg, fullPath, n.path, prefix)
					}
				}

				c := path[0]

				// slash after param
				if n.nType == param && c == slashByte && len(n.edges) == 1 {
					n = n.edges[0]
					n.priority++
					continue walk
				}

				// Check if a edge with the next path byte exists
				for i = 0; i < len(n.indices); i++ {
					if c == n.indices[i] {
						i = n.incrementEdgePriority(i)
						n = n.edges[i]
						continue walk
					}
				}

				// Otherwise insert it
				if c != paramByte && c != wildByte {
					// []byte for proper unicode char conversion
					n.indices += string([]byte{c})
					edge := &node{
						maxParams: numParams,
					}
					n.edges = append(n.edges, edge)
					n.incrementEdgePriority(len(n.indices) - 1)
					n = edge
				}
				if err := n.insertEdge(numParams, path, fullPath, value); err != nil {
					return err
				}

				return nil

			} else if i == len(path) { // Make node a (in-path) leaf
				if n.value != nil {
					return fmt.Errorf("a value is already registered for path '%s'", fullPath)
				}

				n.value = value
			}
			return nil
		}
	} else { // Empty tree
		if err := n.insertEdge(numParams, path, fullPath, value); err != nil {
			return err
		}

		n.nType = root
	}

	return nil
}

func (n *node) insertEdge(numParams uint8, path, fullPath string, value interface{}) error {
	var offset int // already handled bytes of the path

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != paramByte && c != wildByte {
			continue
		}

		// find wildcard end (either '/' or path end)
		end := i + 1
		for end < max && path[end] != slashByte {
			switch path[end] {
			// the wildcard name must not contain ':' and '*'
			case paramByte, wildByte:
				return fmt.Errorf("only one wildcard per path segment is allowed, "+
					"has: '%s' in path '%s'", path[i:], fullPath)
			default:
				end++
			}
		}

		// check if this Node existing edges which would be
		// unreachable if we insert the wildcard here
		if len(n.edges) > 0 {
			return fmt.Errorf("wildcard route '%s' conflicts with existing"+
				" edges in path '%s'", path[i:end], fullPath)
		}

		// check if the wildcard has a name
		if end-i < 2 {
			return fmt.Errorf("wildcards must be named with a non-empty name"+
				" in path '%s'", fullPath)
		}

		if c == paramByte { // param
			// split path at the beginning of the wildcard
			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			edge := &node{
				nType:     param,
				maxParams: numParams,
			}
			n.edges = []*node{edge}
			n.wildChild = true
			n = edge
			n.priority++
			numParams--

			// if the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if end < max {
				n.path = path[offset:end]
				offset = end

				edge := &node{
					maxParams: numParams,
					priority:  1,
				}
				n.edges = []*node{edge}
				n = edge
			}

		} else { // catchAll
			if end != max || numParams > 1 {
				return fmt.Errorf("catch-all routes are only allowed at the end of"+
					" the path in path '%s'", fullPath)
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == slashByte {
				return fmt.Errorf("catch-all conflicts with existing value for the"+
					" path segment root in path '%s'", fullPath)
			}

			// currently fixed width 1 for '/'
			i--
			if path[i] != slashByte {
				return fmt.Errorf("no / before catch-all in path '%s'", fullPath)
			}

			n.path = path[offset:i]

			// first node: catchAll node with empty path
			edge := &node{
				wildChild: true,
				nType:     catchAll,
				maxParams: 1,
			}
			n.edges = []*node{edge}
			n.indices = string(path[i])
			n = edge
			n.priority++

			// second node: node holding the variable
			edge = &node{
				path:      path[i:],
				nType:     catchAll,
				maxParams: 1,
				priority:  1,
				value:     value,
			}
			n.edges = []*node{edge}

			return nil
		}
	}

	// insert remaining path part and value to the leaf
	n.path = path[offset:]
	n.value = value

	return nil
}

// find returns the value registered with the given path (key). The values of
// wildcards are saved to a map. If no value can be found, a TSR (trailing slash
// redirect) recommendation is made if a value exists with an extra (without
// the) trailing slash for the given path.
func (n *node) find(path string) (value interface{}, p PathParams, tsr bool, err error) {
walk: // outer loop for walking the tree
	for {
		if len(path) > len(n.path) {
			if path[:len(n.path)] == n.path {
				path = path[len(n.path):]
				// If this node does not have a wildcard (param or catchAll)
				// edge,  we can just look up the next edge node and continue
				// to walk down the tree
				if !n.wildChild {
					c := path[0]
					for i := 0; i < len(n.indices); i++ {
						if c == n.indices[i] {
							n = n.edges[i]
							continue walk
						}
					}

					// Nothing found.
					// We can recommend to redirect to the same URL without a
					// trailing slash if a leaf exists for that path.
					tsr = (path == SlashString && n.value != nil)
					return

				}

				// value wildcard edge
				n = n.edges[0]
				switch n.nType {
				case param:
					// find param end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != slashByte {
						end++
					}

					// save param value
					if p == nil {
						// lazy allocation
						p = make(PathParams, 0, n.maxParams)
					}
					i := len(p)
					p = p[:i+1] // expand slice within preallocated capacity
					p[i].Key = n.path[1:]
					p[i].Value = path[:end]

					// we need to go deeper!
					if end < len(path) {
						if len(n.edges) > 0 {
							path = path[end:]
							n = n.edges[0]
							continue walk
						}

						// ... but we can't
						tsr = (len(path) == end+1)
						return
					}

					if value = n.value; value != nil {
						return
					} else if len(n.edges) == 1 {
						// No value found. Check if a value for this path + a
						// trailing slash exists for TSR recommendation
						n = n.edges[0]
						tsr = (n.path == SlashString && n.value != nil)
					}

					return

				case catchAll:
					// save param value
					if p == nil {
						// lazy allocation
						p = make(PathParams, 0, n.maxParams)
					}
					i := len(p)
					p = p[:i+1] // expand slice within preallocated capacity
					p[i].Key = n.path[2:]
					p[i].Value = path

					value = n.value
					return

				default:
					err = errInvalidNodeType
					return
				}
			}
		} else if path == n.path {
			// We should have reached the node containing the value.
			// Check if this node has a value registered.
			if value = n.value; value != nil {
				return
			}

			if path == SlashString && n.wildChild && n.nType != root {
				tsr = true
				return
			}

			// No value found. Check if a value for this path + a
			// trailing slash exists for trailing slash recommendation
			for i := 0; i < len(n.indices); i++ {
				if n.indices[i] == slashByte {
					n = n.edges[i]
					tsr = (len(n.path) == 1 && n.value != nil) ||
						(n.nType == catchAll && n.edges[0].value != nil)
					return
				}
			}

			return
		}

		// Nothing found. We can recommend to redirect to the same URL with an
		// extra trailing slash if a leaf exists for that path
		tsr = (path == SlashString) ||
			(len(n.path) == len(path)+1 && n.path[len(path)] == slashByte &&
				path == n.path[:len(n.path)-1] && n.value != nil)
		return
	}
}

// Makes a case-insensitive lookup of the given path and tries to find a handler.
// It can optionally also fix trailing slashes.
// It returns the case-corrected path and a bool indicating whether the lookup
// was successful.
func (n *node) findCaseInsensitive(path string, fixTrailingSlash bool) (string, bool, error) {
	ciPath, found, err := n.findCaseInsensitiveRec(
		path,
		strings.ToLower(path),
		make([]byte, 0, len(path)+1), // preallocate enough memory for new path
		[4]byte{},                    // empty rune buffer
		fixTrailingSlash,
	)

	return string(ciPath), found, err
}

// shift bytes in array by n bytes left
func shiftNRuneBytes(rb [4]byte, n int) [4]byte {
	switch n {
	case 0:
		return rb
	case 1:
		return [4]byte{rb[1], rb[2], rb[3], 0}
	case 2:
		return [4]byte{rb[2], rb[3]}
	case 3:
		return [4]byte{rb[3]}
	default:
		return [4]byte{}
	}
}

// recursive case-insensitive lookup function used by n.findCaseInsensitive
func (n *node) findCaseInsensitiveRec(path, loPath string, ciPath []byte, rb [4]byte, fixTrailingSlash bool) ([]byte, bool, error) {
	loNPath := strings.ToLower(n.path)

walk: // outer loop for walking the tree
	for len(loPath) >= len(loNPath) && (len(loNPath) == 0 || loPath[1:len(loNPath)] == loNPath[1:]) {
		// add common path to result
		ciPath = append(ciPath, n.path...)

		if path = path[len(n.path):]; len(path) > 0 {
			loOld := loPath
			loPath = loPath[len(loNPath):]

			// If this node does not have a wildcard (param or catchAll) edge,
			// we can just look up the next edge node and continue to walk down
			// the tree
			if !n.wildChild {
				// skip rune bytes already processed
				rb = shiftNRuneBytes(rb, len(loNPath))

				if rb[0] != 0 {
					// old rune not finished
					for i := 0; i < len(n.indices); i++ {
						if n.indices[i] == rb[0] {
							// continue with edge node
							n = n.edges[i]
							loNPath = strings.ToLower(n.path)
							continue walk
						}
					}
				} else {
					// process a new rune
					var rv rune

					// find rune start
					// runes are up to 4 byte long,
					// -4 would definitely be another rune
					var off int
					for max := min(len(loNPath), 3); off < max; off++ {
						if i := len(loNPath) - off; utf8.RuneStart(loOld[i]) {
							// read rune from cached lowercase path
							rv, _ = utf8.DecodeRuneInString(loOld[i:])
							break
						}
					}

					// calculate lowercase bytes of current rune
					utf8.EncodeRune(rb[:], rv)
					// skipp already processed bytes
					rb = shiftNRuneBytes(rb, off)

					for i := 0; i < len(n.indices); i++ {
						// lowercase matches
						if n.indices[i] == rb[0] {
							// must use a recursive approach since both the
							// uppercase byte and the lowercase byte might exist
							// as an index
							if out, found, err := n.edges[i].findCaseInsensitiveRec(
								path, loPath, ciPath, rb, fixTrailingSlash,
							); found {
								return out, found, err
							}
							break
						}
					}

					// same for uppercase rune, if it differs
					if up := unicode.ToUpper(rv); up != rv {
						utf8.EncodeRune(rb[:], up)
						rb = shiftNRuneBytes(rb, off)

						for i := 0; i < len(n.indices); i++ {
							// uppercase matches
							if n.indices[i] == rb[0] {
								// continue with edge node
								n = n.edges[i]
								loNPath = strings.ToLower(n.path)
								continue walk
							}
						}
					}
				}

				// Nothing found. We can recommend to redirect to the same URL
				// without a trailing slash if a leaf exists for that path
				return ciPath, (fixTrailingSlash && path == SlashString && n.value != nil), nil
			}

			n = n.edges[0]
			switch n.nType {
			case param:
				// find param end (either '/' or path end)
				k := 0
				for k < len(path) && path[k] != slashByte {
					k++
				}

				// add param value to case insensitive path
				ciPath = append(ciPath, path[:k]...)

				// we need to go deeper!
				if k < len(path) {
					if len(n.edges) > 0 {
						// continue with edge node
						n = n.edges[0]
						loNPath = strings.ToLower(n.path)
						loPath = loPath[k:]
						path = path[k:]
						continue
					}

					// ... but we can't
					if fixTrailingSlash && len(path) == k+1 {
						return ciPath, true, nil
					}
					return ciPath, false, nil
				}

				if n.value != nil {
					return ciPath, true, nil
				} else if fixTrailingSlash && len(n.edges) == 1 {
					// No value found. Check if a value for this path + a
					// trailing slash exists
					n = n.edges[0]
					if n.path == SlashString && n.value != nil {
						return append(ciPath, slashByte), true, nil
					}
				}
				return ciPath, false, nil

			case catchAll:
				return append(ciPath, path...), true, nil

			default:
				return nil, false, errInvalidNodeType
			}
		} else {
			// We should have reached the node containing the value.
			// Check if this node has a value registered.
			if n.value != nil {
				return ciPath, true, nil
			}

			// No value found.
			// Try to fix the path by adding a trailing slash
			if fixTrailingSlash {
				for i := 0; i < len(n.indices); i++ {
					if n.indices[i] == slashByte {
						n = n.edges[i]
						if (len(n.path) == 1 && n.value != nil) ||
							(n.nType == catchAll && n.edges[0].value != nil) {
							return append(ciPath, slashByte), true, nil
						}
						return ciPath, false, nil
					}
				}
			}
			return ciPath, false, nil
		}
	}

	// Nothing found.
	// Try to fix the path by adding / removing a trailing slash
	if fixTrailingSlash {
		if path == SlashString {
			return ciPath, true, nil
		}
		if len(loPath)+1 == len(loNPath) && loNPath[len(loPath)] == slashByte &&
			loPath[1:] == loNPath[1:len(loPath)] && n.value != nil {
			return append(ciPath, n.path...), true, nil
		}
	}
	return ciPath, false, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func countParams(path string) uint8 {
	var n uint
	for i := 0; i < len(path); i++ {
		if path[i] != paramByte && path[i] != wildByte {
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}
