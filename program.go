// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/ainsp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ainsp

import (
	"fmt"
	"path/filepath"

	"aahframework.org/essentials.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Program struct and its methods
//______________________________________________________________________________

// Program holds all details loaded from the Go source code for given Import Path.
type Program struct {
	Path              string
	Packages          []*packageInfo
	RegisteredActions map[string]map[string]uint8
}

// Process method processes all packages in the program for `Type`,
// `Embedded Type`, `Method`, etc.
func (prg *Program) Process() {
	for _, pkgInfo := range prg.Packages {
		pkgInfo.Types = map[string]*typeInfo{}

		// Each source file
		for name, file := range pkgInfo.Pkg.Files {
			pkgInfo.Files = append(pkgInfo.Files, filepath.Base(name))
			fileImports := make(map[string]string)

			for _, decl := range file.Decls {
				// Processing imports
				pkgInfo.processImports(decl, fileImports)

				// Processing types
				pkgInfo.processTypes(decl, fileImports)

				// Processing methods
				processMethods(pkgInfo, prg.RegisteredActions, decl, fileImports)
			}
		}
	}
}

// FindTypeByEmbeddedType method returns all the typeInfo that has directly or
// indirectly embedded by given type name. Type name must be fully qualified
// type name. E.g.: aahframework.org/aah.Controller
func (prg *Program) FindTypeByEmbeddedType(qualifiedTypeName string) []*typeInfo {
	var (
		queue     = []string{qualifiedTypeName}
		processed []string
		result    []*typeInfo
	)

	for len(queue) > 0 {
		typeName := queue[0]
		queue = queue[1:]
		processed = append(processed, typeName)

		// search within all packages in the program
		for _, p := range prg.Packages {
			// search within all struct type in the package
			for _, t := range p.Types {
				// If this one has been processed or is already in queue, then move on.
				if ess.IsSliceContainsString(processed, t.FullyQualifiedName()) ||
					ess.IsSliceContainsString(queue, t.FullyQualifiedName()) {
					continue
				}

				// search through the embedded types to see if the current type is among them.
				for _, et := range t.EmbeddedTypes {
					// If so, add this type's FullyQualifiedName into queue,
					//  and it's typeInfo into result.
					if typeName == et.FullyQualifiedName() {
						queue = append(queue, t.FullyQualifiedName())
						result = append(result, t)
						break
					}
				}
			}
		}
	}

	return result
}

// CreateImportPaths method returns unique package alias with import path.
func (prg *Program) CreateImportPaths(types []*typeInfo, importPaths map[string]string) map[string]string {
	for _, t := range types {
		createAlias(t.PackageName(), t.ImportPath, importPaths)
		for _, m := range t.Methods {
			for _, p := range m.Parameters {
				if !p.Type.IsBuiltIn {
					createAlias(p.Type.PackageName, p.ImportPath, importPaths)
				}
			}
		}
	}

	return importPaths
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// TypeInfo struct and its methods
//______________________________________________________________________________

// TypeInfo holds the information about Controller Name, Methods,
// Embedded types etc.
type typeInfo struct {
	Name          string
	ImportPath    string
	Methods       []*methodInfo
	EmbeddedTypes []*typeInfo
}

// MethodInfo holds the information of single method and it's Parameters.
type methodInfo struct {
	Name       string
	StructName string
	Parameters []*parameterInfo
}

// ParameterInfo holds the information of single Parameter in the method.
type parameterInfo struct {
	Name       string
	ImportPath string
	Type       *typeExpr
}

// FullyQualifiedName method returns the fully qualified type name.
func (t *typeInfo) FullyQualifiedName() string {
	return fmt.Sprintf("%s.%s", t.ImportPath, t.Name)
}

// PackageName method returns types package name from import path.
func (t *typeInfo) PackageName() string {
	return filepath.Base(t.ImportPath)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// TypeExpr struct and its methods
//______________________________________________________________________________

// TypeExpr holds the information of single parameter data type.
type typeExpr struct {
	IsBuiltIn    bool
	Valid        bool
	PackageIndex uint8
	Expr         string
	PackageName  string
	ImportPath   string
}

// Name method returns type name for expression.
func (te *typeExpr) Name() string {
	if te.IsBuiltIn || ess.IsStrEmpty(te.PackageName) {
		return te.Expr
	}

	return fmt.Sprintf("%s%s.%s", te.Expr[:te.PackageIndex], te.PackageName, te.Expr[te.PackageIndex:])
}
