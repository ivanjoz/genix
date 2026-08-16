package exec

import (
	"reflect"
	"testing"
)

func TestObservabilityBackfillCodecRoundTripsRouteTotals(t *testing.T) {
	expected := map[int16]backfillCredits{
		3:   {cpu: 300, inference: 25},
		104: {cpu: 65_536, inference: 1},
	}
	encoded, err := encodeBackfillCreditBlob(expected)
	if err != nil {
		t.Fatal(err)
	}
	decoded := map[int16]backfillCredits{}
	if err := mergeBackfillCreditBlob(encoded, decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, expected) {
		t.Fatalf("decoded %#v, expected %#v", decoded, expected)
	}
}

func TestParseObservabilityBackfillHoursIsBounded(t *testing.T) {
	if hours, err := parseObservabilityBackfillHours(""); err != nil || hours != 4 {
		t.Fatalf("default hours = %d, err = %v", hours, err)
	}
	if hours, err := parseObservabilityBackfillHours("24"); err != nil || hours != 24 {
		t.Fatalf("explicit hours = %d, err = %v", hours, err)
	}
	for _, invalid := range []string{"0", "25", "four"} {
		if _, err := parseObservabilityBackfillHours(invalid); err == nil {
			t.Fatalf("invalid argument %q was accepted", invalid)
		}
	}
}
