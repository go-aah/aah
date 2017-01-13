// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"path/filepath"

	"aahframework.org/aah/ahttp"
	"aahframework.org/essentials"
)

type (
	// Controller type for aah framework, gets embedded in application controller.
	Controller struct {
		// Req is HTTP request instance
		Req *ahttp.Request

		res ahttp.ResponseWriter
	}

	// TypeInfo holds the information about Controller Name, Methods,
	// Embedded types etc.
	TypeInfo struct {
		Name          string
		ImportPath    string
		Methods       []*MethodInfo
		EmbeddedTypes []*TypeInfo
	}

	// MethodInfo holds the information of single method and it's Parameters.
	MethodInfo struct {
		Name       string
		StructName string
		Parameters []*ParameterInfo
	}

	// ParameterInfo holds the information of single Parameter in the method.
	ParameterInfo struct {
		Name       string
		ImportPath string
		Type       *TypeExpr
	}

	// TypeExpr holds the information of single parameter data type.
	TypeExpr struct {
		Expr         string
		IsBuiltIn    bool
		PackageName  string
		ImportPath   string
		PackageIndex uint8
		Valid        bool
	}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// TypeInfo methods
//___________________________________

// FullyQualifiedName method returns the fully qualified type name.
func (t *TypeInfo) FullyQualifiedName() string {
	return fmt.Sprintf("%s.%s", t.ImportPath, t.Name)
}

// PackageName method returns types package name from import path.
func (t *TypeInfo) PackageName() string {
	return filepath.Base(t.ImportPath)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// TypeExpr methods
//___________________________________

// Name method returns type name for expression.
func (te *TypeExpr) Name() string {
	if te.IsBuiltIn || ess.IsStrEmpty(te.PackageName) {
		return te.Expr
	}

	return fmt.Sprintf("%s%s.%s", te.Expr[:te.PackageIndex], te.PackageName, te.Expr[te.PackageIndex:])
}
