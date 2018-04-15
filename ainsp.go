// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// aahframework.org/ainsp source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Package ainsp is a Go ast library for aah framework, it does inspect and
// discovers the Go `struct` which embeds particular type `struct`.
//
// For e.g.: `aahframework.org/{aah.Context, ws.Context}`, etc.
package ainsp

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

var (
	buildImportCache = map[string]string{}

	// Reference: https://golang.org/pkg/builtin/
	builtInDataTypes = map[string]bool{
		"bool":       true,
		"byte":       true,
		"complex128": true,
		"complex64":  true,
		"error":      true,
		"float32":    true,
		"float64":    true,
		"int":        true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"int8":       true,
		"rune":       true,
		"string":     true,
		"uint":       true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uint8":      true,
		"uintptr":    true,
	}

	errInvalidActionParam   = errors.New("aah: invalid action parameter")
	errInterfaceActionParam = errors.New("aah: 'interface{}' is not supported in the action parameter")
	errMapActionParam       = errors.New("aah: 'map' is not supported in the action parameter")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// Inspect method processes the Go source code for the given directory and its
// sub-directories.
func Inspect(path string, excludes ess.Excludes, registeredActions map[string]map[string]uint8) (*program, []error) {
	if err := validateInput(path); err != nil {
		return nil, append([]error{}, err)
	}

	prg := &program{
		Path:              path,
		Packages:          []*packageInfo{},
		RegisteredActions: registeredActions,
	}

	var (
		pkgs map[string]*ast.Package
		errs []error
	)

	err := ess.Walk(path, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			errs = append(errs, err)
		}

		// Excludes
		if excludes.Match(filepath.Base(srcPath)) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if !info.IsDir() {
			return nil
		}

		if info.IsDir() && ess.IsDirEmpty(srcPath) {
			// skip directory if it's empty
			return filepath.SkipDir
		}

		pfset := token.NewFileSet()
		pkgs, err = parser.ParseDir(pfset, srcPath, func(f os.FileInfo) bool {
			return !f.IsDir() && !excludes.Match(f.Name())
		}, 0)

		if err != nil {
			if errList, ok := err.(scanner.ErrorList); ok {
				// TODO parsing error list
				fmt.Println(errList)
			}

			errs = append(errs, fmt.Errorf("error parsing dir[%s]: %s", srcPath, err))
			return nil
		}

		pkg, err := validateAndGetPkg(pkgs, srcPath)
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		if pkg != nil {
			pkg.Fset = pfset
			pkg.FilePath = srcPath
			pkg.ImportPath = stripGoPath(srcPath)
			prg.Packages = append(prg.Packages, pkg)
		}

		return nil
	})

	if err != nil {
		errs = append(errs, err)
	}

	return prg, errs
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Program struct and its methods
//______________________________________________________________________________

// Program holds all details loaded from the Go source code for given Path.
type program struct {
	Path              string
	Packages          []*packageInfo
	RegisteredActions map[string]map[string]uint8
}

// Process method processes all packages in the program for `Type`,
// `Embedded Type`, `Method`, etc.
func (prg *program) Process() {
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
func (prg *program) FindTypeByEmbeddedType(qualifiedTypeName string) []*typeInfo {
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
func (prg *program) CreateImportPaths(types []*typeInfo) map[string]string {
	importPaths := map[string]string{}
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

func createAlias(packageName, importPath string, importPaths map[string]string) {
	importPath = filepath.ToSlash(importPath)
	if _, found := importPaths[importPath]; !found {
		cnt := 0
		pkgAlias := packageName

		for isPkgAliasExists(importPaths, pkgAlias) {
			pkgAlias = fmt.Sprintf("%s%d", packageName, cnt)
			cnt++
		}

		if !ess.IsStrEmpty(pkgAlias) && !ess.IsStrEmpty(importPath) {
			importPaths[importPath] = pkgAlias
		}
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// PackageInfo struct and its methods
//______________________________________________________________________________

// PackageInfo holds the single paackge information.
type packageInfo struct {
	Fset       *token.FileSet
	Pkg        *ast.Package
	Types      map[string]*typeInfo
	ImportPath string
	FilePath   string
	Files      []string
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
					log.Errorf("AST: Unable to find import path: %s", importPath)
					continue
				}
				pkgAlias = pkg.Name
				buildImportCache[importPath] = pkg.Name
			}
		}

		imports[pkgAlias] = importPath
	}
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
	Expr         string
	IsBuiltIn    bool
	PackageName  string
	ImportPath   string
	PackageIndex uint8
	Valid        bool
}

// Name method returns type name for expression.
func (te *typeExpr) Name() string {
	if te.IsBuiltIn || ess.IsStrEmpty(te.PackageName) {
		return te.Expr
	}

	return fmt.Sprintf("%s%s.%s", te.Expr[:te.PackageIndex], te.PackageName, te.Expr[te.PackageIndex:])
}
