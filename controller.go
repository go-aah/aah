// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"reflect"
	"strings"

	"aahframework.org/aah/ahttp"
)

var (
	controllerRegistry   = map[string]*controllerInfo{}
	aahControllerPtrType = reflect.TypeOf((*Controller)(nil))
)

type (
	// Controller type for aah framework, gets embedded in application controller.
	Controller struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		res ahttp.ResponseWriter
	}

	// ControllerInfo holds information of single controller information.
	controllerInfo struct {
		Name            string
		Type            reflect.Type
		Methods         []*MethodInfo
		EmbeddedIndexes [][]int
		lowerName       string
	}

	// MethodInfo holds information of single method information in the controller.
	MethodInfo struct {
		Name       string
		Parameters []*ParameterInfo
		lowerName  string
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
	fmt.Println(c, methods)

	ct := actualType(c)
	fmt.Println("ct name:", ct.Name())

	indexes := findEmbeddedController(ct)
	fmt.Println("indexes:", indexes)

	_ = controllerRegistry
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// ControllerInfo methods
//___________________________________

// FindMethod method returns the `aah.MethodInfo` by given name
// (case insensitive) otherwise nil.
func (ci *controllerInfo) FindMethod(name string) *MethodInfo {
	name = strings.ToLower(name)
	for _, m := range ci.Methods {
		if m.lowerName == name {
			return m
		}
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
			if field.Type == aahControllerPtrType {
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
