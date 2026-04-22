package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/tools/go/ast/astutil"
)

type backendStruct struct {
	Folder string
	Name   string
	Fields []backendField
}

type backendField struct {
	GoName string
	TSName string
	TSType string
}

type tsInterface struct {
	Name     string
	Body     string
	Start    int
	BodyFrom int
	BodyTo   int
	End      int
}

type fieldLine struct {
	Name string
	Text string
}

var structTagPattern = regexp.MustCompile(`(?m)^[ \t]*//STRUCT:([A-Za-z0-9_.]+)[ \t]*$`)

func main() {
	projectRoot, err := findProjectRoot()
	if err != nil {
		exitWithError(err)
	}
	structs, err := collectBackendStructs(projectRoot)
	if err != nil {
		exitWithError(err)
	}

	updated, err := syncFrontendInterfaces(projectRoot, structs)
	if err != nil {
		exitWithError(err)
	}
	if updated == 0 {
		fmt.Println("no changes needed")
	} else {
		fmt.Printf("%d file(s) updated\n", updated)
	}
}

func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(currentDir, "backend")) && exists(filepath.Join(currentDir, "frontend")) {
			return currentDir, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", errors.New("could not find project root with backend and frontend directories")
		}
		currentDir = parentDir
	}
}

func collectBackendStructs(projectRoot string) (map[string][]backendStruct, error) {
	structsByName := map[string][]backendStruct{}
	backendRoot := filepath.Join(projectRoot, "backend")

	err := filepath.WalkDir(backendRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "types" {
			return nil
		}

		folderName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		fileStructs, err := parseBackendStructFile(path, folderName)
		if err != nil {
			return err
		}
		for _, backendStruct := range fileStructs {
			structsByName[backendStruct.Name] = append(structsByName[backendStruct.Name], backendStruct)
		}
		return nil
	})
	return structsByName, err
}

func parseBackendStructFile(path string, folderName string) ([]backendStruct, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var structs []backendStruct
	astutil.Apply(parsedFile, func(cursor *astutil.Cursor) bool {
		typeSpec, ok := cursor.Node().(*ast.TypeSpec)
		if !ok {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		backendStruct := backendStruct{Folder: folderName, Name: typeSpec.Name.Name}
		for _, structField := range structType.Fields.List {
			// Embedded ORM fields do not exist in the JSON payload.
			if len(structField.Names) == 0 {
				continue
			}
			for _, fieldName := range structField.Names {
				if !fieldName.IsExported() || fieldName.Name == "EmpresaID" || fieldName.Name == "CompanyID" {
					continue
				}
				jsonName := jsonFieldName(fieldName.Name, structField.Tag)
				if jsonName == "-" {
					continue
				}
				backendStruct.Fields = append(backendStruct.Fields, backendField{
					GoName: fieldName.Name,
					TSName: jsonName,
					TSType: goTypeToTypeScript(structField.Type),
				})
			}
		}
		structs = append(structs, backendStruct)
		return false
	}, nil)

	return structs, nil
}

func syncFrontendInterfaces(projectRoot string, structsByName map[string][]backendStruct) (int, error) {
	frontendRoutes := filepath.Join(projectRoot, "frontend", "routes")
	updatedCount := 0
	scannedCount := 0

	err := filepath.WalkDir(frontendRoutes, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".ts" {
			return nil
		}
		scannedCount++

		rawFile, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(rawFile)
		matches := structTagPattern.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			tag := m[1]
			if resolved, err := resolveStruct(tag, structsByName); err == nil {
				relPath, _ := filepath.Rel(projectRoot, path)
				fmt.Printf("match: %s -> %s.%s (%d fields)\n", relPath, resolved.Folder, resolved.Name, len(resolved.Fields))
			}
		}

		updatedFile, changed, err := syncFile(content, structsByName)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !changed {
			return nil
		}
		fmt.Printf("  -> updated %s\n", path)
		updatedCount++
		return os.WriteFile(path, []byte(updatedFile), 0644)
	})

	fmt.Printf("scanned %d .ts file(s)\n", scannedCount)
	return updatedCount, err
}

func syncFile(content string, structsByName map[string][]backendStruct) (string, bool, error) {
	matches := structTagPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, false, nil
	}

	declaredInterfaces := collectDeclaredInterfaceNames(content)
	var output strings.Builder
	previousEnd := 0
	changed := false

	for _, match := range matches {
		tagEnd := match[1]
		structReference := content[match[2]:match[3]]

		interfaceBlock, err := findNextInterface(content, tagEnd)
		if err != nil {
			return "", false, err
		}
		backendStruct, err := resolveStruct(structReference, structsByName)
		if err != nil {
			return "", false, err
		}

		newBody := buildInterfaceBody(interfaceBlock.Body, backendStruct.Fields, declaredInterfaces)
		output.WriteString(content[previousEnd:interfaceBlock.BodyFrom])
		output.WriteString(newBody)
		previousEnd = interfaceBlock.BodyTo
		changed = changed || newBody != interfaceBlock.Body
	}

	output.WriteString(content[previousEnd:])
	return output.String(), changed, nil
}

