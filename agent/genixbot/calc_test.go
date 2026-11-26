package genixbot

import "testing"

func TestCalculate(t *testing.T) {
	cases := map[string]string{
		"38224 / 92":               "415.48",
		"(150 - 90) / 150 * 100":   "40.00",
		"1180.50 - 1180.50 / 1.18": "180.08",
		"-2 ^ 2":                   "4.00",
		"45000 * 0.015 / (1 - (1 + 0.015) ^ -12)": "4125.60",
	}
	for expression, want := range cases {
		if got := CalculateResult(expression); got != "Resultado | "+expression+" = "+want {
			t.Errorf("%s: %s, want %s", expression, got, want)
		}
	}
	for _, bad := range []string{"", "2 +", "1/0", "2 x 3", "(1+2"} {
		if _, err := Calculate(bad); err == nil {
			t.Errorf("%q should fail", bad)
		}
	}
}
