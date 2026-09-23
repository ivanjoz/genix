// Copies a backend catalog — a `var X = []T{...}` of literals — into the frontend as a TS
// array, so a list the UI shows and the backend validates is written down once.
//
// This is the sibling of the //STRUCT: sync in sync_struct_interfaces.go: that one carries a
// struct's *shape* across, this one carries a slice's *contents*. Both run under
// `go run ./scripts sync_struct_interfaces`.
//
// Marked in the frontend file by a //CATALOG:<module>.<VarName> comment directly above the
// declaration whose array literal is to be filled:
//
//	//CATALOG:crm.IdentityDocOptions
//	export const IDENTITY_DOC_OPTIONS: IIdentityDocOption[] = [
//	  { id: 1, code: '1', name: 'DNI|DNI' },
//	]
//
// Only the text between `[` and `]` is rewritten. The exported name and the TS type
// annotation stay hand-written, which keeps the generator out of the business of inventing
// TypeScript types and lets a route name the list whatever reads best there.
//
// Go field names arrive lower-cased (`ID` -> `id`, `Code` -> `code`) because the result is
// consumed as option objects by pickers, not as a record mirror.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var catalogTagPattern = regexp.MustCompile(`(?m)^[ \t]*//CATALOG:([A-Za-z0-9_.]+)[ \t]*$`)

// syncFrontendCatalogs fills every //CATALOG: array under routes and services.
//
// It reaches further than the //STRUCT: pass, which only covers routes: a list that both a
// route and a domain component show belongs with the shared API connectors in `services`.
func syncFrontendCatalogs(projectRoot string, catalogsByName map[string][]backendCatalog) (int, error) {
	updatedCount := 0

	walkFile := func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".ts" {
			return nil
		}
		rawFile, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updatedFile, changed, err := syncCatalogFile(string(rawFile), catalogsByName)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !changed {
			return nil
		}
		relPath, _ := filepath.Rel(projectRoot, path)
		fmt.Printf("  -> catalog updated %s\n", relPath)
		updatedCount++
		return os.WriteFile(path, []byte(updatedFile), 0644)
	}

	for _, folder := range []string{"routes", "services"} {
		if err := filepath.WalkDir(filepath.Join(projectRoot, "frontend", folder), walkFile); err != nil {
			return updatedCount, err
		}
	}
	return updatedCount, nil
}

// catalogField is one key of one row, already rendered as the TypeScript literal it becomes.
type catalogField struct {
	Name string
	Text string
}

type backendCatalog struct {
	Folder string
	Name   string
	Rows   [][]catalogField
}

// collectBackendCatalogs reads every `var X = []T{...}` in a backend `*/types` folder whose
// elements are struct literals of constants and basic literals.
//
// Constants are resolved rather than emitted by name: the whole point is that the frontend
// receives `1`, not `IdentityDocDNI`, which it has no way to look up. They are gathered per
// folder in a first pass because a catalog and the constants it names routinely live in the
// same package but not the same file.
func collectBackendCatalogs(projectRoot string) (map[string][]backendCatalog, error) {
	backendRoot := filepath.Join(projectRoot, "backend")
	constantsByFolder := map[string]map[string]string{}
	filesByFolder := map[string][]*ast.File{}
	fileSet := token.NewFileSet()

	err := filepath.WalkDir(backendRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "types" {
			return nil
		}
		parsedFile, parseErr := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("%s: %w", path, parseErr)
		}
		folderName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		filesByFolder[folderName] = append(filesByFolder[folderName], parsedFile)
		if constantsByFolder[folderName] == nil {
			constantsByFolder[folderName] = map[string]string{}
		}
		collectConstantLiterals(parsedFile, constantsByFolder[folderName])
		return nil
	})
	if err != nil {
		return nil, err
	}

	catalogsByName := map[string][]backendCatalog{}
	for folderName, parsedFiles := range filesByFolder {
		for _, parsedFile := range parsedFiles {
			for _, catalog := range parseCatalogVars(parsedFile, folderName, constantsByFolder[folderName]) {
				catalogsByName[catalog.Name] = append(catalogsByName[catalog.Name], catalog)
			}
		}
	}
	return catalogsByName, nil
}

