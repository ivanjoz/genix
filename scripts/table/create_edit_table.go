package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// Field represents a parsed struct field.
type Field struct {
	Name       string
	TypeName   string
	IsKey      bool
	IsSlice    bool
	SliceEType string
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "create":
		NewTable(args)
	case "edit":
		NewTableColumn(args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go run create_edit_table.go <command> [args...]")
	fmt.Println("Commands:")
	fmt.Println("  create <output_path> <table_name> [field:type:key]...")
	fmt.Println("  edit   <table_name> <field:type[:key]>")
}

// NewTable creates a new database table structure from scratch.
func NewTable(args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: create <output_path> <table_name> [field:type:key]...")
	}

	outputPath := args[0]
	tableNameSnake := args[1]
	tableNameCamel := toCamelCase(tableNameSnake)

	backendDir := "backend"
	if _, err := os.Stat(backendDir); os.IsNotExist(err) {
		backendDir = "../backend"
	}

	// Check if the type is already declared
	if isTypeDeclared(backendDir, tableNameCamel) {
		log.Fatalf("Error: Type %s is already declared in the backend.", tableNameCamel)
	}
	if isTypeDeclared(backendDir, tableNameCamel+"Table") {
		log.Fatalf("Error: Type %s is already declared in the backend.", tableNameCamel+"Table")
	}

	var userFields []Field
	if len(args) > 2 {
		for _, arg := range args[2:] {
			parts := strings.Split(arg, ":")
			if len(parts) < 2 {
				log.Fatalf("Invalid field format: %s", arg)
			}

			field := Field{Name: toCamelCase(parts[0])}

			if strings.HasPrefix(parts[1], "[]") {
				field.IsSlice = true
				field.SliceEType = strings.TrimPrefix(parts[1], "[]")
				field.TypeName = parts[1]
			} else {
				field.TypeName = parts[1]
			}

			if len(parts) > 2 && parts[2] == "key" {
				field.IsKey = true
			}

			userFields = append(userFields, field)
		}
	}

	// Generate the Go source code as a string
	sourceCode := generateSourceCode(tableNameCamel, tableNameSnake, userFields)

	// Format the code
	formatted, err := format.Source([]byte(sourceCode))
	if err != nil {
		log.Fatalf("Failed to format generated code: %v\nGenerated code:\n%s", err, sourceCode)
	}

	// Write the file
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("Successfully created %s\n", outputPath)
}

func generateSourceCode(tableNameCamel, tableNameSnake string, userFields []Field) string {
	var sb strings.Builder

	sb.WriteString("package types\n\n")
	sb.WriteString("import \"app/db\"\n\n")

	// --- Base Struct ---
	sb.WriteString(fmt.Sprintf("type %s struct {\n", tableNameCamel))
	sb.WriteString(fmt.Sprintf("\tdb.TableStruct[%sTable, %s]\n", tableNameCamel, tableNameCamel))
	sb.WriteString(buildFieldString("EmpresaID", "int32", true, false, ""))
	sb.WriteString(buildFieldString("Status", "int8", false, false, ""))
	sb.WriteString(buildFieldString("Updated", "int64", false, false, ""))
	sb.WriteString(buildFieldString("UpdatedBy", "int32", false, false, ""))
	for _, f := range userFields {
		sb.WriteString(buildFieldString(f.Name, f.TypeName, f.IsKey, f.IsSlice, f.SliceEType))
	}
	sb.WriteString("}\n\n")

	// --- Table Struct ---
	sb.WriteString(fmt.Sprintf("type %sTable struct {\n", tableNameCamel))
	sb.WriteString(fmt.Sprintf("\tdb.TableStruct[%sTable, %s]\n", tableNameCamel, tableNameCamel))
	sb.WriteString(buildTableFieldString("EmpresaID", "int32", false, "", tableNameCamel))
	sb.WriteString(buildTableFieldString("Status", "int8", false, "", tableNameCamel))
	sb.WriteString(buildTableFieldString("Updated", "int64", false, "", tableNameCamel))
	sb.WriteString(buildTableFieldString("UpdatedBy", "int32", false, "", tableNameCamel))
	for _, f := range userFields {
		sb.WriteString(buildTableFieldString(f.Name, f.TypeName, f.IsSlice, f.SliceEType, tableNameCamel))
	}
	sb.WriteString("}\n\n")

	// --- GetSchema Method ---
	sb.WriteString(fmt.Sprintf("func (e %sTable) GetSchema() db.TableSchema {\n", tableNameCamel))
	sb.WriteString("\treturn db.TableSchema{\n")
	sb.WriteString(fmt.Sprintf("\t\tName:      \"%s\",\n", tableNameSnake))
	sb.WriteString("\t\tPartition: e.EmpresaID,\n")

	var keyFields []string
	for _, f := range userFields {
		if f.IsKey {
			keyFields = append(keyFields, "e."+f.Name)
		}
	}
	sb.WriteString(fmt.Sprintf("\t\tKeys:      []db.Coln{%s},\n", strings.Join(keyFields, ", ")))

	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	return sb.String()
}

