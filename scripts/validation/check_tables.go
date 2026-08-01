package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"log"
	"os"
	"reflect"
	"strings"

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
	structFieldTags := make(map[*types.TypeName]map[string]string)
	schemasByTable := make(map[*types.TypeName]tableSchemaDeclaration)

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			astutil.Apply(file, func(cursor *astutil.Cursor) bool {
				node := cursor.Node()
				if funcDecl, ok := node.(*ast.FuncDecl); ok {
					if tableType, schema, found := readTableSchemaDeclaration(pkg, funcDecl); found {
						schemasByTable[tableType] = schema
					}
					return true
				}
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
						fieldTags := make(map[string]string)

						for i := 0; i < structType.NumFields(); i++ {
							field := structType.Field(i)
							fields = append(fields, field.Name())
							fieldTypes[field.Name()] = field.Type()
							fieldTags[field.Name()] = structType.Tag(i)

							if field.Embedded() {
								// Drivers export TableStruct as a generic alias, and Go 1.23+ models
								// aliases as *types.Alias — so this must unalias before asserting,
								// or the embedded field never matches at all.
								if named, ok := types.Unalias(field.Type()).(*types.Named); ok {
									// A table embeds TableStruct through a driver alias, so the resolved
									// type is db.TableStruct[Driver, XTable, XRecord]. Read the table and
									// record from the LAST two type arguments so the check keeps working
									// however many leading driver parameters the alias supplies.
									pkgPath := named.Obj().Pkg().Path()
									isTableStruct := named.Obj().Name() == "TableStruct" &&
										(pkgPath == "github.com/ivanjoz/genix-orm/db" ||
											pkgPath == "github.com/ivanjoz/genix-orm/scylla")
									if isTableStruct {
										typeArgs := named.TypeArgs()
										if argCount := typeArgs.Len(); argCount >= 2 {
											tableType := typeArgs.At(argCount - 2).(*types.Named).Obj()
											baseType := typeArgs.At(argCount - 1).(*types.Named).Obj()
											baseToTable[baseType] = tableType
										}
									}
								}
							}
						}
						structFields[typeName] = fields
						structFieldTypes[typeName] = fieldTypes
						structFieldTags[typeName] = fieldTags
					}
				}
				return true
			}, nil)
		}
	}

	// Reported so that "no output" cannot be mistaken for success when the embedded-type match
	// finds nothing at all (e.g. after the ORM's import path changes).
	fmt.Printf("Found %d table struct pairs.\n", len(baseToTable))

	checkTableSchemas(baseToTable, schemasByTable, structFieldTypes, structFieldTags)

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

					if isPrimitive && isColSlice {
						colSliceElementType := named.TypeArgs().At(1)
						if !types.Identical(elem, colSliceElementType) {
							fmt.Printf("Error: Inconsistent slice element type for '%s.%s'. Base is '%s', but ColSlice in '%s' has '%s'.\n", base.Name(), fieldName, elem.String(), table.Name(), colSliceElementType.String())
						}
					} else if !isPrimitive {
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

// maxTableID mirrors db.MaxTableID: TableSchema.ID occupies the low 14 bits of the packed
// partition_table_id key of cache_updated_version.
const maxTableID = 1<<14 - 1

// tableSchemaDeclaration is the subset of a GetSchema() literal these checks reason about. It is
// read from the AST rather than by running the code, so a bad declaration is caught before the
// ORM's own compile-time panic ever fires.
type tableSchemaDeclaration struct {
	tableName          string
	id                 int64
	hasID              bool
	saveUpdatedVersion bool
	hasDeltaIndex      bool
}

// readTableSchemaDeclaration returns the table struct a GetSchema() method belongs to, plus what
// its returned literal declares.
func readTableSchemaDeclaration(pkg *packages.Package, funcDecl *ast.FuncDecl) (*types.TypeName, tableSchemaDeclaration, bool) {
	schema := tableSchemaDeclaration{}
	if funcDecl.Name.Name != "GetSchema" || funcDecl.Recv == nil || funcDecl.Body == nil {
		return nil, schema, false
	}

	funcObj, _ := pkg.TypesInfo.Defs[funcDecl.Name].(*types.Func)
	if funcObj == nil {
		return nil, schema, false
	}
	signature, _ := funcObj.Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil {
		return nil, schema, false
	}
	receiverType := signature.Recv().Type()
	if pointer, isPointer := receiverType.(*types.Pointer); isPointer {
		receiverType = pointer.Elem()
	}
	named, isNamed := types.Unalias(receiverType).(*types.Named)
	if !isNamed {
		return nil, schema, false
	}

	literal := findReturnedCompositeLiteral(funcDecl.Body)
	if literal == nil {
		return nil, schema, false
	}

	for _, element := range literal.Elts {
		keyValue, isKeyValue := element.(*ast.KeyValueExpr)
		if !isKeyValue {
			continue
		}
		fieldName, isIdent := keyValue.Key.(*ast.Ident)
		if !isIdent {
			continue
		}

		switch fieldName.Name {
		case "ID":
			if value, ok := constantInt(pkg, keyValue.Value); ok {
				schema.id, schema.hasID = value, true
			}
		case "Name":
			schema.tableName = constantString(pkg, keyValue.Value)
		case "SaveUpdatedVersion":
			schema.saveUpdatedVersion = constantBool(pkg, keyValue.Value)
		case "Indexes":
			schema.hasDeltaIndex = declaresDeltaIndex(keyValue.Value)
		}
	}

	return named.Obj(), schema, true
}

func findReturnedCompositeLiteral(body *ast.BlockStmt) *ast.CompositeLit {
	var found *ast.CompositeLit
	ast.Inspect(body, func(node ast.Node) bool {
		returnStmt, isReturn := node.(*ast.ReturnStmt)
		if !isReturn || len(returnStmt.Results) != 1 || found != nil {
			return true
		}
		if literal, isLiteral := returnStmt.Results[0].(*ast.CompositeLit); isLiteral {
			found = literal
		}
		return true
	})
	return found
}

// declaresDeltaIndex reports whether any entry of an Indexes literal sets Type to a TypeDelta
// selector. Matching on the identifier keeps this independent of which package alias is used.
func declaresDeltaIndex(indexesValue ast.Expr) bool {
	found := false
	ast.Inspect(indexesValue, func(node ast.Node) bool {
		keyValue, isKeyValue := node.(*ast.KeyValueExpr)
		if !isKeyValue {
			return true
		}
		if key, isIdent := keyValue.Key.(*ast.Ident); !isIdent || key.Name != "Type" {
			return true
		}
		if selector, isSelector := keyValue.Value.(*ast.SelectorExpr); isSelector && selector.Sel.Name == "TypeDelta" {
			found = true
		}
		if ident, isIdent := keyValue.Value.(*ast.Ident); isIdent && ident.Name == "TypeDelta" {
			found = true
		}
		return true
	})
	return found
}

func constantInt(pkg *packages.Package, expr ast.Expr) (int64, bool) {
	value := pkg.TypesInfo.Types[expr].Value
	if value == nil {
		return 0, false
	}
	return constant.Int64Val(constant.ToInt(value))
}

func constantString(pkg *packages.Package, expr ast.Expr) string {
	value := pkg.TypesInfo.Types[expr].Value
	if value == nil || value.Kind() != constant.String {
		return ""
	}
	return constant.StringVal(value)
}

func constantBool(pkg *packages.Package, expr ast.Expr) bool {
	value := pkg.TypesInfo.Types[expr].Value
	return value != nil && value.Kind() == constant.Bool && constant.BoolVal(value)
}

// checkTableSchemas enforces the declaration rules the ORM would otherwise only discover at
// runtime: every table owns a unique ID, and any table synced incrementally exposes its write
// sequence number to the client.
func checkTableSchemas(
	baseToTable map[*types.TypeName]*types.TypeName,
	schemasByTable map[*types.TypeName]tableSchemaDeclaration,
	structFieldTypes map[*types.TypeName]map[string]types.Type,
	structFieldTags map[*types.TypeName]map[string]string,
) {
	tableNamesByID := map[int64]string{}

	for base, table := range baseToTable {
		schema, hasSchema := schemasByTable[table]
		if !hasSchema {
			continue
		}
		reportedName := schema.tableName
		if reportedName == "" {
			reportedName = table.Name()
		}

		switch {
		case !schema.hasID:
			fmt.Printf("Error: Table '%s' does not declare TableSchema.ID. Assign it the next free ID (1..%d).\n",
				reportedName, maxTableID)
		case schema.id <= 0 || schema.id > maxTableID:
			fmt.Printf("Error: Table '%s' declares TableSchema.ID %d, outside 1..%d.\n",
				reportedName, schema.id, maxTableID)
		default:
			if claimedBy, isClaimed := tableNamesByID[schema.id]; isClaimed {
				fmt.Printf("Error: TableSchema.ID %d is declared by two tables: '%s' and '%s'.\n",
					schema.id, claimedBy, reportedName)
			} else {
				tableNamesByID[schema.id] = reportedName
			}
		}

		// The by-IDs cache compares slot versions and a delta sync watermarks on the write sequence,
		// so both need "updated_version" to reach the client as the "upv" field.
		if schema.saveUpdatedVersion || schema.hasDeltaIndex {
			if !hasUpdatedVersionField(structFieldTypes[base], structFieldTags[base]) {
				fmt.Printf("Error: Table '%s' is synced incrementally but its record struct '%s' does not declare "+
					"the field 'UpdatedVersion int32' with the json tag 'upv,omitempty'.\n", reportedName, base.Name())
			}
			if _, declared := structFieldTypes[table]["UpdatedVersion"]; !declared {
				fmt.Printf("Error: Table '%s' is synced incrementally but its table struct '%s' does not declare "+
					"the 'UpdatedVersion' column.\n", reportedName, table.Name())
			}
		}

		// The slot version replaced the old per-record cache version outright.
		for fieldName, tag := range structFieldTags[base] {
			if fieldName == "CacheVersion" || jsonTagName(tag) == "ccv" {
				fmt.Printf("Error: Record '%s' still declares the removed cache-version field '%s'. Use "+
					"'UpdatedVersion' with the json tag 'upv' instead.\n", base.Name(), fieldName)
			}
		}
	}
}

func hasUpdatedVersionField(fieldTypes map[string]types.Type, fieldTags map[string]string) bool {
	fieldType, declared := fieldTypes["UpdatedVersion"]
	if !declared {
		return false
	}
	if basic, isBasic := fieldType.Underlying().(*types.Basic); !isBasic || basic.Kind() != types.Int32 {
		return false
	}
	return jsonTagName(fieldTags["UpdatedVersion"]) == "upv"
}

func jsonTagName(rawTag string) string {
	jsonTag := reflect.StructTag(rawTag).Get("json")
	if jsonTag == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(jsonTag, ",")[0])
}
