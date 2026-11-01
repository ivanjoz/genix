//go:build !js

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// forbiddenBrowserPackages are the server-only stacks the browser build (GOOS=js GOARCH=wasm) must
// never link: they cannot run in a tab, and AWS alone was a third of the binary. A file that imports
// one of them belongs behind //go:build !js (docs/WASM_LOCAL_MODE_PLAN.md, phase 1).
var forbiddenBrowserPackages = []string{
	"github.com/aws/",
	"github.com/gocql/",
	"github.com/ivanjoz/genix-orm/scylla",
	"github.com/ivanjoz/genix-orm/dynamo",
	"github.com/ivanjoz/genix-orm/search/genixs3",
}

func TestBrowserBuildLinksNoServerStack(t *testing.T) {
	listCommand := exec.Command("go", "list", "-deps", ".")
	listCommand.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	output, err := listCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("go list for GOOS=js failed: %v\n%s", err, output)
	}

	for _, importPath := range strings.Fields(string(output)) {
		for _, forbiddenPrefix := range forbiddenBrowserPackages {
			if strings.HasPrefix(importPath, forbiddenPrefix) {
				t.Errorf("the browser build links %s; find its importer with: GOOS=js GOARCH=wasm go list -deps -f '{{.ImportPath}}: {{.Imports}}' . | rg %s",
					importPath, forbiddenPrefix)
			}
		}
	}
}
