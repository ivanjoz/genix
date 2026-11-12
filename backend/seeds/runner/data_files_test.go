package runner

import "testing"

func TestParsePriceToCents(t *testing.T) {
	testCases := []struct {
		rawValue      string
		expectedCents int32
	}{
		{"4500.00", 450000},
		{"689.66", 68966},
		{"0.05", 5},
		{" 12.5 ", 1250},
		// Binary floating point turns this into 19.899999…; without rounding it truncates to 1989.
		{"19.90", 1990},
	}

	for _, testCase := range testCases {
		parsedCents, parseError := ParsePriceToCents(testCase.rawValue)
		if parseError != nil {
			t.Fatalf("ParsePriceToCents(%q): %v", testCase.rawValue, parseError)
		}
		if parsedCents != testCase.expectedCents {
			t.Errorf("ParsePriceToCents(%q) = %d, want %d", testCase.rawValue, parsedCents, testCase.expectedCents)
		}
	}

	if _, parseError := ParsePriceToCents("no-es-un-monto"); parseError == nil {
		t.Error("expected an error for a non-numeric amount")
	}
}
