package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShippingBuildTagsAgreeAcrossDeployPaths catches the build-tag lists drifting apart. They are
// spread across Go modules and a CI workflow, so nothing else relates them. Dropping lambda.norpc
// from any one path still compiles and passes every other test -- the binary just grows ~12 MB.
// Any new path that produces a shipping binary belongs in this list.
func TestShippingBuildTagsAgreeAcrossDeployPaths(t *testing.T) {
	for _, buildPath := range []string{
		filepath.Join("..", "scripts", "deploy_vps.go"),                     // VPS deploy
		filepath.Join("..", ".github", "workflows", "release-binaries.yml"), // GitHub release
	} {
		contents, err := os.ReadFile(buildPath)
		if err != nil {
			t.Errorf("%s: %v", buildPath, err)
			continue
		}
		// Matched bare rather than quoted: the Go file quotes it, the YAML workflow does not.
		if !strings.Contains(string(contents), BuildTags) {
			t.Errorf("%s does not declare the shipping build tags %q that cloud/main.go uses", buildPath, BuildTags)
		}
	}
}
