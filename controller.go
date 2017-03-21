// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"errors"
	"reflect"
	"strings"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/router.v0"
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
	cPtrType  = reflect.TypeOf((*Controller)(nil))
	emptyArg  = make([]reflect.Value, 0)

	errTargetNotFound = errors.New("target not found")
)

type (
	// Controller type for aah framework, gets embedded in application controller.
	Controller struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		// Res is HTTP response writer. It is recommended to use
		// `Reply()` builder for composing response.
		Res ahttp.ResponseWriter

		controller string
		action     *MethodInfo
		target     interface{}
		domain     *router.Domain
		reply      *Reply
		viewArgs   map[string]interface{}
		abort      bool
	}

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

	key := createRegistryKey(cType)
	cRegistry[key] = &controllerInfo{
		Type:            cType,
		Namespace:       ess.StripExt(key),
		Methods:         methodMapping,
		EmbeddedIndexes: findEmbeddedController(cType),
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Controller methods
//___________________________________

// Reply method gives you control and convenient way to write
// a response effectively.
func (c *Controller) Reply() *Reply {
	return c.reply
}

// ViewArgs method returns aah framework and request related info that can be
// used in template or view rendering, etc.
func (c *Controller) ViewArgs() map[string]interface{} {
	return c.viewArgs
}

// AddViewArg method adds given key and value into `viewArgs`. These view args
// values accessible on templates. Chained call is possible.
func (c *Controller) AddViewArg(key string, value interface{}) *Controller {
	c.viewArgs[key] = value
	return c
}

// ReverseURL method returns the URL for given route name and args.
// See `Domain.ReverseURL` for more information.
func (c *Controller) ReverseURL(routeName string, args ...interface{}) string {
	return createReverseURL(c.Req.Host, routeName, nil, args...)
}

// ReverseURLm method returns the URL for given route name and key-value paris.
// See `Domain.ReverseURLm` for more information.
func (c *Controller) ReverseURLm(routeName string, args map[string]interface{}) string {
	return createReverseURL(c.Req.Host, routeName, args)
}

// Msg method returns the i18n value for given key otherwise empty string returned.
func (c *Controller) Msg(key string, args ...interface{}) string {
	return AppI18n().Lookup(c.Req.Locale, key, args...)
}

// Msgl method returns the i18n value for given local and key otherwise
// empty string returned.
func (c *Controller) Msgl(locale *ahttp.Locale, key string, args ...interface{}) string {
	return AppI18n().Lookup(locale, key, args...)
}

// Abort method sets the abort to true. It means framework will not proceed with
// next middleware, next interceptor or action based on context it being used.
// Contexts: 1) If it's called in the middleware, then middleware chain stops;
// framework starts processing response. 2) If it's called in Before interceptor
// then Before<Action> interceptor, mapped <Action>, After<Action> interceptor and
// After interceptor will not execute; framework starts processing response.
// 3) If it's called in Mapped <Action> then After<Action> interceptor and
// After interceptor will not execute; framework starts processing response.
func (c *Controller) Abort() {
	c.abort = true
}

// Reset method resets controller instance for reuse.
func (c *Controller) Reset() {
	c.Req = nil
	c.Res = nil
	c.target = nil
	c.domain = nil
	c.controller = ""
	c.action = nil
	c.reply = nil
	c.viewArgs = nil
	c.abort = false
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Controller Unexported methods
//___________________________________

// createRegistryKey method creates the controller registry key.
func createRegistryKey(cType reflect.Type) string {
	namespace := cType.PkgPath()
	idx := strings.Index(namespace, "controllers")
	namespace = namespace[idx+11:]

	if ess.IsStrEmpty(namespace) {
		return strings.ToLower(cType.Name())
	}

	return strings.ToLower(namespace[1:] + "." + cType.Name())
}

// setTarget method sets contoller, action, embedded controller into
// controller.
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
