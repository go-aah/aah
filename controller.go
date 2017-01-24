// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"reflect"
	"strings"

	"aahframework.org/aah/ahttp"
	"aahframework.org/aah/reply"
	"aahframework.org/aah/router"
)

const (
	// Interceptor Action Name
	incpBeforeActionName  = "Before"
	incpAfterActionName   = "After"
	incpPanicActionName   = "Panic"
	incpFinallyActionName = "Finally"
)

var (
	cRegistry = make(controllerRegistry)
	cPtrType  = reflect.TypeOf((*Controller)(nil))
	emptyArg  = make([]reflect.Value, 0)

	errTargetNotFound = errors.New("target not found")
)

type (
	// Controller type for aah framework, gets embedded in application controller.
	Controller struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		// Res is HTTP response writer. Not recommended to use this directly.
		// Instead use `Reply()` builder for composing response.
		Res ahttp.ResponseWriter

		controller string
		action     *MethodInfo
		pathParams *router.PathParams
		target     interface{}
		reply      *reply.Reply
		viewArgs   map[string]interface{}
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

// Reply method gives you control and convenient way to write
// a response effectively.
func (c *Controller) Reply() *reply.Reply {
	return c.reply
}

// ViewArgs method returns aah framework and request related info that can be
// used in template or view rendering, etc.
func (c *Controller) ViewArgs() map[string]interface{} {
	return c.viewArgs
}

// GetPathParam method returns the URL path param value.
// 		Example:
// 			Mapping: /users/:userId
// 			URL: /users/1000001
//
// 		userId := c.GetPathParam("userId")
// 		userId == 1000001
//
func (c *Controller) GetPathParam(name string) string {
	return c.pathParams.Get(name)
}

// AddViewArg method adds given key and value into `viewArgs`. These view args
// values accessible on templates. Chained call is possible.
func (c *Controller) AddViewArg(key string, value interface{}) *Controller {
	c.viewArgs[key] = value
	return c
}

// Reset method resets controller instance for reuse.
func (c *Controller) Reset() {
	c.Req = nil
	c.Res = nil
	c.target = nil
	c.controller = ""
	c.action = nil
	c.pathParams = nil
	c.reply = nil
	c.viewArgs = nil
}

// setTarget method sets contoller, action, embedded controller into
// controller
func (c *Controller) setTarget(route *router.Route) error {
	controller := cRegistry.Lookup(route)
	if controller == nil {
		return errTargetNotFound
	}

	c.controller = controller.Name()
	c.action = controller.FindMethod(route.Action)
	if c.action == nil {
		return errTargetNotFound
	}

	targetPtr := reflect.New(controller.Type)
	target := targetPtr.Elem()
	cv := reflect.ValueOf(c)
	for _, index := range controller.EmbeddedIndexes {
		target.FieldByIndex(index).Set(cv)
	}

	c.target = targetPtr.Interface()
	return nil
}

// close method tries to close if `io.Closer` interface satisfies.
func (c *Controller) close() {
	c.Res.(*ahttp.Response).Close()
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
