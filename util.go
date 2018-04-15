// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/ainsp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ainsp

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

func validateInput(path string) error {
	if ess.IsStrEmpty(path) {
		return errors.New("path is required input")
	}

	if !ess.IsFileExists(path) {
		return fmt.Errorf("path is does not exists: %s", path)
	}

	return nil
}

func validateAndGetPkg(pkgs map[string]*ast.Package, path string) (*packageInfo, error) {
	pkgCnt := len(pkgs)

	// no source code found in the directory
	if pkgCnt == 0 {
		return nil, nil
	}

	// not permitted by Go lang spec
	if pkgCnt > 1 {
		var names []string
		for k := range pkgs {
			names = append(names, k)
		}
		return nil, fmt.Errorf("more than one package name [%s] found in single"+
			" directory: %s", strings.Join(names, ", "), path)
	}

	pkg := &packageInfo{}
	for _, v := range pkgs {
		pkg.Pkg = v
	}

	return pkg, nil
}

func isImportTok(decl *ast.GenDecl) bool {
	return token.IMPORT == decl.Tok
}

func isTypeTok(decl *ast.GenDecl) bool {
	return token.TYPE == decl.Tok
}

func stripGoPath(pkgFilePath string) string {
	idx := strings.Index(pkgFilePath, "src")
	return filepath.Clean(pkgFilePath[idx+4:])
}

func isPkgAliasExists(importPaths map[string]string, pkgAlias string) bool {
	_, found := importPaths[pkgAlias]
	return found
}

func processMethods(pkg *packageInfo, routeMethods map[string]map[string]uint8, decl ast.Decl, imports map[string]string) {
	fn, ok := decl.(*ast.FuncDecl)

	// Do not process if these met:
	// 		1. does not have receiver, it means package function/method
	// 		2. method is not exported
	// 		3. method returns result
	if !ok || fn.Recv == nil || !fn.Name.IsExported() ||
		fn.Type.Results != nil {
		return
	}

	actionName := fn.Name.Name
	if isInterceptorActionName(actionName) {
		return
	}

	controllerName := getName(fn.Recv.List[0].Type)
	method := &methodInfo{Name: actionName, StructName: controllerName, Parameters: []*parameterInfo{}}

	// processed so set to level 2, used to display unimplemented action details
	// TODO for controller check too
	for k, v := range routeMethods {
		if strings.HasSuffix(k, controllerName) {
			if _, found := v[actionName]; found {
				v[actionName] = 2
			}
		}
	}

	// processing method parameters
	for _, field := range fn.Type.Params.List {
		for _, fieldName := range field.Names {
			te, err := parseParamFieldExpr(pkg.Name(), field.Type)
			if err != nil {
				log.Errorf("AST: %s, please fix the parameter '%s' on action '%s.%s'; "+
					"otherwise your action may not work properly", err, fieldName.Name, controllerName, actionName)
				continue
			}

			var importPath string
			if !ess.IsStrEmpty(te.PackageName) {
				var found bool
				if importPath, found = imports[te.PackageName]; !found {
					importPath = pkg.ImportPath
				}
			}

			method.Parameters = append(method.Parameters, &parameterInfo{
				Name:       fieldName.Name,
				ImportPath: importPath,
				Type:       te,
			})
		}
	}

	if ty := pkg.Types[controllerName]; ty == nil {
		pos := pkg.Fset.Position(decl.Pos())
		filename := stripGoPath(pos.Filename)
		log.Errorf("AST: Method '%s' has incorrect struct recevier '%s' on file [%s] at line #%d",
			actionName, controllerName, filename, pos.Line)
	} else {
		ty.Methods = append(ty.Methods, method)
	}
}

func isInterceptorActionName(actionName string) bool {
	return (strings.HasPrefix(actionName, "Before") || strings.HasPrefix(actionName, "After") ||
		strings.HasPrefix(actionName, "Panic") || strings.HasPrefix(actionName, "Finally"))
}

func getName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return getName(t.X)
	case *ast.StarExpr:
		return getName(t.X)
	default:
		return ""
	}
}

func isBuiltInDataType(typeName string) bool {
	_, found := builtInDataTypes[typeName]
	return found
}

// parseStructFieldExpr method to find a direct "embedded|sub-type".
// Struct ast.Field as follows:
//   Ident { "type-name" } e.g. UserController
//   SelectorExpr { "package-name", "type-name" } e.g. aah.Controller
//   StarExpr { "*", "package-name", "type-name"} e.g. *aah.Controller
func parseStructFieldExpr(fieldType ast.Expr) (string, string) {
	for {
		if starExpr, ok := fieldType.(*ast.StarExpr); ok {
			fieldType = starExpr.X
			continue
		}
		break
	}

	// type it's in the same package, it's an ast.Ident.
	if ident, ok := fieldType.(*ast.Ident); ok {
		return "", ident.Name
	}

	// type it's in the different package, it's an ast.SelectorExpr.
	if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
		if pkgIdent, ok := selectorExpr.X.(*ast.Ident); ok {
			return pkgIdent.Name, selectorExpr.Sel.Name
		}
	}

	return "", ""
}

func parseParamFieldExpr(pkgName string, expr ast.Expr) (*typeExpr, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		if isBuiltInDataType(t.Name) {
			return &typeExpr{Expr: t.Name, IsBuiltIn: true}, nil
		}
		return &typeExpr{Expr: t.Name, PackageName: pkgName}, nil
	case *ast.SelectorExpr:
		e, err := parseParamFieldExpr(pkgName, t.X)
		return &typeExpr{Expr: t.Sel.Name, PackageName: e.Expr}, err
	case *ast.StarExpr:
		e, err := parseParamFieldExpr(pkgName, t.X)
		return &typeExpr{Expr: "*" + e.Expr, PackageName: e.PackageName, PackageIndex: e.PackageIndex + uint8(1)}, err
	case *ast.ArrayType:
		e, err := parseParamFieldExpr(pkgName, t.Elt)
		return &typeExpr{Expr: "[]" + e.Expr, PackageName: e.PackageName, PackageIndex: e.PackageIndex + uint8(2)}, err
	case *ast.Ellipsis:
		e, err := parseParamFieldExpr(pkgName, t.Elt)
		return &typeExpr{Expr: "[]" + e.Expr, PackageName: e.PackageName, PackageIndex: e.PackageIndex + uint8(2)}, err
	case *ast.InterfaceType:
		return nil, errInterfaceActionParam
	case *ast.MapType:
		return nil, errMapActionParam
	}

	return nil, errInvalidActionParam
}
