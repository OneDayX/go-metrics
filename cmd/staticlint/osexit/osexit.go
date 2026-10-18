// Package osexit defines an analyzer that forbids direct calls to os.Exit in
// the main function of the main package.
//
// os.Exit stops the program at once: deferred functions do not run, so open
// files, database connections and buffered logs are not closed or flushed.
// Instead, main should call a function that returns an error and handle it
// in one place, for example with log.Fatal:
//
//	func main() {
//		if err := run(); err != nil {
//			log.Fatal(err)
//		}
//	}
package osexit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer reports direct calls to os.Exit in func main of package main.
var Analyzer = &analysis.Analyzer{
	Name: "osexit",
	Doc:  "forbid direct calls to os.Exit in the main function of the main package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		// go test generates a main package that calls os.Exit, skip it and
		// other generated files.
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if ok && isOsExit(pass, call) {
					pass.Reportf(call.Pos(), "direct call to os.Exit in the main function of the main package")
				}
				return true
			})
		}
	}

	return nil, nil
}

// isOsExit reports whether call calls os.Exit. The callee is found by type
// info, so imports like `sys "os"` are caught too.
func isOsExit(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn := typeutil.StaticCallee(pass.TypesInfo, call)
	return fn != nil && fn.Pkg() != nil && fn.Pkg().Path() == "os" && fn.Name() == "Exit"
}
