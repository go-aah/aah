// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/router"
)

var (
	cRegistry   = controllerRegistry{}
	cPkgName    = "controllers"
	cPkgNameLen = len(cPkgName)
	cPtrType    = reflect.TypeOf((*Controller)(nil))

	errTargetNotFound = errors.New("target not found")
)

type (
	// Controller type for aah framework, gets embedded in application controller.
	Controller struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		controller string
		action     string
		target     interface{}
		res        ahttp.ResponseWriter
	}

	// ControllerInfo holds all application controller
	controllerRegistry map[string]*controllerInfo

	// ControllerInfo holds information of single controller information.
	controllerInfo struct {
		Type            reflect.Type
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
// Global methods
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

	cRegistry[strings.ToLower(cType.Name())] = &controllerInfo{
		Type:            cType,
		Methods:         methodMapping,
		EmbeddedIndexes: findEmbeddedController(cType),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Controller methods
//___________________________________

// reset method resets controller instance for reuse.
func (c *Controller) reset() {
	c.Req = nil
	c.res = nil
	c.target = nil
}

// setTarget method sets contoller, action, embedded controller into
// controller
func (c *Controller) setTarget(route *router.Route) error {
	controller := cRegistry.Lookup(route)
	if controller == nil {
		return errTargetNotFound
	}

	action := controller.FindMethod(route.Action)
	if action == nil {
		return errTargetNotFound
	}

	c.controller = controller.Name()
	c.action = action.Name

	targetPtr := reflect.New(controller.Type)
	target := targetPtr.Elem()
	cv := reflect.ValueOf(c)
	for _, index := range controller.EmbeddedIndexes {
		target.FieldByIndex(index).Set(cv)
	}

	c.target = targetPtr
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ControllerRegistry methods
//___________________________________

// Lookup method retuns `controllerInfo` if given route controller and
// action exists in the controller registory.
func (cr controllerRegistry) Lookup(route *router.Route) *controllerInfo {
	if ci, found := cr[strings.ToLower(route.Controller)]; found {
		return ci
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

// findEmbeddedController method does breadth-first search on struct anonymous
// field to find `aah.Controller` index positions.
func findEmbeddedController(controllerType reflect.Type) [][]int {
	var indexes [][]int
	type nodeType struct {
		val   reflect.Value
		index []int
	}

	queue := []nodeType{{reflect.New(controllerType), []int{}}}

	for len(queue) > 0 {
		var (
			node     = queue[0]
			elem     = node.val
			elemType = elem.Type()
		)

		if elemType.Kind() == reflect.Ptr {
			elem = elem.Elem()
			elemType = elem.Type()
		}

		queue = queue[1:]
		if elemType.Kind() != reflect.Struct {
			continue
		}

		for i := 0; i < elem.NumField(); i++ {
			// skip non-anonymous fields
			field := elemType.Field(i)
			if !field.Anonymous {
				continue
			}

			// If it's a `aah.Controller`, record the field indexes
			if field.Type == cPtrType {
				indexes = append(indexes, append(node.index, i))
				continue
			}

			fieldValue := elem.Field(i)
			queue = append(queue,
				nodeType{fieldValue, append(append([]int{}, node.index...), i)})
		}
	}

	return indexes
}

func newCPool() sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			return &Controller{}
		},
	}
}
