//go:build !js

package main

import (
	"app/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tariff the route file states must charge exactly what APICPUCredits charges, at every block
// boundary: the agent computes a write's credits from these numbers alone.
func TestFarewardRouteTariffMatchesAPICPUCredits(t *testing.T) {
	for _, tariff := range farewardRouteTariff {
		baseCredits, err := core.APICPUBaseCredits(tariff.method)
		if err != nil {
			t.Fatalf("%s: %v", tariff.method, err)
		}
		baseBytes, stepBytes := tariff.baseKiB*1024, tariff.stepKiB*1024
		for _, size := range []int{0, 1, baseBytes - 1, baseBytes, baseBytes + 1,
			baseBytes + stepBytes, baseBytes + stepBytes + 1, baseBytes + 7*stepBytes + 3} {
			expected, err := core.APICPUCredits(tariff.method, size)
			if err != nil {
				t.Fatalf("%s %d: %v", tariff.method, size, err)
			}
			steps := 0
			if size > baseBytes {
				steps = (size - baseBytes + stepBytes - 1) / stepBytes
			}
			if got := int(baseCredits) + steps; got != int(expected) {
				t.Fatalf("%s %d bytes: the route file charges %d, APICPUCredits %d",
					tariff.method, size, got, expected)
			}
		}
	}
}

// Every served route resolves into a row the agent can enforce, with the invariants the daemon's
// parser checks: no more than four slots, no zero slot, and no write left open.
func TestFarewardRouteTableCoversEveryHandler(t *testing.T) {
	handlers := *makeAppHandlers()
	routes, err := resolveFarewardRoutes(handlers)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != len(handlers) {
		t.Fatalf("%d routes for %d handlers", len(routes), len(handlers))
	}
	for _, route := range routes {
		name := route.method + "." + route.path
		if len(route.slots) > core.MaxRequiredAccess {
			t.Errorf("%s: %d slots", name, len(route.slots))
		}
		for _, slot := range route.slots {
			if slot == 0 {
				t.Errorf("%s: a zero slot", name)
			}
		}
		// A write the gate admits always names an access or is a self-service route.
		if isWriteMethod(route.method) && !route.public && route.deny == 0 &&
			len(route.slots) == 0 && !selfServiceRoutes[name] {
			t.Errorf("%s: a write with no access and no denial would be open", name)
		}
	}
}

// The written file is complete, and its version moves only when the policy does.
func TestFarewardRouteFileIsWrittenWhole(t *testing.T) {
	handlers := *makeAppHandlers()
	path := filepath.Join(t.TempDir(), "fareward_routes.toml")
	if err := writeFarewardRouteFile(path, handlers); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(first), "version = ") || !strings.Contains(string(first), "[[route]]") {
		t.Fatalf("unexpected file head: %.80s", first)
	}
	if err := writeFarewardRouteFile(path, handlers); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("the same catalogue produced two different files")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.tmp"))
	if len(leftovers) > 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}

	// The cross-language check: FAREWARD_ROUTES_DUMP=<path> go test -run TestFarewardRouteFile, then
	// FAREWARD_ROUTES_FILE_CHECK=<path> cargo test --lib -- --ignored a_generated_routes_file in
	// the fareward repository parses this exact file with the daemon's own parser.
	if dump := os.Getenv("FAREWARD_ROUTES_DUMP"); dump != "" {
		if err := os.WriteFile(dump, first, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
