// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"fmt"
	"path"
	"strings"

	"aahframework.org/essentials.v0"
)

const (
	ruleStartByte = '['
	ruleEndByte   = ']'
)

func suffixCommaValue(s, v string) string {
	if ess.IsStrEmpty(s) {
		return v
	}
	return s + ", " + v
}

func findActionByHTTPMethod(method string) string {
	if action, found := HTTPMethodActionMap[method]; found {
		return action
	}
	return ""
}

func addRegisteredAction(methods map[string]map[string]uint8, route *Route) {
	if target, found := methods[route.Target]; found {
		target[route.Action] = 1
	} else {
		methods[route.Target] = map[string]uint8{route.Action: 1}
	}
}

func routeConstraintExists(routePath string) bool {
	sidx := strings.IndexByte(routePath, ruleStartByte)
	eidx := strings.IndexByte(routePath, ruleEndByte)
	return sidx > 0 || eidx > 0
}

// Return values are -
// 1. route path
// 2. route constraints
// 3. error
func parseRouteConstraints(routeName, routePath string) (string, map[string]string, error) {
	if !routeConstraintExists(routePath) {
		return routePath, nil, nil
	}

	constraints := make(map[string]string)
	actualRoutePath := "/"
	for _, seg := range strings.Split(routePath, "/")[1:] {
		if seg[0] == paramByte || seg[0] == wildByte {
			param, constraint, exists, valid := parameterConstraint(seg)
			if exists {
				if valid {
					constraints[param[1:]] = constraint
				} else {
					return routePath, constraints, fmt.Errorf("'%s.path' has invalid contraint in path => '%s' (param => '%s')", routeName, routePath, seg)
				}
			}

			actualRoutePath = path.Join(actualRoutePath, param)
		} else {
			actualRoutePath = path.Join(actualRoutePath, seg)
		}
	}

	return actualRoutePath, constraints, nil
}

// Return values are -
// 1. path param
// 2. param constraint
// 3. is constraint exists
// 4. is constraint valid
func parameterConstraint(pathSeg string) (string, string, bool, bool) {
	sidx := strings.IndexByte(pathSeg, ruleStartByte)
	eidx := strings.IndexByte(pathSeg, ruleEndByte)

	// Validation rule exists but invalid
	if (sidx == -1 && eidx > 0) || (sidx >= 0 && eidx == -1) {
		return "", "", true, false
	}

	constraint := strings.TrimSpace(pathSeg[sidx+1 : eidx])
	return strings.TrimSpace(pathSeg[:sidx]),
		constraint,
		true,
		len(constraint) > 0
}
