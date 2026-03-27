package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func noDirectOSExitAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "noosexit",
		Doc:  "forbids direct os.Exit calls inside main.main",
		Run:  runNoDirectOSExit,
	}
}

func runNoDirectOSExit(pass *analysis.Pass) (any, error) {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				used, ok := pass.TypesInfo.Uses[sel.Sel]
				if !ok {
					return true
				}

				fnObj, ok := used.(*types.Func)
				if !ok || fnObj.Pkg() == nil {
					return true
				}
				if fnObj.Pkg().Path() != "os" || fnObj.Name() != "Exit" {
					return true
				}

				pass.Reportf(call.Pos(), "direct call to os.Exit in main.main is forbidden")
				return true
			})
		}
	}

	return nil, nil
}
