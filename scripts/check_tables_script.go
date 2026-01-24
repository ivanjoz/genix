package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

func CheckTables() {
	fmt.Println("Checking tables...")

	cfg := &packages.Config{
		Mode:  packages.LoadSyntax | packages.LoadTypes,
		Dir:   "../backend",
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

	// Now, let's check for consistency
	for base, table := range baseToTable {
		// Check if the table struct name is valid
		if table.Name() != base.Name()+"Table" {
			fmt.Printf("Error: Inconsistent table struct name for %s. Expected %s, but got %s\n", base.Name(), base.Name()+"Table", table.Name())
		}

		// Check if the fields are consistent
		baseFields, ok := structFields[base]
		if !ok {
			fmt.Printf("Error: Could not find fields for base struct %s\n", base.Name())
			continue
		}

		tableFields, ok := structFields[table]
		if !ok {
			fmt.Printf("Error: Could not find fields for table struct %s\n", table.Name())
			continue
		}

		if len(baseFields) != len(tableFields) {
			fmt.Printf("Error: Inconsistent number of fields for %s and %s\n", base.Name(), table.Name())
			continue
		}

		baseFieldTypes := structFieldTypes[base]
		tableFieldTypes := structFieldTypes[table]

		for i, fieldName := range baseFields {
			if fieldName != tableFields[i] {
				fmt.Printf("Error: Inconsistent field name at index %d for %s and %s. Expected %s, but got %s\n", i, base.Name(), table.Name(), fieldName, tableFields[i])
			}

			// Check if the types are consistent
			baseFieldType := baseFieldTypes[fieldName]
			tableFieldType := tableFieldTypes[fieldName]

			if named, ok := tableFieldType.(*types.Named); ok {
				var expectedType types.Type
				if named.Obj().Name() == "Col" {
					if typeArgs := named.TypeArgs(); typeArgs.Len() == 2 {
						expectedType = typeArgs.At(1)
					}
				} else if named.Obj().Name() == "ColSlice" {
					if typeArgs := named.TypeArgs(); typeArgs.Len() == 2 {
						sliceType := typeArgs.At(1)
						expectedType = types.NewSlice(sliceType)
					}
				}

				if expectedType != nil && !types.Identical(expectedType, baseFieldType) {
					fmt.Printf("Error: Inconsistent field type for %s.%s. Expected %s, but got %s\n", base.Name(), fieldName, expectedType.String(), baseFieldType.String())
				}
			}
		}
	}
}
