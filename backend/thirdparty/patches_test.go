// Package thirdparty holds the forked protobuf this app cannot use unmodified, and the tests that
// keep it honest. The gocql fork lives in genix-orm/thirdparty/, with the module that imports it.
//
// The forks exist for one reason: without them the linker's dead-method elimination is disabled
// program-wide and the binary is ~12 MB larger. That failure is completely silent -- a dependency
// bump that drops a patch still compiles, still passes every other test, and just costs 12 MB. The
// tests here are the only thing that notices.
//
// See README.md for the mechanism and docs/BINARY_SIZE_PLAN.md for the measurements.
package thirdparty

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// TestPatchedProtobufStillEncodesQdrantMessages exercises the patched code paths on the messages
// this repo actually sends: internal/impl builds the message info (oneof and extension discovery),
// and internal/descfmt formats descriptors. A patch applied wrongly breaks here rather than in
// production, where the first symptom would be a failed vector search.
func TestPatchedProtobufStillEncodesQdrantMessages(t *testing.T) {
	original := &qdrant.QueryPoints{
		CollectionName: "documentation",
		Limit:          qdrant.PtrOf(uint64(7)),
		Query:          qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{qdrant.NewMatchKeyword("status", "implemented")},
		},
		WithPayload: qdrant.NewWithPayloadInclude("route", "module"),
	}

	wireBytes, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	decoded := &qdrant.QueryPoints{}
	if err := proto.Unmarshal(wireBytes, decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !proto.Equal(original, decoded) {
		t.Fatalf("round trip changed the message:\nbefore %v\nafter  %v", original, decoded)
	}

	// protojson walks the descriptor, which is where a broken oneof mapping surfaces.
	jsonBytes, err := protojson.Marshal(original)
	if err != nil {
		t.Fatalf("protojson: %v", err)
	}
	if !strings.Contains(string(jsonBytes), `"collectionName"`) {
		t.Fatalf("protojson lost field names: %s", jsonBytes)
	}

	// Oneof accessors are what the impl patch touches most directly.
	point := &qdrant.PointStruct{Id: qdrant.NewIDUUID("2f4e1c6a-0000-4000-8000-000000000001")}
	if point.GetId().GetUuid() == "" {
		t.Fatal("oneof accessor lost the uuid")
	}

	// %v on a descriptor is the descfmt path. It must still render fields, not panic or blank out.
	descriptorText := fmt.Sprintf("%v", original.ProtoReflect().Descriptor())
	if !strings.Contains(descriptorText, "collection_name") {
		t.Fatalf("descfmt stopped describing fields: %q", descriptorText)
	}
}

// TestForkedModulesStillCarryTheirPatches fails if a dependency bump regenerated a fork without
// re-applying the patch. Without this, the only symptom is a 12 MB binary.
func TestForkedModulesStillCarryTheirPatches(t *testing.T) {
	for _, forkedFile := range []struct {
		path           string
		mustContain    string
		mustNotContain string
		whatItProtects string
	}{
		{
			path:           "protobuf/internal/descfmt/stringer.go",
			mustContain:    "reflect.ValueOf(t.Methods)",
			mustNotContain: `rv.MethodByName("Methods")`,
			whatItProtects: "R_USENAMEDMETHOD on \"Methods\" reaches reflect.Value.Methods, which sets reflectSeen",
		},
		{
			path:           "protobuf/internal/impl/legacy_message.go",
			mustContain:    "zeroMessage.MethodByName",
			mustNotContain: `t.MethodByName("XXX_OneofWrappers")`,
			whatItProtects: "reflect.Type.MethodByName is an interface call that retains xunsafe.Field's promoted wrapper",
		},
		{
			path:           "protobuf/internal/impl/message.go",
			mustContain:    "zeroMessagePointer.MethodByName",
			mustNotContain: `reflect.PtrTo(t).MethodByName(`,
			whatItProtects: "same interface-call registration as legacy_message.go",
		},
	} {
		contents, err := os.ReadFile(forkedFile.path)
		if err != nil {
			t.Errorf("%s: %v", forkedFile.path, err)
			continue
		}
		if !strings.Contains(string(contents), forkedFile.mustContain) {
			t.Errorf("%s lost its patch (expected %q). Re-run thirdparty/regenerate.sh. Protects against: %s",
				forkedFile.path, forkedFile.mustContain, forkedFile.whatItProtects)
		}
		if strings.Contains(string(contents), forkedFile.mustNotContain) {
			t.Errorf("%s has the unpatched call %q back. Re-run thirdparty/regenerate.sh. Protects against: %s",
				forkedFile.path, forkedFile.mustNotContain, forkedFile.whatItProtects)
		}
	}
}

// TestShippingBuildTagsAgreeAcrossDeployPaths catches the build-tag lists drifting apart. They are
// spread across Go modules, a shell script and a CI workflow, so nothing else relates them --
// which is exactly how the CI release build and backend/deploy.sh were first missed. Any new path
// that produces a shipping binary belongs in this list.
func TestShippingBuildTagsAgreeAcrossDeployPaths(t *testing.T) {
	const expectedTags = "lambda.norpc,grpcnotrace"

	for _, buildPath := range []string{
		filepath.Join("..", "..", "cloud", "main.go"),                             // AWS Lambda
		filepath.Join("..", "..", "scripts", "deploy_vps.go"),                     // VPS deploy
		filepath.Join("..", "deploy.sh"),                                          // SAM deploy
		filepath.Join("..", "..", ".github", "workflows", "release-binaries.yml"), // GitHub release
	} {
		contents, err := os.ReadFile(buildPath)
		if err != nil {
			t.Errorf("%s: %v", buildPath, err)
			continue
		}
		// Matched bare rather than quoted: the Go files and the shell script quote it, the YAML
		// workflow does not. The joined form only ever appears in a real declaration -- the prose
		// around it names the two tags separately.
		if !strings.Contains(string(contents), expectedTags) {
			t.Errorf("%s does not declare the shipping build tags %q. Every shipping build path must "+
				"carry them or the binary silently grows ~12 MB", buildPath, expectedTags)
		}
	}
}
