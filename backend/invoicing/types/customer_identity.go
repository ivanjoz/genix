// Who a document has to identify, and when.
//
// It lives in this leaf package because two module bodies need the same answer at
// two different moments: `sales` when a sale is stamped with the series it will be
// issued under, and `invoicing` when the document is actually built. The two are
// hours apart and the first one is the only one that can still be fixed by the
// person at the till, so the rule has to be the same in both places or the sale
// takes an id no document can ever use.

package types

import (
	"errors"
	"strings"
)

// BoletaIdentifiedFrom is the amount, in cents, from which SUNAT requires the buyer
// to be named on a boleta: 700 soles. Below it a till sale stays anonymous.
const BoletaIdentifiedFrom = int32(70_000)

// A DNI is 8 digits and a RUC is 11, but a foreign buyer's document (carné de
// extranjería, passport) has no fixed length — hence a minimum rather than a set of
// exact lengths. The factura rule below is the exception and is exact.
const minIdentityDocumentLength = 8

// RequiresCustomerIdentity is whether this document can be issued to an
// unidentified buyer.
//
// Callers use it to skip reading the client at all: the till's most common sale is
// a small boleta, and that one needs nobody.
func RequiresCustomerIdentity(docType int8, totalAmount int32) bool {
	return docType == DocTypeFactura ||
		(docType == DocTypeBoleta && totalAmount >= BoletaIdentifiedFrom)
}

// ValidateCustomerIdentity checks the buyer against what the document demands.
//
// A note (credit or debit) is not checked here: it inherits the customer of the
// document it corrects, which was already validated when that one was issued.
func ValidateCustomerIdentity(docType int8, totalAmount int32, name, registryNumber string) error {
	if !RequiresCustomerIdentity(docType, totalAmount) {
		return nil
	}

	name = strings.TrimSpace(name)
	registryNumber = strings.TrimSpace(registryNumber)

	if docType == DocTypeFactura {
		// A factura is business to business and SUNAT accepts nothing but a RUC on
		// one, so the shape is checked and not just the presence.
		if len(registryNumber) != 11 || !isAllDigits(registryNumber) {
			return errors.New("una factura necesita un cliente con RUC de 11 dígitos")
		}
		if name == "" {
			return errors.New("una factura necesita la razón social del cliente")
		}
		return nil
	}

	if len(registryNumber) < minIdentityDocumentLength {
		return errors.New("una boleta desde S/ 700 necesita el DNI o RUC del cliente")
	}
	if name == "" {
		return errors.New("una boleta desde S/ 700 necesita el nombre del cliente")
	}
	return nil
}

func isAllDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return len(value) > 0
}
