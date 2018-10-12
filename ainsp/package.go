// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ainsp

import (
	"go/ast"
	"go/build"
	"go/token"
	"path/filepath"

	"aahframe.work/essentials"
	"aahframe.work/log"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// PackageInfo struct and its methods
//______________________________________________________________________________

// PackageInfo holds the single paackge information.
type packageInfo struct {
	ImportPath string
	FilePath   string
	Files      []string
	Types      map[string]*typeInfo
	Fset       *token.FileSet
	Pkg        *ast.Package
}

// Name method return package name
func (p *packageInfo) Name() string {
	return filepath.Base(p.ImportPath)
}

func (p *packageInfo) processTypes(decl ast.Decl, imports map[string]string) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || !isTypeTok(genDecl) || len(genDecl.Specs) == 0 {
		return
	}

	spec := genDecl.Specs[0].(*ast.TypeSpec)
	st, ok := spec.Type.(*ast.StructType)
	if !ok {
		// Not a struct type
		return
	}

	typeName := spec.Name.Name
	ty := &typeInfo{
		Name:          typeName,
		ImportPath:    filepath.ToSlash(p.ImportPath),
		Methods:       make([]*methodInfo, 0),
		EmbeddedTypes: make([]*typeInfo, 0),
	}

	for _, field := range st.Fields.List {
		// If field.Names is set, it's not an embedded type.
		if field.Names != nil && len(field.Names) > 0 {
			continue
		}

		fPkgName, fTypeName := parseStructFieldExpr(field.Type)
		if ess.IsStrEmpty(fTypeName) {
			continue
		}

		// Find the import path for embedded type. If it was referenced without
		// a package name, use the current package import path otherwise
		// get the import path by package name.
		var eTypeImportPath string
		if ess.IsStrEmpty(fPkgName) {
			eTypeImportPath = ty.ImportPath
		} else {
			var found bool
			if eTypeImportPath, found = imports[fPkgName]; !found {
				log.Errorf("AST: Unable to find import path for %s.%s", fPkgName, fTypeName)
				continue
			}
		}

		ty.EmbeddedTypes = append(ty.EmbeddedTypes, &typeInfo{Name: fTypeName, ImportPath: eTypeImportPath})
	}

	p.Types[typeName] = ty
}

func (p *packageInfo) processImports(decl ast.Decl, imports map[string]string) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || !isImportTok(genDecl) {
		return
	}

	for _, dspec := range genDecl.Specs {
		spec := dspec.(*ast.ImportSpec)
		var pkgAlias string
		if spec.Name != nil {
			if spec.Name.Name == "_" {
				continue
			}

			pkgAlias = spec.Name.Name
		}

		importPath := spec.Path.Value[1 : len(spec.Path.Value)-1]
		if ess.IsStrEmpty(pkgAlias) {
			if alias, found := buildImportCache[importPath]; found {
				pkgAlias = alias
			} else { // build cache
				pkg, err := build.Import(importPath, p.FilePath, 0)
				if err != nil {
					// log.Errorf("AST: Unable to find import path: %s", importPath)
					continue
				}
				pkgAlias = pkg.Name
				buildImportCache[importPath] = pkg.Name
			}
		}

		imports[pkgAlias] = importPath
	}
}
