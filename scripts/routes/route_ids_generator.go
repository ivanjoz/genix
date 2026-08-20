// Assigns one stable number to every API route and writes them to a generated Go file.
//
// A route travels as a string everywhere in the backend, which is right for a router and wrong
// for a log line and a database column: "POST.almacen-producto-stock" costs 27 bytes in every
// CloudWatch entry and cannot be packed into an aggregation key at all. This generator is what
// turns that string into an int16, once, at build time.
//
// The one invariant: an ID is handed out once and never reused. A route that disappears keeps its
// number — annotated "retired" — because rows written last month still name it, and recycling the
// number onto a different route would silently rewrite that history.
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
	"strconv"
	"strings"
)

// Output file relative to the project root, and the package it declares.
const (
	routesOutputPath    = "backend/core/api_routes.generated.go"
	routesOutputPackage = "core"
)

// The map variable read back on the next run to recover what has already been assigned. The
// generated file is its own registry: no second source of truth to drift out of sync.
const routeIDsVariableName = "APIRouteIDs"

// routeAssignment is one route and the number it owns. Retired means the route was not found in
// this scan; the entry stays so its ID can never be handed to something else.
type routeAssignment struct {
	Route   string
	ID      int16
	Retired bool
}

func main() {
	checkOnly := false
	for _, argument := range os.Args[1:] {
		if argument == "--check" {
			checkOnly = true
		}
	}

	projectRoot, err := findProjectRootDir()
	if err != nil {
		exitWith(err)
	}

	declaredRoutes, err := scanModuleHandlerRoutes(filepath.Join(projectRoot, "backend"))
	if err != nil {
		exitWith(err)
	}
	if len(declaredRoutes) == 0 {
		exitWith(fmt.Errorf("no routes found: every ModuleHandlers map is empty or the scan is broken"))
	}

	outputPath := filepath.Join(projectRoot, routesOutputPath)
	existing, err := readExistingAssignments(outputPath)
	if err != nil {
		exitWith(err)
	}

	assignments, newlyAssigned, err := mergeAssignments(existing, declaredRoutes)
	if err != nil {
		exitWith(err)
	}
	source, err := renderRoutesFile(assignments)
	if err != nil {
		exitWith(err)
	}

	if checkOnly {
		current, readErr := os.ReadFile(outputPath)
		if readErr != nil || !bytes.Equal(current, source) {
			fmt.Fprintf(os.Stderr,
				"%s is stale: run \"go run . generate_route_ids\" and commit the result\n", routesOutputPath)
			os.Exit(1)
		}
		fmt.Printf("%s is current (%d routes)\n", routesOutputPath, len(assignments))
		return
	}

	if err := os.WriteFile(outputPath, source, 0644); err != nil {
		exitWith(err)
	}
	fmt.Printf("generated %s (%d routes, %d newly assigned)\n",
		routesOutputPath, len(assignments), newlyAssigned)
}

// maxEncodableRouteID is the ceiling of the fourteen-bit route field in the credit usage blob
// header, mirrored from server_utils/src/limiter/credits_blob.rs and
// backend/core/server_utils/credits.go. Numbers are never reused, so the count only ever climbs;
// failing here is the one place that can say so before a route exists that cannot be charged.
const maxEncodableRouteID = int16(16_383)

// mergeAssignments keeps every ID already handed out and numbers the rest from the current
// maximum. New routes are sorted before being numbered so two branches adding routes in the same
// week produce the same file, in the same order, whatever order the filesystem walk returned.
func mergeAssignments(existing map[string]int16, declared map[string]bool) ([]routeAssignment, int, error) {
	assignments := make([]routeAssignment, 0, len(existing)+len(declared))
	highestID := int16(0)
	for route, id := range existing {
		assignments = append(assignments, routeAssignment{Route: route, ID: id, Retired: !declared[route]})
		if id > highestID {
			highestID = id
		}
	}

	unassigned := []string{}
	for route := range declared {
		if _, alreadyAssigned := existing[route]; !alreadyAssigned {
			unassigned = append(unassigned, route)
		}
	}
	sort.Strings(unassigned)

	for _, route := range unassigned {
		if highestID >= maxEncodableRouteID {
			return nil, 0, fmt.Errorf(
				"route %q would be numbered past %d, the widest ID the credit usage blob header can "+
					"hold; widen the header in server_utils/src/limiter/credits_blob.rs and both "+
					"decoders before adding it", route, maxEncodableRouteID)
		}
		highestID++
		assignments = append(assignments, routeAssignment{Route: route, ID: highestID})
	}

	// Alphabetical in the file, so a new route lands next to its neighbours and the diff is one
	// line rather than a renumbered block.
	sort.Slice(assignments, func(first, second int) bool {
		return assignments[first].Route < assignments[second].Route
	})
	return assignments, len(unassigned), nil
}

