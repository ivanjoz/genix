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

func NewTableColumn(args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: new_table_column <table_name> field:type[:key]...")
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
	modified := false
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
				if starExpr, ok := x.Recv.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						if ident.Name == tableNameCamel+"Table" {
							// Update GetSchema to add new key if needed
							if field.IsKey {
								updateGetSchemaKeys(x.Body, field)
								fmt.Printf("Updated GetSchema for field %s as clustering key\n", field.Name)
							}
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
	typeParams := []ast.Expr{
		ast.NewIdent(tableName + "Table"),
		ast.NewIdent(elemType),
	}

	fieldType := &ast.IndexExpr{
		X: &ast.SelectorExpr{
			X:   ast.NewIdent("db"),
			Sel: ast.NewIdent(colType),
		},
		Index: &ast.CompositeLit{
			Elts: typeParams,
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
				if callExpr, ok := returnStmt.Results[0].(*ast.CallExpr); ok {
					if len(callExpr.Args) > 0 {
						if compositeLit, ok := callExpr.Args[0].(*ast.CompositeLit); ok {
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
			}
		}
		return true
	})
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