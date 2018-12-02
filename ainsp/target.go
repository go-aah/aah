// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ainsp

import (
	"path"
	"reflect"
	"strings"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Target Registry
//______________________________________________________________________________

// TargetRegistry struct holds registered information and provides lookup.
// Such as aah application controllers, websockets, etc.
type TargetRegistry struct {
	Registry   map[string]*Target
	SearchType reflect.Type
}

// Add method adds given target struct and its methods after processing.
func (tr *TargetRegistry) Add(t interface{}, methods []*Method) {
	ttyp := atype(t)

	// Method Info
	methodMapping := map[string]*Method{}
	for _, method := range methods {
		for _, param := range method.Parameters {
			param.Type = param.Type.Elem()
			param.Kind = kind(param.Type)
		}
		methodMapping[strings.ToLower(method.Name)] = method
	}

	// Create target details
	key, namespace := targetKeyAndNamespace(ttyp)
	target := &Target{
		Name:            ttyp.Name(),
		Type:            ttyp,
		Namespace:       namespace,
		Methods:         methodMapping,
		EmbeddedIndexes: FindFieldIndexes(ttyp, tr.SearchType),
	}

	// Fully qualified name
	target.FqName = path.Join(target.Namespace, target.Name)

	// No suffix name which maps to target name towards directory
	// name by convention.
	// For e.g.: UserController, User ==> User
	noSuffixName := target.Name
	if strings.HasSuffix(strings.ToLower(noSuffixName), "controller") {
		noSuffixName = noSuffixName[:len(noSuffixName)-len("Controller")]
	} else if strings.HasSuffix(strings.ToLower(noSuffixName), "websocket") {
		noSuffixName = noSuffixName[:len(noSuffixName)-len("websocket")]
	}
	target.NoSuffixName = noSuffixName

	// adding to registry
	tr.Registry[key] = target
}

// Lookup method returns `Target` info from registry for given `fqName`
// (fully qualified name) otherwise nil.
//
// It does exact match or exact suffix match.
func (tr *TargetRegistry) Lookup(fqName string) *Target {
	if t, found := tr.Registry[strings.ToLower(fqName)]; found {
		return t
	}

	for _, t := range tr.Registry {
		// match exact character case
		if strings.HasSuffix(fqName, t.Name) {
			return t
		}
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Target struct and related types
//______________________________________________________________________________

// Target struct holds info about targeted controller, websocket, etc.
type Target struct {
	Name            string
	FqName          string
	NoSuffixName    string
	Namespace       string
	Type            reflect.Type
	Methods         map[string]*Method
	EmbeddedIndexes [][]int
}

// Method holds single method information of target.
type Method struct {
	Name       string
	Parameters []*Parameter
}

// Parameter holds parameter information of method.
type Parameter struct {
	Name string
	Type reflect.Type
	Kind reflect.Kind
}

// Lookup method returns method info for given name (case insensitive)
// otherwise nil.
func (t *Target) Lookup(methodName string) *Method {
	if method, found := t.Methods[strings.ToLower(methodName)]; found {
		return method
	}
	return nil
}