// scanModuleHandlerRoutes walks backend/ and collects the keys of every
// `var ModuleHandlers = core.AppRouterType{...}` literal. Parsing the source rather than running
// the binary keeps the generator usable while the backend does not compile, which is exactly when
// a new route is being added.
func scanModuleHandlerRoutes(backendDir string) (map[string]bool, error) {
	routes := map[string]bool{}

	err := filepath.WalkDir(backendDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" || name == "genix-orm" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileSet := token.NewFileSet()
		parsedFile, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		for _, declaration := range parsedFile.Decls {
			genericDecl, isGeneric := declaration.(*ast.GenDecl)
			if !isGeneric || genericDecl.Tok != token.VAR {
				continue
			}
			for _, spec := range genericDecl.Specs {
				valueSpec, isValue := spec.(*ast.ValueSpec)
				if !isValue || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
					continue
				}
				if valueSpec.Names[0].Name != "ModuleHandlers" {
					continue
				}
				literal, isLiteral := valueSpec.Values[0].(*ast.CompositeLit)
				if !isLiteral {
					continue
				}
				for _, element := range literal.Elts {
					keyValue, isKeyValue := element.(*ast.KeyValueExpr)
					if !isKeyValue {
						continue
					}
					key, isString := stringLiteral(keyValue.Key)
					if !isString {
						return fmt.Errorf("%s: a ModuleHandlers key is not a string literal", path)
					}
					routes[key] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return routes, nil
}

// readExistingAssignments recovers the IDs already handed out. A missing file is the first run,
// not an error; anything else must fail loudly, because silently starting over at 1 would
// reassign every number in the project.
func readExistingAssignments(outputPath string) (map[string]int16, error) {
	assignments := map[string]int16{}

	source, err := os.ReadFile(outputPath)
	if os.IsNotExist(err) {
		return assignments, nil
	}
	if err != nil {
		return nil, err
	}

	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, outputPath, source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", outputPath, err)
	}

	for _, declaration := range parsedFile.Decls {
		genericDecl, isGeneric := declaration.(*ast.GenDecl)
		if !isGeneric || genericDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genericDecl.Specs {
			valueSpec, isValue := spec.(*ast.ValueSpec)
			if !isValue || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			if valueSpec.Names[0].Name != routeIDsVariableName {
				continue
			}
			literal, isLiteral := valueSpec.Values[0].(*ast.CompositeLit)
			if !isLiteral {
				continue
			}
			for _, element := range literal.Elts {
				keyValue, isKeyValue := element.(*ast.KeyValueExpr)
				if !isKeyValue {
					continue
				}
				route, isString := stringLiteral(keyValue.Key)
				if !isString {
					return nil, fmt.Errorf("%s: a %s key is not a string literal", outputPath, routeIDsVariableName)
				}
				basicLit, isBasic := keyValue.Value.(*ast.BasicLit)
				if !isBasic || basicLit.Kind != token.INT {
					return nil, fmt.Errorf("%s: route %q has a non-literal ID", outputPath, route)
				}
				id, parseErr := strconv.ParseInt(basicLit.Value, 10, 16)
				if parseErr != nil {
					return nil, fmt.Errorf("%s: route %q has an unreadable ID: %w", outputPath, route, parseErr)
				}
				if id <= 0 {
					return nil, fmt.Errorf("%s: route %q has ID %d; zero means \"unknown\" and must not be assigned",
						outputPath, route, id)
				}
				assignments[route] = int16(id)
			}
		}
	}
	return assignments, nil
}

func renderRoutesFile(assignments []routeAssignment) ([]byte, error) {
	highestID := int16(0)
	for _, assignment := range assignments {
		if assignment.ID > highestID {
			highestID = assignment.ID
		}
	}

	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, `// Code generated by "cd scripts && go run . generate_route_ids". DO NOT EDIT.

package %s

// APIRouteIDs numbers every API route so it can travel as two bytes instead of a path: once in a
// log prefix, once inside the packed aggregation key of user_logs.
//
// An ID is assigned once and never reused. A route that no longer exists keeps its number,
// annotated "retired", because rows already written still name it and handing that number to a
// different route would rewrite what those rows mean.
var %s = map[string]int16{
`, routesOutputPackage, routeIDsVariableName)

	for _, assignment := range assignments {
		fmt.Fprintf(buffer, "\t%q: %d,", assignment.Route, assignment.ID)
		if assignment.Retired {
			buffer.WriteString(" // retired")
		}
		buffer.WriteString("\n")
	}

	fmt.Fprintf(buffer, `}

// APIRouteNames reads an ID back into its route, for the dashboard and for anyone holding a log
// line with an "r118" token in it. Retired routes are included, which is the only thing that keeps
// an old row readable.
//
// Inverted at start rather than generated as a second literal: two maps in one file are two things
// that can disagree, and this one is exactly the first one backwards.
var APIRouteNames = make(map[int16]string, len(%s))

func init() {
	for route, id := range %s {
		APIRouteNames[id] = route
	}
}

// MaxAPIRouteID is the highest number handed out so far, retired routes included.`,
		routeIDsVariableName, routeIDsVariableName)

	fmt.Fprintf(buffer, `
const MaxAPIRouteID int16 = %d

// APIRouteID resolves a "METHOD.route" path to its number. Zero means unknown — a 404, or a route
// added since the last generation — and is never a valid assignment.
func APIRouteID(funcPath string) int16 {
	return %s[funcPath]
}
`, highestID, routeIDsVariableName)

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generated source does not parse: %w", err)
	}
	return formatted, nil
}

func stringLiteral(expr ast.Expr) (string, bool) {
	basicLit, isBasic := expr.(*ast.BasicLit)
	if !isBasic || basicLit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(basicLit.Value)
	if err != nil {
		return "", false
	}
	return value, true
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

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func exitWith(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
