// SUNAT's "tipo de documento de identidad" catalog — Anexo 1 of RS 112-2021, and column 11
// of the SIRE sales book.
//
// The column stores a number, not the character SUNAT prints. The catalog is a picker in the
// UI, and every other picker in the app keys on an int id; storing the character forced the
// frontend to carry string ids for this one list alone. The cost is one conversion at each
// boundary that speaks SUNAT — SunatIdentityDocCode, called in sale_order_to_cpe.go for the
// XML and in sales_book.go for the book.
//
// Two rules fix the numbers, and IdentityDocOptions is the only place they are written down:
//
//   - The id is the SUNAT digit wherever SUNAT uses a digit (1, 4, 6, 7), and continues
//     through the hex values for "A".."F" (10..15). For the four types that cover nearly
//     every real row, a raw DB dump still reads as the code it prints.
//   - 0 means *not declared*, so `json:",omitempty"` drops it and DeriveIdentityDocType can
//     key on it. SUNAT's own code "0" therefore cannot be id 0 — it takes 9, the one free
//     slot below the hex range. It is the single id that does not read as its own code,
//     which is what TestIdentityDocCodesMatchFacturago exists to keep honest.

package types

import (
	"fmt"
	"strings"
)

const (
	IdentityDocDNI        int8 = 1  // SUNAT "1"
	IdentityDocForeign    int8 = 4  // SUNAT "4" — carné de extranjería
	IdentityDocRUC        int8 = 6  // SUNAT "6"
	IdentityDocPassport   int8 = 7  // SUNAT "7"
	IdentityDocNone       int8 = 9  // SUNAT "0" — doc. tributario no domiciliado sin RUC
	IdentityDocDiplomatic int8 = 10 // SUNAT "A" — cédula diplomática de identidad
	IdentityDocResidence  int8 = 11 // SUNAT "B" — documento del país de residencia
	IdentityDocTIN        int8 = 12 // SUNAT "C"
	IdentityDocIN         int8 = 13 // SUNAT "D"
	IdentityDocTAM        int8 = 14 // SUNAT "E"
	IdentityDocPTP        int8 = 15 // SUNAT "F"
)

// IdentityDocOption is one entry of the catalog. Name carries the app's "EN|ES" bilingual
// form because the frontend copy of this list feeds a picker directly.
//
// Label is the abbreviation a narrow control shows instead of Name — the till's document
// field puts the picker inside the input, where "Carné de extranjería" eats the number it is
// qualifying. Only the entries whose name does not fit carry one.
type IdentityDocOption struct {
	ID    int8   `json:",omitempty"`
	Code  string `json:",omitempty"` // SUNAT Anexo 1
	Name  string `json:",omitempty"`
	Label string `json:",omitempty"`
}

// IdentityDocOptions is the whole catalog: what the save path validates against, what
// SunatIdentityDocCode converts through, and — via the //CATALOG: marker in
// frontend/services/crm/client-provider.svelte.ts — what the UI pickers show.
//
// The order is the order the pickers list them: the four a till actually sees first, then
// the rest.
var IdentityDocOptions = []IdentityDocOption{
	{ID: IdentityDocDNI, Code: "1", Name: "DNI|DNI"},
	{ID: IdentityDocRUC, Code: "6", Name: "RUC|RUC"},
	{ID: IdentityDocForeign, Code: "4", Name: "Foreign ID|Carné de extranjería", Label: "F.ID|C.E."},
	{ID: IdentityDocPassport, Code: "7", Name: "Passport|Pasaporte", Label: "PSP"},
	{ID: IdentityDocNone, Code: "0", Name: "No RUC (non-domiciled)|Sin RUC (no domiciliado)"},
	{ID: IdentityDocDiplomatic, Code: "A", Name: "Diplomatic ID|Cédula diplomática"},
	{ID: IdentityDocResidence, Code: "B", Name: "Residence country doc|Doc. país de residencia"},
	{ID: IdentityDocTIN, Code: "C", Name: "TIN|TIN"},
	{ID: IdentityDocIN, Code: "D", Name: "IN|IN"},
	{ID: IdentityDocTAM, Code: "E", Name: "TAM|TAM"},
	{ID: IdentityDocPTP, Code: "F", Name: "PTP|PTP"},
}

// SunatIdentityDocCode is the character SUNAT prints for a stored document type, and the
// empty string for one that was never declared.
func SunatIdentityDocCode(docType int8) string {
	for index := range IdentityDocOptions {
		if IdentityDocOptions[index].ID == docType {
			return IdentityDocOptions[index].Code
		}
	}
	return ""
}

// IsIdentityDocType reports whether a value names a catalog entry. 0 is not one: it means
// the client declared nothing and wants the fallback.
func IsIdentityDocType(docType int8) bool {
	return SunatIdentityDocCode(docType) != ""
}

// IdentityDocTypeNames lists the catalog the way a validation error should show it.
func IdentityDocTypeNames() string {
	names := make([]string, 0, len(IdentityDocOptions))
	for index := range IdentityDocOptions {
		names = append(names, fmt.Sprint(IdentityDocOptions[index].ID, "=", IdentityDocOptions[index].Code))
	}
	return strings.Join(names, ", ")
}

// DeriveIdentityDocType guesses the kind of identity document from its shape: in Peru
// eleven digits is a RUC and eight is a DNI.
//
// It is a fallback, not the rule. The shape cannot separate a carné de extranjería from a
// pasaporte, a TAM or a PTP — they are all "something else" — which is exactly why
// ClientProvider carries the column, and why the till now offers a picker. This fills it in
// for a row saved without one.
func DeriveIdentityDocType(registryNumber string) int8 {
	switch len(registryNumber) {
	case 11:
		return IdentityDocRUC
	case 8:
		return IdentityDocDNI
	case 0:
		return 0
	}
	return IdentityDocForeign
}
