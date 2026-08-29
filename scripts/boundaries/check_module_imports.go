// check_module_imports enforces the backend layering rule from
// backend/docs/MODULE_BOUNDARIES.md: a module body may never import another module body.
// Shared code crosses module lines through `<module>/types` only.
//
// A rule that is not checked decays, and this one is invisible at compile time — Go is
// perfectly happy with `accounting` importing `finance`.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Layers, lowest first. A package may import its own layer and anything below it, except
// where a rule below says otherwise. Kept as literal path lists rather than inferred, so
// the policy is greppable and a new package fails loudly instead of being guessed at.
var (
	// L0/L1: foundation and core. No app-level business logic depends on module state.
	foundationPrefixes = []string{"app/db", "app/libs", "app/core"}

	// L3: infrastructure services. May use any */types but no module body.
	infrastructurePackages = map[string]bool{"app/cloud": true}

	// L4: module bodies. These are the packages the rule is about.
	moduleBodies = map[string]bool{
		"app/accounting": true, "app/agent": true, "app/business": true,
		"app/config": true, "app/crm": true, "app/finance": true,
		"app/invoicing": true, "app/logistics": true, "app/production": true,
		"app/sales": true, "app/security": true, "app/webpage": true,
	}

	// L5: composition roots. These wire the modules together and may import anything.
	compositionRootPrefixes = []string{"app/exec", "app/tests", "main"}
)

type goPackage struct {
	ImportPath  string
	Name        string
	Imports     []string
	TestImports []string
}

func main() {
	packages, err := loadBackendPackages()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	violations := []string{}
	for _, pkg := range packages {
		for _, imported := range allImports(pkg) {
			if reason := violation(pkg.ImportPath, imported); reason != "" {
				violations = append(violations, fmt.Sprintf("  %s\n      imports %s\n      %s", pkg.ImportPath, imported, reason))
			}
		}
	}
	sort.Strings(violations)

	fmt.Printf("Checked %d backend packages.\n", len(packages))
	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d module boundary violation(s):\n\n%s\n\nSee backend/docs/MODULE_BOUNDARIES.md\n",
			len(violations), strings.Join(violations, "\n\n"))
		os.Exit(1)
	}
	fmt.Println("No module boundary violations.")
}

// violation returns why the edge is illegal, or "" when it is allowed.
func violation(from, to string) string {
	if !strings.HasPrefix(to, "app/") || from == to {
		return ""
	}
	// Composition roots wire the modules together; that is their whole job.
	if hasAnyPrefix(from, compositionRootPrefixes) {
		return ""
	}
	// Anything may import foundation, core, and any */types.
	if hasAnyPrefix(to, foundationPrefixes) || isTypesPackage(to) {
		return ""
	}
	// A module's own subpackages are internal to it.
	if strings.HasPrefix(to, from+"/") || strings.HasPrefix(from, to+"/") {
		return ""
	}
	fromModule, toModule := moduleOf(from), moduleOf(to)
	if fromModule == toModule {
		return ""
	}

	// A */types package must stay a leaf. Everything imports it, so if it could reach a module
	// body — or even infrastructure — every module could route around the rule through it.
	// Checked before the infrastructure allowance below, which is the looser rule.
	if isTypesPackage(from) {
		return "a */types package must stay a leaf (db, core and other */types only)"
	}
	// Infrastructure sits below the module bodies.
	if infrastructurePackages[to] {
		return ""
	}
	if moduleBodies[toModule] {
		return fmt.Sprintf("a module body may only be imported by a composition root; move the shared code into %s/types", toModule)
	}
	return ""
}

func isTypesPackage(importPath string) bool {
	return strings.HasSuffix(importPath, "/types")
}

// moduleOf returns the top-level module an app package belongs to: app/agent/llm -> app/agent.
func moduleOf(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) < 2 {
		return importPath
	}
	return parts[0] + "/" + parts[1]
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if value == prefix || strings.HasPrefix(value, prefix+"/") {
			return true
		}
	}
	return false
}

func allImports(pkg goPackage) []string {
	return append(append([]string{}, pkg.Imports...), pkg.TestImports...)
}

func loadBackendPackages() ([]goPackage, error) {
	backendDir := "backend"
	if _, err := os.Stat(backendDir); os.IsNotExist(err) {
		backendDir = "../backend"
	}
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = backendDir
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed in %s: %w", backendDir, err)
	}

	packages := []goPackage{}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		pkg := goPackage{}
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}
