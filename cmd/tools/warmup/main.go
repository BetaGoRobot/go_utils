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
	PackageName string
	Imports     []importLine
	WarmupCalls []warmupCall
}

type rawCall struct {
	Expr    string
	Comment string
}

type packageData struct {
	PackageName string
	Dir         string
	RawCalls    []rawCall
	ImportPaths map[string]string
}

func main() {
	dir := flag.String("dir", ".", "target directory to scan")
	flag.Parse()

	generate(dir)
}

func generate(dir *string) {
	pkgs := scanPackages(*dir)
	modulePrefix := getGoModModuleName(*dir)

	for _, pkg := range pkgs {
		if len(pkg.RawCalls) == 0 {
			continue
		}
		imports := buildImportLines(pkg.ImportPaths, modulePrefix)
		uniqueCalls := deduplicateCalls(pkg.RawCalls)
		generateWarmupCode(pkg, imports, uniqueCalls)
	}
}

func scanPackages(dir string) []packageData {
	pkgMap := map[string]*packageData{}

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || filepath.Ext(path) != ".go" || filepath.Base(path) == "warmup.gen.go" {
			return nil
		}

		fs := token.NewFileSet()
		node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
		if err != nil || node.Name.Name == "main" {
			return nil
		}

		pkgDir := filepath.Dir(path)
		pkgName := node.Name.Name

		pkg, ok := pkgMap[pkgDir]
		if !ok {
			pkg = &packageData{
				PackageName: pkgName,
				Dir:         pkgDir,
				ImportPaths: make(map[string]string),
			}
			pkgMap[pkgDir] = pkg
		}

		relImportPath, _ := filepath.Rel(dir, pkgDir)
		pkg.ImportPaths[pkgName] = relImportPath

		importMap := map[string]string{}
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

			// Skip generic functions
			if funcDecl.Type.TypeParams != nil {
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
				relPath, _ := filepath.Rel(dir, funcPos.Filename)
				comment := fmt.Sprintf("%s:%d", relPath, callPos.Line)

				log.Printf("Found function: %s in file %s", fullCall, comment)
				pkg.RawCalls = append(pkg.RawCalls, rawCall{Expr: fullCall, Comment: comment})
				return true
			})
		}

		return nil
	})

	var result []packageData
	for _, pkg := range pkgMap {
		result = append(result, *pkg)
	}
	return result
}

func buildFunctionCall(currentPkg string, funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil {
		return funcDecl.Name.Name
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
		return fmt.Sprintf("(*%s).%s", structName, funcDecl.Name.Name)
	}
	return fmt.Sprintf("%s.%s", structName, funcDecl.Name.Name)
}

func buildImportLines(importPaths map[string]string, modulePrefix string) []importLine {
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

func generateWarmupCode(pkg packageData, imports []importLine, calls []warmupCall) {
	slices.SortFunc(imports, func(a, b importLine) int {
		return strings.Compare(a.Path, b.Path)
	})
	slices.SortFunc(calls, func(a, b warmupCall) int {
		return strings.Compare(a.Expr, b.Expr)
	})
	if len(calls) == 0 {
		return
	}
	tpl := template.Must(template.New("warmup").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(warmupTemplateText))

	var buf bytes.Buffer
	tplData := warmupTemplateData{
		PackageName: pkg.PackageName,
		Imports:     imports,
		WarmupCalls: calls,
	}

	if err := tpl.Execute(&buf, tplData); err != nil {
		log.Fatalf("template execution failed: %v", err)
	}

	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("failed to format code: %v", err)
	}

	outputFile := filepath.Join(pkg.Dir, "warmup.gen.go")
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
