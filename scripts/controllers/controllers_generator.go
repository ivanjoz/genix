package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Output file relative to the project root.
const controllersOutputPath = "backend/exec/controllers.generated.go"

// Package the generated file lives in. Types defined in this package are referenced
// without a qualifier in the generated controller list.
const controllersOutputPackage = "app/exec"

// tableEntry describes one base struct (the X in db.TableStruct[XTable, X]) discovered
// during the AST scan. PackagePath is the full import path (e.g. "app/logistica/types").
type tableEntry struct {
	TypeName    string
	PackagePath string
}

func main() {
	projectRoot, err := findProjectRootDir()
	if err != nil {
		exitWith(err)
	}

	entries, err := scanTableStructs(filepath.Join(projectRoot, "backend"))
	if err != nil {
		exitWith(err)
	}

	// Stable alphabetical ordering by (qualified reference) — keeps diffs clean
	// regardless of filesystem walk order.
	aliases := assignPackageAliases(entries)
	sort.Slice(entries, func(i, j int) bool {
		return qualifiedReference(entries[i], aliases) < qualifiedReference(entries[j], aliases)
	})

	source, err := renderControllersFile(entries, aliases)
	if err != nil {
		exitWith(err)
	}

	outputPath := filepath.Join(projectRoot, controllersOutputPath)
	if err := os.WriteFile(outputPath, source, 0644); err != nil {
		exitWith(err)
	}
	fmt.Printf("generated %s (%d controllers)\n", controllersOutputPath, len(entries))
}

func findProjectRootDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if pathExists(filepath.Join(currentDir, "backend")) && pathExists(filepath.Join(currentDir, "scripts")) {
			return currentDir, nil
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return "", fmt.Errorf("could not find project root containing backend/ and scripts/")
		}
		currentDir = parent
	}
}