// collectConstantLiterals records `const Name = <literal>` so a catalog row naming a constant
// can be flattened to its value.
func collectConstantLiterals(parsedFile *ast.File, into map[string]string) {
	for _, declaration := range parsedFile.Decls {
		generalDeclaration, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || generalDeclaration.Tok != token.CONST {
			continue
		}
		for _, specification := range generalDeclaration.Specs {
			valueSpec, isValue := specification.(*ast.ValueSpec)
			if !isValue || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			if literal, isLiteral := valueSpec.Values[0].(*ast.BasicLit); isLiteral {
				into[valueSpec.Names[0].Name] = literal.Value
			}
		}
	}
}

func parseCatalogVars(
	parsedFile *ast.File, folderName string, constants map[string]string,
) []backendCatalog {

	catalogs := []backendCatalog{}
	for _, declaration := range parsedFile.Decls {
		generalDeclaration, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || generalDeclaration.Tok != token.VAR {
			continue
		}
		for _, specification := range generalDeclaration.Specs {
			valueSpec, isValue := specification.(*ast.ValueSpec)
			if !isValue || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			sliceLiteral, isComposite := valueSpec.Values[0].(*ast.CompositeLit)
			if !isComposite {
				continue
			}
			if _, isSlice := sliceLiteral.Type.(*ast.ArrayType); !isSlice {
				continue
			}
			rows, ok := parseCatalogRows(sliceLiteral, constants)
			if !ok {
				continue
			}
			catalogs = append(catalogs, backendCatalog{
				Folder: folderName,
				Name:   valueSpec.Names[0].Name,
				Rows:   rows,
			})
		}
	}
	return catalogs
}

// parseCatalogRows flattens the slice's elements, and reports false for anything it cannot
// render literally — a slice of plain strings, a row built by a function call. Such a var is
// simply not a catalog, and skipping it silently is what lets this walk every types folder.
func parseCatalogRows(sliceLiteral *ast.CompositeLit, constants map[string]string) ([][]catalogField, bool) {
	rows := make([][]catalogField, 0, len(sliceLiteral.Elts))
	for _, element := range sliceLiteral.Elts {
		rowLiteral, isComposite := element.(*ast.CompositeLit)
		if !isComposite || len(rowLiteral.Elts) == 0 {
			return nil, false
		}
		fields := make([]catalogField, 0, len(rowLiteral.Elts))
		for _, rowElement := range rowLiteral.Elts {
			keyValue, isKeyValue := rowElement.(*ast.KeyValueExpr)
			if !isKeyValue {
				return nil, false
			}
			fieldName, isIdentifier := keyValue.Key.(*ast.Ident)
			if !isIdentifier {
				return nil, false
			}
			text, resolved := catalogValueText(keyValue.Value, constants)
			if !resolved {
				return nil, false
			}
			fields = append(fields, catalogField{Name: typeScriptFieldName(fieldName.Name), Text: text})
		}
		rows = append(rows, fields)
	}
	return rows, len(rows) > 0
}

// catalogValueText renders one Go value as the TypeScript literal it becomes. Go's own
// double-quoted strings are re-quoted single, which is what the frontend style uses.
func catalogValueText(expression ast.Expr, constants map[string]string) (string, bool) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		return goLiteralToTypeScript(value.Value), true
	case *ast.Ident:
		constantValue, isKnown := constants[value.Name]
		if !isKnown {
			return "", false
		}
		return goLiteralToTypeScript(constantValue), true
	case *ast.UnaryExpr:
		// Negative numbers parse as a unary minus over a literal.
		if value.Op != token.SUB {
			return "", false
		}
		inner, resolved := catalogValueText(value.X, constants)
		return "-" + inner, resolved
	}
	return "", false
}

