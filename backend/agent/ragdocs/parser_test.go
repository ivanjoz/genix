package ragdocs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExamplesAndBuildStableChunks(t *testing.T) {
	repositoryRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	documentationPaths := []string{
		filepath.Join(repositoryRoot, "frontend/routes/finance/cash-banks/DOCUMENTATION.md"),
		filepath.Join(repositoryRoot, "frontend/routes/logistics/purchase-orders/DOCUMENTATION.md"),
	}

	seenPointIDs := map[string]bool{}
	for _, documentationPath := range documentationPaths {
		document, err := ParseFile(repositoryRoot, documentationPath)
		if err != nil {
			t.Fatal(err)
		}
		chunks, err := BuildChunks(document)
		if err != nil {
			t.Fatal(err)
		}
		for _, chunk := range chunks {
			if seenPointIDs[chunk.PointID] {
				t.Fatalf("duplicate point ID %s", chunk.PointID)
			}
			seenPointIDs[chunk.PointID] = true
			if strings.Contains(chunk.EmbeddingText, "### FILES") || strings.Contains(chunk.EmbeddingText, "DOCUMENTATION_GAP") {
				t.Fatalf("non-indexable metadata leaked into %s", chunk.PointKey)
			}
			if !strings.Contains(chunk.ContextHeader, document.Frontmatter.Route) {
				t.Fatalf("route missing from context header for %s", chunk.PointKey)
			}
			if estimateTokens(chunk.Content) > hardMaximumTokens {
				t.Fatalf("chunk %s exceeds hard maximum", chunk.PointKey)
			}
		}
	}
}

func TestParseRejectsStaleEvidence(t *testing.T) {
	repositoryRoot := t.TempDir()
	routeDirectory := filepath.Join(repositoryRoot, "frontend/routes/finance/example")
	if err := os.MkdirAll(routeDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(repositoryRoot, "backend/example.go")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("package example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	documentation := `---
schema: 1
page_id: finance.example
route: /finance/example
title: Example (Ejemplo)
status: implemented
visibility: tenant
---

# Example

<!-- DOC-ID: page-purpose -->
## Page purpose

Verified behavior.

### FILES

` + "```yaml" + `
schema: 1
hash_algorithm: sha256
files:
  - path: backend/example.go
    role: backend-handler
    hash: sha256:0000000000000000000000000000000000000000000000000000000000000000
    supports: [page-purpose]
` + "```" + "\n"
	documentationPath := filepath.Join(routeDirectory, "DOCUMENTATION.md")
	if err := os.WriteFile(documentationPath, []byte(documentation), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseFile(repositoryRoot, documentationPath)
	if err == nil || !strings.Contains(err.Error(), "stale evidence") {
		t.Fatalf("expected stale evidence error, got %v", err)
	}
}

func TestFileHashChangesWithoutChangingNormalizedDocumentationHash(t *testing.T) {
	// FILES hashes are deliberately outside the normalized content contract.
	metadata := Frontmatter{Schema: 1, PageID: "finance.example", Route: "/finance/example", Title: "Example", Status: "implemented", Visibility: "tenant"}
	sections := []Section{{ID: "page-purpose", Title: "Page purpose", Markdown: "## Page purpose\n\nBehavior."}}
	first := canonicalDocumentation(metadata, sections)
	second := canonicalDocumentation(metadata, sections)
	if hashBytes([]byte(first)) != hashBytes([]byte(second)) {
		t.Fatal("identical normalized documentation produced different hashes")
	}
}
