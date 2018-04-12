// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path"
	"reflect"
	"strings"

	"aahframework.org/router.v0"
)

const (
	// Interceptor Action Name
	incpBeforeActionName  = "Before"
	incpAfterActionName   = "After"
	incpPanicActionName   = "Panic"
	incpFinallyActionName = "Finally"
)

var (
	emptyArg = make([]reflect.Value, 0)
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) AddController(c interface{}, methods []*MethodInfo) {
	cType := actualType(c)

	// Method Info
	methodMapping := map[string]*MethodInfo{}
	for _, method := range methods {
		for _, param := range method.Parameters {
			param.Type = param.Type.Elem()
			param.kind = kind(param.Type)
		}
		methodMapping[strings.ToLower(method.Name)] = method
	}

	// Controller Info
	key, namespace := createRegistryKeyAndNamespace(cType)
	controllerInfo := &controllerInfo{
		Name:            cType.Name(),
		Type:            cType,
		Namespace:       namespace,
		Methods:         methodMapping,
		EmbeddedIndexes: findEmbeddedContext(cType),
	}

	// Fully qualified name
	controllerInfo.FqName = path.Join(controllerInfo.Namespace, controllerInfo.Name)

	// No suffix name which maps to controller name towards directory
	// name by convention.
	// For e.g.: UserController, User ==> User
	noSuffixName := controllerInfo.Name
	if strings.HasSuffix(noSuffixName, "Controller") {
		noSuffixName = noSuffixName[:len(noSuffixName)-len("Controller")]
	}
	controllerInfo.NoSuffixName = noSuffixName

	a.engine.cregistry.Add(key, controllerInfo)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ControllerRegistry
//______________________________________________________________________________

// ControllerRegistry struct holds all application controller and related methods.
type controllerRegistry map[string]*controllerInfo

func (cr controllerRegistry) Add(key string, ci *controllerInfo) {
	cr[key] = ci
}

// Lookup method returns `controllerInfo` if given route controller and
// action exists in the controller registory.
func (cr controllerRegistry) Lookup(route *router.Route) *controllerInfo {
	if ci, found := cr[strings.ToLower(route.Controller)]; found {
		return ci
	}

	for _, ci := range cr {
		// match exact character case
		if strings.HasSuffix(route.Controller, ci.Name) {
			return ci
		}
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ControllerInfo
//______________________________________________________________________________

// ControllerInfo holds information of single controller information.
type controllerInfo struct {
	Name            string
	FqName          string
	NoSuffixName    string
	Type            reflect.Type
	Namespace       string
	Methods         map[string]*MethodInfo
	EmbeddedIndexes [][]int
}

// MethodInfo holds information of single method information in the controller.
type MethodInfo struct {
	Name       string
	Parameters []*ParameterInfo
}

// ParameterInfo holds information of single parameter in the method.
type ParameterInfo struct {
	Name string
	Type reflect.Type
	kind reflect.Kind
}

// Lookup method returns the `aah.MethodInfo` by given name
// (case insensitive) otherwise nil.
func (ci *controllerInfo) Lookup(name string) *MethodInfo {
	if method, found := ci.Methods[strings.ToLower(name)]; found {
		return method
	}

	return nil
}