func findNextInterface(content string, offset int) (tsInterface, error) {
	interfacePattern := regexp.MustCompile(`export\s+interface\s+([A-Za-z_][A-Za-z0-9_]*)[^{]*\{`)
	location := interfacePattern.FindStringSubmatchIndex(content[offset:])
	if location == nil {
		return tsInterface{}, errors.New("STRUCT tag is not followed by an exported interface")
	}

	interfaceStart := offset + location[0]
	bodyStart := offset + location[1]
	interfaceName := content[offset+location[2] : offset+location[3]]
	bodyEnd, err := findMatchingBrace(content, bodyStart-1)
	if err != nil {
		return tsInterface{}, err
	}

	return tsInterface{
		Name:     interfaceName,
		Body:     content[bodyStart:bodyEnd],
		Start:    interfaceStart,
		BodyFrom: bodyStart,
		BodyTo:   bodyEnd,
		End:      bodyEnd + 1,
	}, nil
}

func findMatchingBrace(content string, openBrace int) (int, error) {
	depth := 0
	inLineComment := false
	inBlockComment := false
	inString := rune(0)

	for index, currentRune := range content[openBrace:] {
		absoluteIndex := openBrace + index
		nextByte := byte(0)
		if absoluteIndex+1 < len(content) {
			nextByte = content[absoluteIndex+1]
		}

		if inLineComment {
			if currentRune == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if currentRune == '*' && nextByte == '/' {
				inBlockComment = false
			}
			continue
		}
		if inString != 0 {
			if currentRune == '\\' {
				continue
			}
			if currentRune == inString {
				inString = 0
			}
			continue
		}

		if currentRune == '/' && nextByte == '/' {
			inLineComment = true
			continue
		}
		if currentRune == '/' && nextByte == '*' {
			inBlockComment = true
			continue
		}
		if currentRune == '\'' || currentRune == '"' || currentRune == '`' {
			inString = currentRune
			continue
		}
		if currentRune == '{' {
			depth++
			continue
		}
		if currentRune == '}' {
			depth--
			if depth == 0 {
				return absoluteIndex, nil
			}
		}
	}
	return 0, errors.New("could not find matching interface brace")
}

func resolveStruct(reference string, structsByName map[string][]backendStruct) (backendStruct, error) {
	folderName := ""
	structName := reference
	if strings.Contains(reference, ".") {
		parts := strings.Split(reference, ".")
		if len(parts) != 2 {
			return backendStruct{}, fmt.Errorf("invalid STRUCT reference %q", reference)
		}
		folderName = parts[0]
		structName = parts[1]
	}

	candidates := structsByName[structName]
	if len(candidates) == 0 {
		return backendStruct{}, fmt.Errorf("backend struct %q was not found", reference)
	}

	if folderName != "" {
		for _, candidate := range candidates {
			if candidate.Folder == folderName {
				return candidate, nil
			}
		}
		return backendStruct{}, fmt.Errorf("backend struct %q was not found in folder %q", structName, folderName)
	}

	if len(candidates) > 1 {
		folders := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			folders = append(folders, candidate.Folder)
		}
		sort.Strings(folders)
		return backendStruct{}, fmt.Errorf("STRUCT:%s is ambiguous; use one of these folders: %s", structName, strings.Join(folders, ", "))
	}
	return candidates[0], nil
}

func buildInterfaceBody(currentBody string, backendFields []backendField, declaredInterfaces map[string]bool) string {
	existingFields, extraFields := splitInterfaceFields(currentBody, backendFields)
	var buffer strings.Builder

	buffer.WriteByte('\n')
	for _, backendField := range backendFields {
		typeScriptType := backendField.TSType
		existingField, fieldExists := existingFields[backendField.TSName]
		if isUnknownStructType(typeScriptType) && fieldExists && isUnknownStructType(fieldType(existingField.Text)) {
			typeScriptType = fieldType(existingField.Text)
		} else if isUnknownStructType(typeScriptType) && !declaredInterfaces[strings.TrimSuffix(typeScriptType, "[]")] {
			typeScriptType = "any"
		}

		// Preserve aliases and local comments when an existing frontend field already maps this struct field.
		if fieldExists && typeScriptType != "any" {
			buffer.WriteString("  ")
			buffer.WriteString(existingField.Name)
			buffer.WriteString(": ")
			buffer.WriteString(typeScriptType)
			buffer.WriteString(fieldTerminator(existingField.Text))
			buffer.WriteByte('\n')
			continue
		}
		buffer.WriteString("  ")
		buffer.WriteString(backendField.TSName)
		buffer.WriteString(": ")
		buffer.WriteString(typeScriptType)
		buffer.WriteByte('\n')
	}

	if len(extraFields) > 0 {
		buffer.WriteString("  /* extra fields */\n")
		for _, extraField := range extraFields {
			buffer.WriteString(extraField.Text)
			if !strings.HasSuffix(extraField.Text, "\n") {
				buffer.WriteByte('\n')
			}
		}
	}

	return buffer.String()
}

