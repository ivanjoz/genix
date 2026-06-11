package exec

import (
	"app/core"
	"os"
	"path/filepath"
	"testing"
)

func TestParseCompanyIDArgument(t *testing.T) {
	companyID, parseError := parseCompanyIDArgument("123")
	if parseError != nil || companyID != 123 {
		t.Fatalf("expected CompanyID 123, got %d, error=%v", companyID, parseError)
	}

	invalidArguments := []string{"", "0", "-1", "abc", "1 2"}
	for _, invalidArgument := range invalidArguments {
		if _, parseError := parseCompanyIDArgument(invalidArgument); parseError == nil {
			t.Fatalf("expected %q to fail", invalidArgument)
		}
	}
}

func TestCompanyWebpageAssetBase(t *testing.T) {
	previousEnv := core.Env
	core.Env = &core.EnvStruct{FRONTEND_CDN: "https://cdn.example.com/"}
	t.Cleanup(func() { core.Env = previousEnv })

	assetBase, assetBaseError := companyWebpageAssetBase(123)
	if assetBaseError != nil {
		t.Fatal(assetBaseError)
	}
	if assetBase != "https://cdn.example.com/websites/123" {
		t.Fatalf("unexpected asset base: %s", assetBase)
	}
}

func TestWebpageAssetFingerprintDetection(t *testing.T) {
	fingerprintedAssets := []string{"app.ABCdef12.js", "ABCdef12.js", "styles.ABCdef12.css"}
	for _, fileName := range fingerprintedAssets {
		if !isFingerprintedWebpageAsset(fileName) {
			t.Fatalf("expected fingerprinted asset: %s", fileName)
		}
	}

	mutableAssets := []string{"env.js", "blurhash.js", "app.js", "styles.css"}
	for _, fileName := range mutableAssets {
		if isFingerprintedWebpageAsset(fileName) {
			t.Fatalf("expected mutable asset: %s", fileName)
		}
	}
}

func TestNormalizeAndValidateHostname(t *testing.T) {
	hostname, normalizeError := normalizeAndValidateHostname(" HTTPS://Store.UN.PE/path ")
	if normalizeError != nil || hostname != "store.un.pe" {
		t.Fatalf("unexpected hostname=%q error=%v", hostname, normalizeError)
	}

	invalidHostnames := []string{"localhost", "-store.un.pe", "store_.un.pe", "store..un.pe"}
	for _, invalidHostname := range invalidHostnames {
		if _, normalizeError := normalizeAndValidateHostname(invalidHostname); normalizeError == nil {
			t.Fatalf("expected %q to fail", invalidHostname)
		}
	}
}

func TestValidateWebpageArtifacts(t *testing.T) {
	webpagesDirectory := t.TempDir()
	tenantCount, validationError := validateWebpageArtifacts(webpagesDirectory)
	if validationError != nil || tenantCount != 0 {
		t.Fatalf("empty webpages should be valid, count=%d error=%v", tenantCount, validationError)
	}

	tenantDirectory := filepath.Join(webpagesDirectory, "store.un.pe")
	if createError := os.Mkdir(tenantDirectory, 0o755); createError != nil {
		t.Fatal(createError)
	}
	if writeError := os.WriteFile(filepath.Join(tenantDirectory, "index.html"), []byte("ok"), 0o644); writeError != nil {
		t.Fatal(writeError)
	}

	tenantCount, validationError = validateWebpageArtifacts(webpagesDirectory)
	if validationError != nil || tenantCount != 1 {
		t.Fatalf("expected one tenant, count=%d error=%v", tenantCount, validationError)
	}
}

func TestReplaceWebpageDirectory(t *testing.T) {
	baseDirectory := t.TempDir()
	targetDirectory := filepath.Join(baseDirectory, "store.un.pe")
	temporaryDirectory := filepath.Join(baseDirectory, ".build-store.un.pe")
	if createError := os.Mkdir(targetDirectory, 0o755); createError != nil {
		t.Fatal(createError)
	}
	if createError := os.Mkdir(temporaryDirectory, 0o755); createError != nil {
		t.Fatal(createError)
	}
	if writeError := os.WriteFile(filepath.Join(targetDirectory, "old.txt"), []byte("old"), 0o644); writeError != nil {
		t.Fatal(writeError)
	}
	if writeError := os.WriteFile(filepath.Join(temporaryDirectory, "index.html"), []byte("new"), 0o644); writeError != nil {
		t.Fatal(writeError)
	}

	if replaceError := replaceWebpageDirectory(targetDirectory, temporaryDirectory); replaceError != nil {
		t.Fatal(replaceError)
	}
	if !fileExists(filepath.Join(targetDirectory, "index.html")) {
		t.Fatal("new webpage was not published")
	}
	if fileExists(filepath.Join(targetDirectory, "old.txt")) {
		t.Fatal("old webpage content remains")
	}
}
