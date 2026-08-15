package types

import (
	"app/db"
	"testing"
)

// Compiling the schema is what catches an unsupported key or index shape here rather than at
// deploy time, where the first sign of trouble is fn-homologate refusing to create the table.
func TestRequestLogSchemasCompile(t *testing.T) {
	userLogColumns := db.MakeTable[UserLog]().GetColumns()
	if len(userLogColumns) == 0 {
		t.Fatal("user_logs compiled to zero columns")
	}
	errorColumns := db.MakeTable[RequestError]().GetColumns()
	if len(errorColumns) == 0 {
		t.Fatal("request_errors compiled to zero columns")
	}
}

// These vectors are the contract with server_utils/src/reqlog/protocol.rs, which is the side that
// writes the column. The Rust test file carries the same numbers; if either moves, the dashboard
// silently reads rows that were packed under a different layout.
func TestMakeFrameRouteCompanyAgg(t *testing.T) {
	cases := []struct {
		name      string
		frame     uint8
		routeID   int16
		companyID int32
		expected  int64
	}{
		{"all zero", 0, 0, 0, 0},
		{"frame only", 1, 0, 0, 1 << 40},
		{"route only", 0, 1, 0, 1 << 24},
		{"company only", 0, 0, 1, 1},
		{"last frame of the day", 95, 0, 0, 95 << 40},
		{"a real row", 41, 102, 7, 41<<40 | 102<<24 | 7},
		{"every field at its ceiling", 95, 32767, 16_777_215, 95<<40 | 32767<<24 | 16_777_215},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			packed := MakeFrameRouteCompanyAgg(testCase.frame, testCase.routeID, testCase.companyID)
			if packed != testCase.expected {
				t.Fatalf("packed %d, expected %d", packed, testCase.expected)
			}
			if packed < 0 {
				t.Fatalf("packed value went negative: %d", packed)
			}
		})
	}
}

// A frame's range must cover every route and company inside it and stop before the next frame
// begins, or the delta read either misses rows or double counts them.
func TestFrameRangeCoversExactlyOneFrame(t *testing.T) {
	lowest, highest := FrameRange(41)
	inside := MakeFrameRouteCompanyAgg(41, 102, 7)
	if inside < lowest || inside > highest {
		t.Fatalf("a row of frame 41 (%d) fell outside its own range [%d, %d]", inside, lowest, highest)
	}
	if ceiling := MakeFrameRouteCompanyAgg(41, 32767, 16_777_215); ceiling > highest {
		t.Fatalf("the widest row of frame 41 (%d) exceeded the range ceiling %d", ceiling, highest)
	}
	nextLowest, _ := FrameRange(42)
	if highest >= nextLowest {
		t.Fatalf("frame 41 ends at %d, which overlaps frame 42 starting at %d", highest, nextLowest)
	}
}

func TestFrameOfDay(t *testing.T) {
	cases := []struct {
		unixSeconds int64
		expected    uint8
	}{
		{0, 0},                       // midnight UTC
		{14 * 60, 0},                 // still the first slot
		{15 * 60, 1},                 // the boundary belongs to the next slot
		{86_400 - 1, 95},             // the last second of the day
		{86_400, 0},                  // and the next day starts over
		{1_767_225_600 + 615*60, 41}, // 10:15 UTC — the worked example above
	}
	for _, testCase := range cases {
		if frame := FrameOfDay(testCase.unixSeconds); frame != testCase.expected {
			t.Fatalf("unix %d gave frame %d, expected %d", testCase.unixSeconds, frame, testCase.expected)
		}
	}
}

// The ID is what a user_logs row stores, so it has to be stable across runs and positive: the
// column is a signed int32 and a negative ID reads as corruption to anyone who sees one.
func TestMakeRequestErrorIDIsStableAndPositive(t *testing.T) {
	first := MakeRequestErrorID("responses.go:539")
	if first != MakeRequestErrorID("responses.go:539") {
		t.Fatal("the same code line hashed to two different IDs")
	}
	if first == MakeRequestErrorID("responses.go:540") {
		t.Fatal("two different code lines collided; the vectors need revisiting")
	}
	for _, codeLine := range []string{"", "a", "responses.go:539", "product-stock-movement.go:1204"} {
		if id := MakeRequestErrorID(codeLine); id < 0 {
			t.Fatalf("code line %q hashed to a negative ID: %d", codeLine, id)
		}
	}
}