func buildFieldString(name, typeName string, isKey, isSlice bool, sliceEType string) string {
	snakeName := toSnakeCase(name)
	dbTag := fmt.Sprintf(`db:"%s"`, snakeName)
	if isKey && name != "EmpresaID" {
		dbTag = fmt.Sprintf(`db:"%s,pk"`, snakeName)
	}

	jsonTag := fmt.Sprintf(`json:"%s,omitempty"`, snakeName)
	if name == "Updated" {
		jsonTag = `json:"upd,omitempty"`
	} else if name == "Status" {
		jsonTag = `json:"ss,omitempty"`
	}

	return fmt.Sprintf("\t%s %s `" + jsonTag + " " + dbTag + "`\n", name, typeName)
}

func buildTableFieldString(name, typeName string, isSlice bool, sliceEType, tableCamelName string) string {
	colType := "Col"
	elemType := typeName
	if isSlice {
		colType = "ColSlice"
		elemType = sliceEType
	}
	return fmt.Sprintf("\t%s db.%s[%sTable, %s]\n", name, colType, tableCamelName, elemType)
}

// NewTableColumn adds a new column to an existing database table.
func NewTableColumn(args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: edit <table_name> field:type[:key]...")
	}

	tableNameSnake := args[0]
	tableNameCamel := toCamelCase(tableNameSnake)

	// Parse the field
	parts := strings.Split(args[1], ":")
	if len(parts) < 2 {
		log.Fatalf("Invalid field format: %s", args[1])
	}

	field := Field{
		Name:     toCamelCase(parts[0]),
		IsKey:    len(parts) > 2 && parts[2] == "key",
	}
	if strings.HasPrefix(parts[1], "[]") {
		field.IsSlice = true
		field.SliceEType = strings.TrimPrefix(parts[1], "[]")
		field.TypeName = parts[1]
	} else {
		field.TypeName = parts[1]
	}

	backendDir := "backend"
	if _, err := os.Stat(backendDir); os.IsNotExist(err) {
		backendDir = "../backend"
	}

	// Search for the table file
	filePath, err := findTableFile(backendDir, tableNameCamel)
	if err != nil {
		log.Fatalf("Error finding table file: %v", err)
	}

	fmt.Printf("Found table file: %s\n", filePath)

	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Error parsing file: %v", err)
	}

	// Modify the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						// Base struct
						if typeSpec.Name.Name == tableNameCamel {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								newField := buildBaseFieldAST(field)
								structType.Fields.List = append(structType.Fields.List, newField)
								fmt.Printf("Added field %s to base struct %s\n", field.Name, tableNameCamel)
							}
						}
						// Table struct
						if typeSpec.Name.Name == tableNameCamel+"Table" {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								newField := buildTableFieldAST(field, tableNameCamel)
								structType.Fields.List = append(structType.Fields.List, newField)
								fmt.Printf("Added field %s to table struct %sTable\n", field.Name, tableNameCamel)
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if x.Name.Name == "GetSchema" && x.Recv != nil {
				// Check for value receiver: func (e ProductInventoryTable) GetSchema()
				if ident, ok := x.Recv.List[0].Type.(*ast.Ident); ok {
					if ident.Name == tableNameCamel+"Table" && field.IsKey {
						updateGetSchemaKeys(x.Body, field)
						fmt.Printf("Updated GetSchema for field %s as clustering key\n", field.Name)
					}
				}
				// Check for pointer receiver: func (e *ProductInventoryTable) GetSchema()
				if starExpr, ok := x.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						if ident.Name == tableNameCamel+"Table" && field.IsKey {
							updateGetSchemaKeys(x.Body, field)
							fmt.Printf("Updated GetSchema for field %s as clustering key\n", field.Name)
						}
					}
				}
			}
		}
		return true
	})

	// Format and write the file
	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		log.Fatalf("Error formatting code: %v", err)
	}

	if err := os.WriteFile(filePath, []byte(buf.String()), 0644); err != nil {
		log.Fatalf("Error writing file: %v", err)
	}

	fmt.Printf("Successfully added column %s to table %s in %s\n", field.Name, tableNameCamel, filePath)
}