func splitInterfaceFields(body string, backendFields []backendField) (map[string]fieldLine, []fieldLine) {
	backendNames := map[string]bool{}
	for _, backendField := range backendFields {
		backendNames[backendField.TSName] = true
	}

	beforeExtraBody := body
	afterExtraBody := ""
	if extraMarkerIndex := strings.Index(body, "/* extra fields */"); extraMarkerIndex >= 0 {
		beforeExtraBody = body[:extraMarkerIndex]
		afterExtraBody = body[extraMarkerIndex+len("/* extra fields */"):]
	}

	fieldLines := parseInterfaceFieldLines(beforeExtraBody)
	existingFields := map[string]fieldLine{}
	var extraFields []fieldLine
	for _, line := range fieldLines {
		if backendNames[line.Name] {
			existingFields[line.Name] = line
			continue
		}
		extraFields = append(extraFields, line)
	}

	for _, line := range parseInterfaceFieldLines(afterExtraBody) {
		extraFields = append(extraFields, line)
	}

	return existingFields, extraFields
}

func parseInterfaceFieldLines(body string) []fieldLine {
	lines := strings.SplitAfter(body, "\n")
	fields := make([]fieldLine, 0, len(lines))
	var pendingComments []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "/*") || strings.HasPrefix(trimmedLine, "*") {
			pendingComments = append(pendingComments, line)
			continue
		}
		fieldName := parseTypeScriptFieldName(trimmedLine)
		if fieldName == "" {
			pendingComments = nil
			continue
		}
		fieldText := strings.Join(pendingComments, "") + line
		fields = append(fields, fieldLine{Name: fieldName, Text: fieldText})
		pendingComments = nil
	}

	return fields
}

func parseTypeScriptFieldName(line string) string {
	for index, currentRune := range line {
		if currentRune == ':' || currentRune == '?' {
			name := strings.TrimSpace(line[:index])
			if isTypeScriptIdentifier(name) {
				return name
			}
			return ""
		}
	}
	return ""
}

func isTypeScriptIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for index, currentRune := range name {
		if index == 0 && !(unicode.IsLetter(currentRune) || currentRune == '_' || currentRune == '$') {
			return false
		}
		if !(unicode.IsLetter(currentRune) || unicode.IsDigit(currentRune) || currentRune == '_' || currentRune == '$') {
			return false
		}
	}
	return true
}

func collectDeclaredInterfaceNames(content string) map[string]bool {
	declared := map[string]bool{}
	interfacePattern := regexp.MustCompile(`\bexport\s+interface\s+([A-Za-z_][A-Za-z0-9_]*)`)
	for _, match := range interfacePattern.FindAllStringSubmatch(content, -1) {
		declared[match[1]] = true
	}
	return declared
}

func fieldType(text string) string {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	typePart := strings.TrimSpace(parts[1])
	typePart = strings.TrimSuffix(typePart, ",")
	typePart = strings.TrimSuffix(typePart, ";")
	return strings.TrimSpace(typePart)
}

func fieldTerminator(text string) string {
	trimmedText := strings.TrimRight(text, "\r\n\t ")
	if strings.HasSuffix(trimmedText, ",") || strings.HasSuffix(trimmedText, ";") {
		return string(trimmedText[len(trimmedText)-1])
	}
	return ""
}

func jsonFieldName(goName string, tagLiteral *ast.BasicLit) string {
	if tagLiteral == nil {
		return goName
	}
	tag := strings.Trim(tagLiteral.Value, "`")
	for _, part := range strings.Split(tag, " ") {
		if !strings.HasPrefix(part, `json:"`) {
			continue
		}
		jsonTag := strings.TrimPrefix(part, `json:"`)
		jsonTag = strings.TrimSuffix(jsonTag, `"`)
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			return goName
		}
		return jsonName
	}
	return goName
}

func goTypeToTypeScript(expression ast.Expr) string {
	switch typedExpression := expression.(type) {
	case *ast.Ident:
		return identToTypeScript(typedExpression.Name)
	case *ast.ArrayType:
		return goTypeToTypeScript(typedExpression.Elt) + "[]"
	case *ast.StarExpr:
		return goTypeToTypeScript(typedExpression.X)
	case *ast.SelectorExpr:
		return typedExpression.Sel.Name
	case *ast.MapType:
		return "Record<string, " + goTypeToTypeScript(typedExpression.Value) + ">"
	default:
		var source bytes.Buffer
		if err := format.Node(&source, token.NewFileSet(), expression); err != nil {
			return "any"
		}
		return source.String()
	}
}

func identToTypeScript(name string) string {
	switch name {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "number"
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "any", "interface{}":
		return "any"
	default:
		return "I" + name
	}
}

func isUnknownStructType(typeScriptType string) bool {
	baseType := strings.TrimSuffix(typeScriptType, "[]")
	return strings.HasPrefix(baseType, "I")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