func goLiteralToTypeScript(goLiteral string) string {
	if !strings.HasPrefix(goLiteral, `"`) {
		return goLiteral
	}
	unquoted := strings.TrimSuffix(strings.TrimPrefix(goLiteral, `"`), `"`)
	unquoted = strings.ReplaceAll(unquoted, `\"`, `"`)
	unquoted = strings.ReplaceAll(unquoted, `\`, `\\`)
	unquoted = strings.ReplaceAll(unquoted, `'`, `\'`)
	return "'" + unquoted + "'"
}

// typeScriptFieldName lowercases a Go field for use as an option key: an all-caps name goes
// down whole (`ID` -> `id`), anything else loses only its first letter.
func typeScriptFieldName(goName string) string {
	if goName == strings.ToUpper(goName) {
		return strings.ToLower(goName)
	}
	return strings.ToLower(goName[:1]) + goName[1:]
}

func syncCatalogFile(content string, catalogsByName map[string][]backendCatalog) (string, bool, error) {
	matches := catalogTagPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, false, nil
	}

	var output strings.Builder
	previousEnd := 0
	changed := false

	for _, match := range matches {
		catalogReference := content[match[2]:match[3]]
		catalog, err := resolveCatalog(catalogReference, catalogsByName)
		if err != nil {
			return "", false, err
		}

		// The `[` to fill is the one after `=`, never the first one on the line: a declaration
		// annotated `IIdentityDocOption[]` carries a bracket pair in its *type* as well.
		assignment := strings.Index(content[match[1]:], "=")
		if assignment < 0 {
			return "", false, fmt.Errorf("CATALOG tag %q is not followed by an assignment", catalogReference)
		}
		openBracket := strings.Index(content[match[1]+assignment:], "[")
		if openBracket < 0 {
			return "", false, fmt.Errorf("CATALOG tag %q is not followed by an array literal", catalogReference)
		}
		openBracket += match[1] + assignment
		closeBracket, err := findMatchingDelimiter(content, openBracket, '[', ']')
		if err != nil {
			return "", false, fmt.Errorf("CATALOG tag %q: %w", catalogReference, err)
		}

		newBody := buildCatalogBody(catalog, indentationOf(content, match[0]))
		output.WriteString(content[previousEnd : openBracket+1])
		output.WriteString(newBody)
		previousEnd = closeBracket
		changed = changed || newBody != content[openBracket+1:closeBracket]
	}

	output.WriteString(content[previousEnd:])
	return output.String(), changed, nil
}

// indentationOf returns the leading whitespace of the line the marker sits on, so a catalog
// nested inside a block keeps the surrounding indentation.
func indentationOf(content string, markerStart int) string {
	lineStart := strings.LastIndex(content[:markerStart], "\n") + 1
	return content[lineStart:markerStart]
}

func buildCatalogBody(catalog backendCatalog, indentation string) string {
	var body strings.Builder
	body.WriteString("\n")
	for _, row := range catalog.Rows {
		pairs := make([]string, 0, len(row))
		for _, field := range row {
			pairs = append(pairs, fmt.Sprintf("%s: %s", field.Name, field.Text))
		}
		body.WriteString(fmt.Sprintf("%s  { %s },\n", indentation, strings.Join(pairs, ", ")))
	}
	body.WriteString(indentation)
	return body.String()
}

func resolveCatalog(reference string, catalogsByName map[string][]backendCatalog) (backendCatalog, error) {
	folderName, catalogName, hasFolder := strings.Cut(reference, ".")
	if !hasFolder {
		folderName, catalogName = "", reference
	}

	candidates := catalogsByName[catalogName]
	if len(candidates) == 0 {
		return backendCatalog{}, fmt.Errorf("no backend catalog named %q", catalogName)
	}
	if folderName == "" {
		if len(candidates) > 1 {
			return backendCatalog{}, fmt.Errorf(
				"catalog %q is declared in several modules; qualify it as <module>.%s", catalogName, catalogName)
		}
		return candidates[0], nil
	}
	for _, candidate := range candidates {
		if candidate.Folder == folderName {
			return candidate, nil
		}
	}
	return backendCatalog{}, fmt.Errorf("no catalog %q in module %q", catalogName, folderName)
}
