// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
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
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"aahframe.work/essentials"
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
func Inspect(dir, importPath string, excludes ess.Excludes, registeredActions map[string]map[string]uint8) (*Program, []error) {
	prg := &Program{
		Path:              dir,
		Packages:          []*packageInfo{},
		RegisteredActions: registeredActions,
	}

	if err := validateInput(dir); err != nil {
		return prg, append([]error{}, err)
	}

	var (
		pkgs map[string]*ast.Package
		errs []error
	)

	err := ess.Walk(dir, func(srcPath string, info os.FileInfo, err error) error {
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
			pkgImportPath := filepath.ToSlash(srcPath)
			i := strings.LastIndex(pkgImportPath, "app/")
			pkgImportPath = filepath.ToSlash(path.Clean(path.Join(importPath, srcPath[i:])))

			pkg.Fset = pfset
			pkg.FilePath = srcPath
			pkg.ImportPath = pkgImportPath
			prg.Packages = append(prg.Packages, pkg)
		}

		return nil
	})

	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		prg.process()
	}

	return prg, errs
}

// FindFieldIndexes method does breadth-first search on struct
// anonymous field to find given type `struct` discover index positions.
//
// For e.g.: `aah.Context`, `ws.Context`, etc.
func FindFieldIndexes(targetTyp reflect.Type, searchTyp reflect.Type) [][]int {
	var indexes [][]int
	type nodeType struct {
		val   reflect.Value
		index []int
	}

	queue := []nodeType{{reflect.New(targetTyp), []int{}}}

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

			// If it's a search type then record the field index and move on
			if field.Type == searchTyp {
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
