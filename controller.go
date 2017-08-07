// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path"
	"reflect"
	"strings"

	"aahframework.org/essentials.v0"
	"aahframework.org/router.v0-unstable"
)

const (
	controllerNameSuffix    = "Controller"
	controllerNameSuffixLen = len(controllerNameSuffix)

	// Interceptor Action Name
	incpBeforeActionName  = "Before"
	incpAfterActionName   = "After"
	incpPanicActionName   = "Panic"
	incpFinallyActionName = "Finally"
)

var (
	cRegistry = make(controllerRegistry)
	emptyArg  = make([]reflect.Value, 0)
)

type (
	// ControllerInfo holds all application controller
	controllerRegistry map[string]*controllerInfo

	// ControllerInfo holds information of single controller information.
	controllerInfo struct {
		Type            reflect.Type
		Namespace       string
		Methods         map[string]*MethodInfo
		EmbeddedIndexes [][]int
	}

	// MethodInfo holds information of single method information in the controller.
	MethodInfo struct {
		Name       string
		Parameters []*ParameterInfo
	}

	// ParameterInfo holds information of single parameter in the method.
	ParameterInfo struct {
		Name string
		Type reflect.Type
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//___________________________________

// AddController method adds given controller into controller registory.
// with "dereferenced" a.k.a "indirecting".
func AddController(c interface{}, methods []*MethodInfo) {
	cType := actualType(c)

	methodMapping := map[string]*MethodInfo{}
	for _, method := range methods {
		for _, param := range method.Parameters {
			param.Type = actualType(param.Type)
		}
		methodMapping[strings.ToLower(method.Name)] = method
	}

	key, namespace := createRegistryKeyAndNamespace(cType)
	cRegistry[key] = &controllerInfo{
		Type:            cType,
		Namespace:       namespace,
		Methods:         methodMapping,
		EmbeddedIndexes: findEmbeddedContext(cType),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ControllerRegistry methods
//___________________________________

// Lookup method returns `controllerInfo` if given route controller and
// action exists in the controller registory.
func (cr controllerRegistry) Lookup(route *router.Route) *controllerInfo {
	if ci, found := cr[strings.ToLower(route.Controller)]; found {
		return ci
	}

	for _, ci := range cr {
		// match exact character case
		if strings.HasSuffix(route.Controller, ci.Name()) {
			return ci
		}
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ControllerInfo methods
//___________________________________

// Name method returns name of the controller.
func (ci *controllerInfo) Name() string {
	return ci.Type.Name()
}

// FindMethod method returns the `aah.MethodInfo` by given name
// (case insensitive) otherwise nil.
func (ci *controllerInfo) FindMethod(name string) *MethodInfo {
	if method, found := ci.Methods[strings.ToLower(name)]; found {
		return method
	}

	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func actualType(v interface{}) reflect.Type {
	vt := reflect.TypeOf(v)
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}

	return vt
}

// createRegistryKeyAndNamespace method creates the controller registry key.
func createRegistryKeyAndNamespace(cType reflect.Type) (string, string) {
	namespace := cType.PkgPath()
	if idx := strings.Index(namespace, "controllers"); idx > -1 {
		namespace = namespace[idx+11:]
	}

	if ess.IsStrEmpty(namespace) {
		return strings.ToLower(cType.Name()), ""
	}

	return strings.ToLower(path.Join(namespace[1:], cType.Name())), namespace[1:]
}
