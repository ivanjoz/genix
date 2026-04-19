package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"os"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

func main() {
	CheckTables()
}

func CheckTables() {
	fmt.Println("Checking tables...")

	backendDir := "backend"
	if _, err := os.Stat(backendDir); os.IsNotExist(err) {
		backendDir = "../backend"
	}

	cfg := &packages.Config{
		Mode:  packages.LoadSyntax | packages.LoadTypes,
		Dir:   backendDir,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatal(err)
	}

	baseToTable := make(map[*types.TypeName]*types.TypeName)
	structFields := make(map[*types.TypeName][]string)
	structFieldTypes := make(map[*types.TypeName]map[string]types.Type)

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			astutil.Apply(file, func(cursor *astutil.Cursor) bool {
				node := cursor.Node()
				if ts, ok := node.(*ast.TypeSpec); ok {
					if _, ok := ts.Type.(*ast.StructType); ok {
						obj := pkg.TypesInfo.Defs[ts.Name]
						if obj == nil {
							return true
						}
						typeName := obj.(*types.TypeName)
						named := typeName.Type().(*types.Named)
						structType := named.Underlying().(*types.Struct)

						var fields []string
						fieldTypes := make(map[string]types.Type)

						for i := 0; i < structType.NumFields(); i++ {
							field := structType.Field(i)
							fields = append(fields, field.Name())
							fieldTypes[field.Name()] = field.Type()

							if field.Embedded() {
								if named, ok := field.Type().(*types.Named); ok {
									if named.Obj().Pkg().Path() == "app/db" && named.Obj().Name() == "TableStruct" {
										if typeArgs := named.TypeArgs(); typeArgs.Len() == 2 {
											tableType := typeArgs.At(0).(*types.Named).Obj()
											baseType := typeArgs.At(1).(*types.Named).Obj()
											baseToTable[baseType] = tableType
										}
									}
								}
							}
						}
						structFields[typeName] = fields
						structFieldTypes[typeName] = fieldTypes
					}
				}
				return true
			}, nil)
		}
	}

	for base, table := range baseToTable {
		if table.Name() != base.Name()+"Table" {
			fmt.Printf("Error: Inconsistent table struct name for %s. Expected %s, but got %s\n", base.Name(), base.Name()+"Table", table.Name())
		}

		baseFieldTypes, ok := structFieldTypes[base]
		if !ok {
			fmt.Printf("Error: Could not find fields for base struct %s\n", base.Name())
			continue
		}

		tableFields, ok := structFields[table]
		if !ok {
			fmt.Printf("Error: Could not find fields for table struct %s\n", table.Name())
			continue
		}
		tableFieldTypes := structFieldTypes[table]

		for _, fieldName := range tableFields {
			if fieldName == "TableStruct" {
				continue
			}

			baseFieldType, ok := baseFieldTypes[fieldName]
			if !ok {
				fmt.Printf("Error: Field '%s' from table struct '%s' does not exist in base struct '%s'\n", fieldName, table.Name(), base.Name())
				continue
			}

			tableFieldType := tableFieldTypes[fieldName]

			if named, ok := tableFieldType.(*types.Named); ok {
				isCol := named.Obj().Name() == "Col"
				isColSlice := named.Obj().Name() == "ColSlice"

				if !isCol && !isColSlice {
					continue
				}

				if slice, ok := baseFieldType.(*types.Slice); ok {
					elem := slice.Elem()
					isPrimitive := false
					if _, ok := elem.Underlying().(*types.Basic); ok {
						isPrimitive = true
					} else if n, ok := elem.Underlying().(*types.Named); ok {
						if _, ok := n.Underlying().(*types.Basic); ok {
							isPrimitive = true
						}
					}

					if isPrimitive {
						if !isColSlice {
							fmt.Printf("Error: Field '%s.%s' is a primitive slice. Use db.ColSlice in table struct '%s', not db.Col.\n", base.Name(), fieldName, table.Name())
							continue
						}
						colSliceElementType := named.TypeArgs().At(1)
						if !types.Identical(elem, colSliceElementType) {
							fmt.Printf("Error: Inconsistent slice element type for '%s.%s'. Base is '%s', but ColSlice in '%s' has '%s'.\n", base.Name(), fieldName, elem.String(), table.Name(), colSliceElementType.String())
						}
					} else {
						if !isCol {
							fmt.Printf("Error: Field '%s.%s' is a complex slice. Use db.Col in table struct '%s', not db.ColSlice.\n", base.Name(), fieldName, table.Name())
							continue
						}
						colType := named.TypeArgs().At(1)
						if !types.Identical(baseFieldType, colType) {
							fmt.Printf("Error: Inconsistent type for '%s.%s'. Base is '%s', but Col in '%s' has '%s'.\n", base.Name(), fieldName, baseFieldType.String(), table.Name(), colType.String())
						}
					}
				} else {
					if isColSlice {
						fmt.Printf("Error: Field '%s.%s' is not a slice, but table struct '%s' uses db.ColSlice. Use db.Col instead.\n", base.Name(), fieldName, table.Name())
						continue
					}
					colType := named.TypeArgs().At(1)
					if !types.Identical(baseFieldType, colType) {
						fmt.Printf("Error: Inconsistent type for '%s.%s'. Base is '%s', but Col in '%s' has '%s'.\n", base.Name(), fieldName, baseFieldType.String(), table.Name(), colType.String())
					}
				}
			}
		}
	}
}
