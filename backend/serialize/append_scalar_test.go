package serialize

// Differential tests pinning the hand-written scalar encoders against sonic, which used to
// render these bytes. Any divergence here is a silent wire-format change, so these compare
// against the real encoder rather than against hand-written expectations.

import (
	"math"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/bytedance/sonic"
)

// sonicString is the oracle: what sonic would have produced for this string.
func sonicString(t *testing.T, value string) string {
	t.Helper()
	encoded, err := sonic.Marshal(value)
	if err != nil {
		t.Fatalf("sonic.Marshal(%q): %v", value, err)
	}
	return string(encoded)
}

func TestAppendJSONStringMatchesSonic(t *testing.T) {
	cases := []string{
		"",
		"plain",
		`quote " backslash \ end`,
		"tab\there\nnewline\rcarriage",
		"\x00\x01\x1f",
		"\b\f",
		"\x7f",
		"del\x7fhere",
		// sonic does not HTML-escape; encoding/json would mangle all of these.
		"<p>hola & adiós</p>",
		"</script>",
		"a < b && c > d",
		// sonic does not escape the line/paragraph separators either.
		"line sep end",
		"unicode: ñ é 中文",
		"emoji 🎉 family 👨‍👩‍👧",
		string([]byte{0xff, 0xfe}),
		"valid" + string([]byte{0xff}) + "tail",
		string([]byte{0xe2, 0x28, 0xa1}),
	}

	for _, value := range cases {
		got := string(appendJSONString(nil, value))
		want := sonicString(t, value)
		if got != want {
			t.Errorf("appendJSONString(%q):\n got  %s\n want %s", value, got, want)
		}
	}
}

// TestAppendJSONStringMatchesSonicQuick fuzzes the escaper over random strings, including the
// invalid-UTF-8 sequences that quick.Value produces for []byte-derived strings.
func TestAppendJSONStringMatchesSonicQuick(t *testing.T) {
	assertion := func(raw []byte) bool {
		value := string(raw)
		encoded, err := sonic.Marshal(value)
		if err != nil {
			return true // sonic refused it; nothing to match
		}
		return string(appendJSONString(nil, value)) == string(encoded)
	}

	if err := quick.Check(assertion, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("appendJSONString diverged from sonic: %v", err)
	}
}

func TestAppendScalarMatchesSonic(t *testing.T) {
	values := []any{
		true, false,
		int(0), int(-1), int8(127), int16(-32768), int32(2147483647), int64(math.MinInt64),
		uint(0), uint8(255), uint16(65535), uint32(4294967295), uint64(math.MaxUint64),
		float32(0), float32(10.5), float32(0.1), float32(-1.25e-7), float32(3.4e38),
		float64(0), float64(10.5), float64(0.1), float64(1e21), float64(1e-7),
		float64(1e-9), float64(-2.5e-10), float64(123456789.123456),
		"string value",
	}

	for _, value := range values {
		got, err := appendScalar(nil, reflect.ValueOf(value))
		if err != nil {
			t.Fatalf("appendScalar(%v of %T): %v", value, value, err)
		}

		want, err := sonic.Marshal(value)
		if err != nil {
			t.Fatalf("sonic.Marshal(%v of %T): %v", value, value, err)
		}

		if string(got) != string(want) {
			t.Errorf("appendScalar(%v of %T): got %s, want %s", value, value, got, want)
		}
	}
}

// TestAppendScalarFloatsMatchSonicQuick covers the exponent-notation boundaries that the
// hand-picked cases above can only sample.
func TestAppendScalarFloatsMatchSonicQuick(t *testing.T) {
	assertion := func(value float64) bool {
		if math.IsInf(value, 0) || math.IsNaN(value) {
			return true // JSON cannot represent these; sonic errors instead
		}
		got, err := appendScalar(nil, reflect.ValueOf(value))
		if err != nil {
			return false
		}
		want, err := sonic.Marshal(value)
		if err != nil {
			return true
		}
		return string(got) == string(want)
	}

	if err := quick.Check(assertion, &quick.Config{MaxCount: 20000}); err != nil {
		t.Errorf("appendScalar diverged from sonic on floats: %v", err)
	}
}
