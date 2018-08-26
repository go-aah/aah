// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"aahframe.work/aah/ahttp"
)

var errNodeExists = errors.New("aah/router: node exists")

type nodeType uint8

const (
	staticNode   nodeType = iota // 0 => static segment in the path
	paramNode                    // 1 => named parameter segment in the path e.g. /path/:to/:route/value
	wildcardNode                 // 2 => wildcard segment at the end of the path e.g. /path/to/route/*value
)

const (
	// SlashString const for comparison use
	SlashString = "/"

	dotByte   = '.'
	slashByte = '/'
	paramByte = ':'
	wildByte  = '*'
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// tree struct and its methods
//______________________________________________________________________________

type tree struct {
	tralingSlash bool
	maxParams    uint8
	root         *node
}

func (t *tree) lookup(p string) (r *Route, params ahttp.URLParams, rts bool) {
	s, l, sn, pn := strings.ToLower(p), len(p), t.root, t.root
	ll := l
walk:
	for {
		if sn == nil {
			r, params = nil, nil
			return
		}
		i := 0
		if sn.typ == staticNode {
			max := len(sn.label)
			if ll <= max {
				max = ll
			}
			for i < max && s[i] == sn.label[i] {
				i++
			}
			if i != max {
				r, params = nil, nil
				return
			}
		} else if sn.typ == paramNode {
			for i < ll && s[i] != slashByte {
				i++
			}
			if params == nil {
				params = make(ahttp.URLParams, 0, t.maxParams)
			}
			j := len(params)
			params = params[:j+1]
			params[j].Key = sn.arg
			params[j].Value = p[:i]
		} else if sn.typ == wildcardNode {
			if params == nil {
				params = make(ahttp.URLParams, 0, t.maxParams)
			}
			j := len(params)
			params = params[:j+1]
			params[j].Key = sn.arg
			params[j].Value = p[i:]
			r = sn.value
			return
		}
		s, p = s[i:], p[i:]
		ll = len(s)
		if ll == 0 {
			if (i < len(sn.label) || sn.value == nil) && t.tralingSlash {
				if sn.label[len(sn.label)-1] == slashByte && sn.value != nil {
					r, params, rts = nil, nil, true
				} else if sn = sn.findByIdx(slashByte); sn != nil && sn.value != nil {
					r, params, rts = nil, nil, true
				} else if pn.value != nil {
					r, params, rts = nil, nil, true
				}
				return
			} else if sn.value != nil { // edge found
				r = sn.value
				return
			}
			params = nil
			return
		} else if ll == 1 && s == SlashString && len(sn.edges) == 0 {
			if sn.value != nil {
				r, params, rts = nil, nil, true
				return
			}
			r, params = nil, nil
			return
		}

		for _, e := range sn.edges {
			if e.idx == s[0] && e.typ == staticNode {
				pn = sn
				sn = e
				continue walk
			}
		}
		pn = sn
		sn = sn.wnode
	}
}

func (t *tree) add(p string, r *Route) error {
	fp := p
	p = strings.ToLower(p)
	var err error
	maxParams := countParams(p)
	if maxParams > t.maxParams {
		t.maxParams = maxParams
	}
	for i, l := 0, len(p); i < l; i++ {
		switch p[i] {
		case paramByte:
			_ = t.insertEdge(staticNode, p[:i], "", nil)
			j := i + 1
			for i < l && p[i] != slashByte {
				i++
			}
			arg := fp[j:i]
			if err = checkParameter(p, arg); err != nil {
				return err
			}

			p, fp = p[:j]+p[i:], fp[:j]+fp[i:]
			i, l = j, len(p)
			if i == l {
				return t.insertEdge(paramNode, p[:i], arg, r)
			}
			if err = t.insertEdge(paramNode, p[:i], arg, nil); err != nil {
				return err
			}
		case wildByte:
			if idx := strings.IndexByte(p[i+1:], slashByte); idx > 0 {
				return fmt.Errorf("incorrect use of wildcard URL param [%s]."+
					" It should come as last param [%s]", fp, fp[:i+idx+1])
			} else if err = checkParameter(p, fp[i+1:]); err != nil {
				return err
			}
			_ = t.insertEdge(staticNode, p[:i], "", nil)
			return t.insertEdge(wildcardNode, p[:i+1], fp[i+1:], r)
		}
	}
	return t.insertEdge(staticNode, p, "", r)
}

func (t *tree) insertEdge(typ nodeType, p, arg string, r *Route) error {
	s, sn := p, t.root
	var err error
	for {
		i, max := 0, min(len(s), len(sn.label))
		for i < max && s[i] == sn.label[i] {
			i++
		}
		switch {
		case i == 0: // assign to current/root node
			sn.idx = s[0]
			sn.label = s
			sn.typ = typ
			if r != nil {
				sn.typ = typ
				sn.value = r
				sn.arg = arg
			}
		case i < len(sn.label): // split the node
			edge := newNode(sn.typ, sn.label[i:], sn.arg, sn.value, sn.edges)
			sn.typ = staticNode
			sn.idx = sn.label[0]
			sn.label = sn.label[:i]
			sn.value = nil
			sn.arg = ""
			sn.edges = []*node{edge}
			if i == len(s) {
				if sn.value != nil {
					return errNodeExists
				}
				sn.typ = typ
				sn.value = r
				sn.arg = arg
			} else if err = sn.addEdge(p, newNode(typ, s[i:], arg, r, []*node{})); err != nil {
				return err
			}
		case i < len(s): // navigate, check and add new edge
			s = s[i:]
			if n := sn.findByIdx(s[0]); n != nil {
				if len(s) == 1 && len(n.arg) > 0 && n.arg != arg {
					return fmt.Errorf("aah/router: parameter based edge already exists[%s%s...] new[%s%s...]", p, n.arg, p, arg)
				}
				sn = n
				continue
			}
			if err = sn.addEdge(p, newNode(typ, s, arg, r, []*node{})); err != nil {
				return err
			}
		default:
			if r != nil {
				if sn.value != nil {
					return errNodeExists
				}
				sn.value = r
			}
		}
		return nil
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// node struct and its methods
//______________________________________________________________________________

type node struct {
	idx   byte
	typ   nodeType
	label string
	arg   string
	value *Route
	wnode *node
	edges []*node
}

// String method returns string representation of node.
func (n node) String() string {
	var typ string
	switch n.typ {
	case staticNode:
		typ = "static-node"
	case paramNode:
		typ = "param-node"
	case wildcardNode:
		typ = "wildcard-node"
	}

	var value string
	if n.value != nil {
		value = fmt.Sprintf("value->%v", n.value)
	}

	return fmt.Sprintf("%s%s type->%v edges->%v %v", n.label, n.arg, typ, len(n.edges), value)
}

func (n *node) addEdge(p string, nn *node) error {
	switch nn.typ {
	case paramNode:
		if c := n.findByIdx(wildByte); c != nil {
			return fmt.Errorf("aah/router: parameter based edge already exists[%s%s%s...] new[%s%s...]",
				p[:len(p)-1], string(c.idx), c.arg, p, nn.arg)
		}
	case wildcardNode:
		if c := n.findByIdx(paramByte); c != nil {
			return fmt.Errorf("aah/router: parameter based edge already exists[%s%s%s...] new[%s%s...]",
				p[:len(p)-1], string(c.idx), c.arg, p, nn.arg)
		}
	}
	n.edges = append(n.edges, nn)
	return nil
}

func (n *node) inferwnode() {
	for _, e := range n.edges {
		if e.typ == paramNode || e.typ == wildcardNode {
			n.wnode = e
			break
		}
	}
	for _, e := range n.edges {
		e.inferwnode()
	}
}

func (n *node) findByIdx(i byte) *node {
	for _, e := range n.edges {
		if e.idx == i {
			return e
		}
	}
	return nil
}

func (n *node) printTree(w io.Writer, level int) {
	space := ""
	for i := 0; i < level; i++ {
		space += "  "
	}
	fmt.Fprintln(w, space, n)
	for _, e := range n.edges {
		e.printTree(w, level+1)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Helper methods
//______________________________________________________________________________

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func newNode(typ nodeType, label, arg string, value *Route, edges []*node) *node {
	return &node{
		typ:   typ,
		idx:   label[0],
		label: label,
		value: value,
		arg:   arg,
		edges: edges,
	}
}

func checkParameter(p, arg string) error {
	if len(arg) == 0 {
		return fmt.Errorf("aah/router: parameter name required: '%s'", p)
	}
	if strings.IndexByte(arg, ':') > 0 || strings.IndexByte(arg, '*') > 0 {
		return fmt.Errorf("aah/router: only one paramter allowed in the path segment: '%s'", p)
	}
	return nil
}

func countParams(p string) uint8 {
	var n uint
	for i := 0; i < len(p); i++ {
		if p[i] != paramByte && p[i] != wildByte {
			continue
		}
		n++
	}
	if n >= 255 {
		return 255
	}
	return uint8(n)
}
