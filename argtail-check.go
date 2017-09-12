// argtail-check modifies Go code to check for trailing args that otherwise would have been ignored.
// If the code calls flag.Args() or flag.NArg(), then it's already looking at the extra args, so don't modify those.
//
// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
	"strings"
)

var (
	errNoFlagParseCalls = fmt.Errorf("no flag parse calls found")
	errFuncNotFound     = fmt.Errorf("function to modify not found")
	errAlreadyChecking  = fmt.Errorf("code already checking traling args")
)

// hasCall checks if there is a mod.Sym() call anywhere in the ast.
func hasCall(f ast.Node, mod, sym string) bool {
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if m, ok := x.X.(*ast.Ident); ok {
				if m.Name == mod && x.Sel.Name == sym {
					found = true
				}
			}
		}
		return !found
	})
	return found
}

// modifyFunc modifies the function to add the log.Fatalf() check after "flag.Parse()".
func modifyFunc(n ast.Node) {
	block := n.(*ast.BlockStmt)
	i := -1
	for cur, st := range block.List {
		if expr, ok := st.(*ast.ExprStmt); ok {
			if call, ok := expr.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if m, ok := sel.X.(*ast.Ident); ok {
						if m.Name == "flag" && sel.Sel.Name == "Parse" {
							i = cur + 1
							break
						}
					}
				}
			}
		}
	}
	check := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("flag"),
					Sel: ast.NewIdent("NArg"),
				},
			},
			Y: &ast.BasicLit{
				Value: "0",
			},
			Op: token.NEQ,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("log"),
							Sel: ast.NewIdent("Fatalf"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: "\"Trailing args not expected: %q\"",
							},
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   ast.NewIdent("flag"),
									Sel: ast.NewIdent("Args"),
								},
							},
						},
					},
				},
			},
		},
	}

	list := make([]ast.Stmt, len(block.List)+1)
	copy(list, block.List[:i])
	list[i] = check
	copy(list[i+1:], block.List[i:])
	block.List = list
}

// findAndModifyFunc finds a function and calls modifyFunc on it.
// Used to find 'main' to then add the flag check.
func findAndModifyFunc(n ast.Node, f string) error {
	found := false
	ast.Inspect(n, func(inner ast.Node) bool {
		switch x := inner.(type) {
		case *ast.FuncDecl:
			if x.Name.Name == f {
				modifyFunc(x.Body)
				found = true
			}
		}
		return !found
	})
	if !found {
		return errFuncNotFound
	}
	return nil
}

// findAndModifyImports adds flag and log imports.
func findAndModifyImports(n ast.Node) {
	imports := make(map[string]bool)
	ast.Inspect(n, func(inner ast.Node) bool {
		switch x := inner.(type) {
		case *ast.File:
			for _, i := range x.Imports {
				if i.Name != nil {
					imports[i.Name.Name] = true
				} else {
					imports[strings.TrimSuffix(path.Base(i.Path.Value), `"`)] = true
				}
			}
		case *ast.GenDecl:
			if x.Tok == token.IMPORT {
				for _, imp := range []string{
					"flag",
					"log",
				} {
					if imports[path.Base(imp)] {
						continue
					}
					x.Specs = append(x.Specs, &ast.ImportSpec{
						Path: &ast.BasicLit{
							Value: `"` + imp + `"`,
						},
					})
				}
			}
			return false
		}
		return true
	})
}

// fix adds the check to a file (file name not used yet, but could be helpful for error messages).
func fix(fn, s string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fn, s, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse %q: %v", fn, err)
	}

	if !hasCall(f, "flag", "Parse") {
		// Program doesn't flag.Parse(). Nothing to do.
		return "", errNoFlagParseCalls
	}

	if hasCall(f, "flag", "Args") || hasCall(f, "flag", "NArg") {
		// Program checks for extra args already. Nothing to do.
		return "", errAlreadyChecking
	}

	if err := findAndModifyFunc(f, "main"); err != nil {
		return "", err
	}
	findAndModifyImports(f)
	var out bytes.Buffer
	format.Node(&out, fset, f)
	return out.String(), nil
}

func main() {
	flag.Parse()
	ctx := context.Background()
	for _, fn := range flag.Args() {
		s, err := ioutil.ReadFile(fn)
		if err != nil {
			log.Fatalf("Failed to read %q: %v", fn, err)
		}

		so, err := fix(fn, string(s))
		switch err {
		case errNoFlagParseCalls, errFuncNotFound, errAlreadyChecking:
			fmt.Printf("Nothing to do: %v", err)
			continue
		case nil:
			if err := exec.CommandContext(ctx, "g4", "edit", fn).Run(); err != nil {
				log.Fatalf("Failed to g4 edit %q: %v", fn, err)
			}
			if err := ioutil.WriteFile(fn, []byte(so), 0); err != nil {
				log.Fatalf("Failed to write back file: %v", err)
			}
		default:
			log.Fatalf("Failed with %q: %v", fn, err)
		}
	}
}
