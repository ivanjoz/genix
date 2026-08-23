package main

import "testing"

// The regression this guards: `type CashMovementType int8` parses as an unknown identifier,
// so the field would map to ICashMovementType, fail the declared-interface lookup, and be
// emitted as `any` — silently downgrading a typed numeric column in the frontend interface.
func TestResolveNamedBasicType(t *testing.T) {
	namedBasics := map[string]string{"CashMovementType": "int8", "ExpenseStatus": "int16"}
	cases := []struct{ in, want string }{
		{"ICashMovementType", "number"},
		{"ICashMovementType[]", "number[]"},
		{"IExpenseStatus", "number"},
		{"IProduct", "IProduct"},     // a real struct reference must be left alone
		{"IProduct[]", "IProduct[]"}, // including its slice form
		{"number", "number"},
		{"string", "string"},
	}
	for _, c := range cases {
		if got := resolveNamedBasicType(c.in, namedBasics); got != c.want {
			t.Errorf("resolveNamedBasicType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The declaration and the struct that uses it can live in different files of the same types
// folder, which is why resolution is a second pass over all collected structs.
func TestParseBackendStructFileCollectsNamedBasics(t *testing.T) {
	structs, namedBasics, err := parseBackendStructFile("../../backend/finance/types/cash_banks.go", "finance")
	if err != nil {
		t.Fatal(err)
	}
	if namedBasics["CashMovementType"] != "int8" {
		t.Fatalf("CashMovementType not collected: %v", namedBasics)
	}
	for _, s := range structs {
		if s.Name != "CashBankMovement" {
			continue
		}
		for _, f := range s.Fields {
			if f.GoName != "Type" {
				continue
			}
			if got := resolveNamedBasicType(f.TSType, namedBasics); got != "number" {
				t.Fatalf("CashBankMovement.Type resolved to %q, want number", got)
			}
			return
		}
		t.Fatal("CashBankMovement.Type field not found")
	}
	t.Fatal("CashBankMovement struct not found")
}