func findTableFile(dir, tableName string) (string, error) {
	var foundPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if foundPath != "" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(content), "type "+tableName+" struct") {
				foundPath = path
				return filepath.SkipDir
			}
		}
		return nil
	})
	if foundPath == "" {
		return "", fmt.Errorf("table %s not found in directory %s", tableName, dir)
	}
	return foundPath, err
}

func buildBaseFieldAST(field Field) *ast.Field {
	snakeName := toSnakeCase(field.Name)

	jsonTag := fmt.Sprintf(`json:"%s,omitempty"`, snakeName)
	dbTag := fmt.Sprintf(`db:"%s"`, snakeName)
	if field.IsKey {
		dbTag = fmt.Sprintf(`db:"%s,pk"`, snakeName)
	}
	if field.Name == "Updated" {
		jsonTag = `json:"upd,omitempty"`
	} else if field.Name == "Status" {
		jsonTag = `json:"ss,omitempty"`
	}

	var fieldType ast.Expr
	if strings.HasPrefix(field.TypeName, "[]") {
		fieldType = &ast.ArrayType{
			Elt: ast.NewIdent(strings.TrimPrefix(field.TypeName, "[]")),
		}
	} else {
		fieldType = ast.NewIdent(field.TypeName)
	}

	return &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(field.Name)},
		Type:  fieldType,
		Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`" + jsonTag + " " + dbTag + "`"},
	}
}

func buildTableFieldAST(field Field, tableName string) *ast.Field {
	var colType string
	var elemType string

	if field.IsSlice {
		colType = "ColSlice"
		elemType = field.SliceEType
	} else {
		colType = "Col"
		elemType = field.TypeName
	}

	// Build db.Col[TableType, FieldType] or db.ColSlice[TableType, ElementType]
	fieldType := &ast.IndexListExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("db"),
			Sel: ast.NewIdent(colType),
		},
		Indices: []ast.Expr{
			ast.NewIdent(tableName + "Table"),
			ast.NewIdent(elemType),
		},
	}

	return &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(field.Name)},
		Type:  fieldType,
	}
}

func updateGetSchemaKeys(body *ast.BlockStmt, field Field) {
	ast.Inspect(body, func(n ast.Node) bool {
		if returnStmt, ok := n.(*ast.ReturnStmt); ok {
			if len(returnStmt.Results) > 0 {
				if compositeLit, ok := returnStmt.Results[0].(*ast.CompositeLit); ok {
					for _, elt := range compositeLit.Elts {
						if kvExpr, ok := elt.(*ast.KeyValueExpr); ok {
							if key, ok := kvExpr.Key.(*ast.Ident); ok {
								if key.Name == "Keys" {
									if keysLit, ok := kvExpr.Value.(*ast.CompositeLit); ok {
										// Add new key: e.FieldName
										newKey := &ast.SelectorExpr{
											X:   ast.NewIdent("e"),
											Sel: ast.NewIdent(field.Name),
										}
										keysLit.Elts = append(keysLit.Elts, newKey)
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

func isTypeDeclared(dir, typeName string) bool {
	found := false
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				// Silently ignore parsing errors in unrelated files
				return nil
			}

			ast.Inspect(node, func(n ast.Node) bool {
				if found {
					return false
				}
				switch x := n.(type) {
				case *ast.TypeSpec:
					if x.Name.Name == typeName {
						found = true
						return false
					}
				}
				return true
			})
		}
		if found {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil && err != filepath.SkipDir {
		log.Printf("Warning: Error walking directory %s: %v", dir, err)
	}

	return found
}

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func toCamelCase(str string) string {
	var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")
	return link.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}