// scanTableStructs walks backend/ and returns one entry per struct that embeds
// db.TableStruct[XTable, X] (or, when inside the db package itself, TableStruct[XTable, X])
// AND whose own type name equals the second type argument. That second-arg check is what
// distinguishes the "base" struct from its companion "Table" struct, since both embed the
// same TableStruct generic.
func scanTableStructs(backendDir string) ([]tableEntry, error) {
	var entries []tableEntry

	err := filepath.WalkDir(backendDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			// Skip vendor / VCS / dependency trees plus app/db itself — ORM-internal tables
			// (Increment, CacheVersion) are bootstrapped by db.Init() and must not appear in
			// MakeScyllaControllers().
			if name == "vendor" || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			if path == filepath.Join(backendDir, "db") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.HasSuffix(path, "controllers.generated.go") {
			return nil
		}

		fileSet := token.NewFileSet()
		parsedFile, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		packagePath, err := derivePackageImportPath(backendDir, path)
		if err != nil {
			return err
		}

		for _, declaration := range parsedFile.Decls {
			genericDecl, ok := declaration.(*ast.GenDecl)
			if !ok || genericDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genericDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok || structType.Fields == nil || len(structType.Fields.List) == 0 {
					continue
				}

				if isBaseTableStruct(structType.Fields.List[0], typeSpec.Name.Name) {
					entries = append(entries, tableEntry{
						TypeName:    typeSpec.Name.Name,
						PackagePath: packagePath,
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// isBaseTableStruct returns true when the first field of a struct is an embedded
// db.TableStruct[XTable, X] AND the second type argument matches the struct's own name.
// That second-arg match is what selects the "base" struct of each pair (the X), never
// the companion XTable struct (which embeds the same generic).
func isBaseTableStruct(firstField *ast.Field, ownTypeName string) bool {
	// An embedded field has no Names.
	if len(firstField.Names) != 0 {
		return false
	}

	// Generic embed: TableStruct[A, B] parses as ast.IndexListExpr; a single type arg
	// would be ast.IndexExpr. The ORM contract always uses two type args.
	indexList, ok := firstField.Type.(*ast.IndexListExpr)
	if !ok || len(indexList.Indices) != 2 {
		return false
	}

	selector, ok := indexList.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	packageIdent, ok := selector.X.(*ast.Ident)
	if !ok || packageIdent.Name != "db" || selector.Sel.Name != "TableStruct" {
		return false
	}

	secondArgIdent, ok := indexList.Indices[1].(*ast.Ident)
	if !ok {
		return false
	}
	return secondArgIdent.Name == ownTypeName
}

// derivePackageImportPath turns "/abs/path/to/backend/logistica/types/foo.go" into
// "app/logistica/types". The module name "app" is fixed by backend/go.mod.
func derivePackageImportPath(backendDir, filePath string) (string, error) {
	relativeDir, err := filepath.Rel(backendDir, filepath.Dir(filePath))
	if err != nil {
		return "", err
	}
	if relativeDir == "." {
		return "app", nil
	}
	return "app/" + filepath.ToSlash(relativeDir), nil
}

// assignPackageAliases standardizes one alias per package across the generated file.
// Rules:
//   - packages ending in "/types" get "<parentDir>Types" (e.g. app/configuracion/types -> configuracionTypes)
//   - the top-level "app/types" gets "appTypes"
//   - app/exec is the output package itself; it carries no alias (types are referenced bare)
//   - any other package uses its trailing path segment as both the bare import name and the qualifier
//
// Returns a map: packagePath -> alias (or "" when the type is in the output package).
func assignPackageAliases(entries []tableEntry) map[string]string {
	aliasByPackage := map[string]string{}
	for _, entry := range entries {
		if _, exists := aliasByPackage[entry.PackagePath]; exists {
			continue
		}
		aliasByPackage[entry.PackagePath] = deriveAliasForPackage(entry.PackagePath)
	}
	return aliasByPackage
}

func deriveAliasForPackage(packagePath string) string {
	if packagePath == controllersOutputPackage {
		return ""
	}
	if packagePath == "app/types" {
		return "appTypes"
	}
	segments := strings.Split(packagePath, "/")
	lastSegment := segments[len(segments)-1]
	if lastSegment == "types" && len(segments) >= 2 {
		return segments[len(segments)-2] + "Types"
	}
	// Use the bare directory name; this happens to match the package name for db, core, etc.
	return lastSegment
}

func qualifiedReference(entry tableEntry, aliases map[string]string) string {
	alias := aliases[entry.PackagePath]
	if alias == "" {
		return entry.TypeName
	}
	return alias + "." + entry.TypeName
}

// renderControllersFile builds the final Go source for controllers.generated.go.
// We hand-assemble the file (rather than building AST nodes) since the output is small
// and tightly templated, then run go/format to canonicalize whitespace.
func renderControllersFile(entries []tableEntry, aliases map[string]string) ([]byte, error) {
	imports := buildImportLines(entries, aliases)

	var buffer bytes.Buffer
	buffer.WriteString("// Code generated by scripts/controllers/controllers_generator.go; DO NOT EDIT.\n")
	buffer.WriteString("package exec\n\n")

	if len(imports) > 0 {
		buffer.WriteString("import (\n")
		for _, line := range imports {
			buffer.WriteString("\t")
			buffer.WriteString(line)
			buffer.WriteString("\n")
		}
		buffer.WriteString(")\n\n")
	}

	buffer.WriteString("func MakeScyllaControllers() []db.ScyllaControllerInterface {\n")
	buffer.WriteString("\treturn []db.ScyllaControllerInterface{\n")
	for _, entry := range entries {
		fmt.Fprintf(&buffer, "\t\tmakeDBController[%s](),\n", qualifiedReference(entry, aliases))
	}
	buffer.WriteString("\t}\n")
	buffer.WriteString("}\n")

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		// Surface the unformatted source on failure so the underlying syntax error is visible.
		return nil, fmt.Errorf("format generated source: %w\n--- source ---\n%s", err, buffer.String())
	}
	return formatted, nil
}

// buildImportLines emits the import block lines, always including app/db (the return
// type's package) even when no scanned entries live there.
func buildImportLines(entries []tableEntry, aliases map[string]string) []string {
	requiredPackages := map[string]bool{"app/db": true}
	for _, entry := range entries {
		if entry.PackagePath == controllersOutputPackage {
			continue
		}
		requiredPackages[entry.PackagePath] = true
	}

	packagePaths := make([]string, 0, len(requiredPackages))
	for packagePath := range requiredPackages {
		packagePaths = append(packagePaths, packagePath)
	}
	sort.Strings(packagePaths)

	lines := make([]string, 0, len(packagePaths))
	for _, packagePath := range packagePaths {
		alias := aliases[packagePath]
		if alias == "" {
			// app/db has no scanned entries so it has no alias entry; fall through to bare import.
			alias = deriveAliasForPackage(packagePath)
		}
		// Use bare import (no alias prefix) when the alias already matches the trailing segment —
		// keeps the file readable for simple packages like "app/db", "app/core".
		segments := strings.Split(packagePath, "/")
		if alias == segments[len(segments)-1] {
			lines = append(lines, fmt.Sprintf("%q", packagePath))
		} else {
			lines = append(lines, fmt.Sprintf("%s %q", alias, packagePath))
		}
	}
	return lines
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func exitWith(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
