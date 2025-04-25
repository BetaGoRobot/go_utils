package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

type warmupCall struct {
	Expr     string
	Comments []string
}

type importLine struct {
	Alias string
	Path  string
}

type warmupTemplateData struct {
	Imports     []importLine
	WarmupCalls []warmupCall
}

type rawCall struct {
	Expr    string
	Comment string
}

func main() {
	dir := flag.String("dir", ".", "target directory to scan")
	flag.Parse()

	generate(dir)
}

func generate(dir *string) {
	rawCalls, importPaths, _ := scanGoFiles(*dir)
	imports := buildImportLines(importPaths, *dir)
	uniqueCalls := deduplicateCalls(rawCalls)
	generateWarmupCode(*dir, imports, uniqueCalls)
}

func scanGoFiles(dir string) ([]rawCall, map[string]string, map[string]string) {
	var rawCalls []rawCall
	importPaths := map[string]string{}
	importMap := map[string]string{}

	_ = filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
		if err != nil || filepath.Ext(path) != ".go" || filepath.Base(path) == "warmup_gen.go" {
			return nil
		}
		fileCalls, pkgImport, fileImportMap := analyzeFile(path, dir)
		rawCalls = append(rawCalls, fileCalls...)
		for k, v := range pkgImport {
			importPaths[k] = v
		}
		for k, v := range fileImportMap {
			importMap[k] = v
		}
		return nil
	})
	return rawCalls, importPaths, importMap
}

func analyzeFile(path, baseDir string) ([]rawCall, map[string]string, map[string]string) {
	var results []rawCall
	importPaths := map[string]string{}
	importMap := map[string]string{}

	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
	if err != nil || node.Name.Name == "main" {
		return nil, importPaths, importMap
	}

	pkgName := node.Name.Name
	relImportPath, _ := filepath.Rel(baseDir, filepath.Dir(path))
	importPaths[pkgName] = relImportPath

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			importMap[imp.Name.Name] = importPath
		} else {
			segments := strings.Split(importPath, "/")
			importMap[segments[len(segments)-1]] = importPath
		}
	}

	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}

		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok || !containsGetCurrentFunc(callExpr.Fun, importMap) {
				return true
			}

			funcPos := fs.Position(funcDecl.Pos())
			callPos := fs.Position(callExpr.Lparen)

			fullCall := buildFunctionCall(pkgName, funcDecl)
			relPath, _ := filepath.Rel(baseDir, funcPos.Filename)
			comment := fmt.Sprintf("%s:%d", relPath, callPos.Line)
			log.Printf("Found function: %s in file %s", fullCall, comment)
			log.Printf("  ↳ calls reflecting.GetCurrentFunc() at %s:%d", relPath, callPos.Line)
			results = append(results, rawCall{Expr: fullCall, Comment: comment})
			return true
		})
	}
	return results, importPaths, importMap
}

func buildFunctionCall(pkgName string, funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil {
		return fmt.Sprintf("%s.%s", pkgName, funcDecl.Name.Name)
	}
	var structName string
	var isPointer bool
	switch recvType := funcDecl.Recv.List[0].Type.(type) {
	case *ast.Ident:
		structName = recvType.Name
	case *ast.StarExpr:
		if ident, ok := recvType.X.(*ast.Ident); ok {
			structName = ident.Name
			isPointer = true
		}
	}
	if isPointer {
		return fmt.Sprintf("(*%s.%s).%s", pkgName, structName, funcDecl.Name.Name)
	}
	return fmt.Sprintf("%s.%s.%s", pkgName, structName, funcDecl.Name.Name)
}

func buildImportLines(importPaths map[string]string, baseDir string) []importLine {
	modulePrefix := getGoModModuleName(baseDir)
	var imports []importLine
	for alias, path := range importPaths {
		imports = append(imports, importLine{
			Alias: alias,
			Path:  fmt.Sprintf("%s/%s", modulePrefix, path),
		})
	}
	return imports
}

func deduplicateCalls(rawCalls []rawCall) []warmupCall {
	callMap := map[string][]string{}
	for _, c := range rawCalls {
		callMap[c.Expr] = append(callMap[c.Expr], c.Comment)
	}
	var result []warmupCall
	for expr, comments := range callMap {
		result = append(result, warmupCall{Expr: expr, Comments: comments})
	}
	return result
}

func generateWarmupCode(dir string, imports []importLine, calls []warmupCall) {
	// sort warmcalls & importLine
	slices.SortFunc(
		imports,
		func(a, b importLine) int {
			if a.Path < b.Path {
				return -1
			} else if a.Path == b.Path {
				return 0
			}
			return 1
		},
	)
	slices.SortFunc(
		calls,
		func(a, b warmupCall) int {
			if a.Expr < b.Expr {
				return -1
			} else if a.Expr == b.Expr {
				return 0
			}
			return 1
		},
	)
	tpl := template.Must(template.New("warmup").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(warmupTemplateText))

	var buf bytes.Buffer
	tplData := warmupTemplateData{Imports: imports, WarmupCalls: calls}
	if err := tpl.Execute(&buf, tplData); err != nil {
		log.Fatalf("template execution failed: %v", err)
	}

	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("failed to format code: %v", err)
	}

	outputFile := filepath.Join(dir, "warmup.gen.go")
	if err := os.WriteFile(outputFile, formattedCode, 0644); err != nil {
		log.Fatalf("failed to write file: %v", err)
	}

	if err := exec.Command("goimports", "-w", outputFile).Run(); err != nil {
		log.Fatalf("goimports failed: %v", err)
	}

	log.Printf("Generated file at: %s", outputFile)
}

func containsGetCurrentFunc(expr ast.Expr, importMap map[string]string) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok && e.Sel.Name == "GetCurrentFunc" {
			if path, exists := importMap[ident.Name]; exists && path == "github.com/BetaGoRobot/go_utils/reflecting" {
				return true
			}
		}
	case *ast.CallExpr:
		if fun, ok := e.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := fun.X.(*ast.Ident); ok && fun.Sel.Name == "GetCurrentFunc" {
				if path, exists := importMap[ident.Name]; exists && path == "github.com/BetaGoRobot/go_utils/reflecting" {
					return true
				}
			}
		}
	}
	return false
}

func getGoModModuleName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		log.Fatalf("failed to read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	log.Fatal("module name not found in go.mod")
	return ""
}
