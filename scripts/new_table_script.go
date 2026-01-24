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

func NewTable(args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: new-table <output_path> <table_name> [field:type:key]...")
	}

	outputPath := args[0]
	tableNameSnake := args[1]
	tableNameCamel := toCamelCase(tableNameSnake)

	// Check if the type is already declared
	if isTypeDeclared("../backend", tableNameCamel) {
		log.Fatalf("Error: Type %s is already declared in the backend.", tableNameCamel)
	}
	if isTypeDeclared("../backend", tableNameCamel+"Table") {
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
				if found { return false }
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
