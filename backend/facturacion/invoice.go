package facturacion

func NewInvoiceLine() FacInvoiceLine {

	return FacInvoiceLine{
		ID: 1,
		InvoicedQuantity: FacGeneric{
			Content:                "1",
			UnitCode:               "NIU",
			UnitCodeListID:         "UN/ECE rec 20",
			UnitCodeListAgencyName: "United Nations Economic Commission forEurope",
		},
		LineExtensionAmount: FacGeneric{CurrencyID: "PEN", Content: "845.76"},
		PricingReference: FacPricingReference{
			PriceAmount: FacGeneric{CurrencyID: "PEN", Content: "998.00"},
			PriceTypeCode: FacGeneric{
				Content:        "01",
				ListName:       "SUNAT:Indicador de Tipo de Precio",
				ListAgencyName: "PE:SUNAT",
				ListURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo16",
			},
		},
		TaxTotal: FacLineTaxTotal{
			TaxAmount:    FacGeneric{CurrencyID: "PEN", Content: "0.00"},
			SubTaxAmount: FacGeneric{CurrencyID: "PEN", Content: "152.24"},
			SubTaxCategoryID: FacGeneric{
				Content:          "18.00",
				SchemeID:         "UN/ECE 5305",
				SchemeName:       "Tax Category Identifier",
				SchemeAgencyName: "United Nations Economic Commission for Europe",
			},
			SubTaxExemptionReasonCode: FacGeneric{
				Content:        "10",
				ListAgencyName: "PE:SUNAT",
				ListName:       "SUNAT:Codigo de Tipo de Afectación del IGV",
				ListURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo07",
			},
			SubTaxScheme: FacTaxCategory{
				ID: FacGeneric{
					Content:          "1000",
					SchemeID:         "UN/ECE 5153",
					SchemeName:       "Tax Scheme Identifier",
					SchemeAgencyName: "United Nations Economic Commission for Europe",
				},
				SchemeName:        "IGV",
				SchemeTaxTypeCode: "VAT",
			},
		},
		Item: FacItem{
			Description:                 "Refrigeradora marca “AXM” no frost de 200 ltrs.",
			SellersItemIdentificationID: "REF564",
			CommodityClassificationCode: FacGeneric{
				Content:        "52141501",
				ListID:         "UNSPSC",
				ListAgencyName: "listAgencyName",
				ListName:       "Item Classification",
			},
		},
		PriceAmount: FacGeneric{CurrencyID: "PEN", Content: "845.76"},
	}

}

// https://learn.microsoft.com/en-us/previous-versions/windows/desktop/ms767623(v=vs.85)

func NewInvoice() FacInvoice {
	invoice := FacInvoice{
		Xmlns:     "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2",
		XmlnsCAC:  "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2",
		XmlnsCBC:  "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2",
		XmlnsCCTS: "urn:un:unece:uncefact:documentation:2",
		XmlnsDS:   "http://www.w3.org/2000/09/xmldsig#",
		XmlnsEXT:  "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2",
		XmlnsQDT:  "urn:oasis:names:specification:ubl:schema:xsd:QualifiedDatatypes-2",
		XmlnsUDT:  "urn:un:unece:uncefact:data:specification:UnqualifiedDataTypesSchemaModule:2",
		XmlnsXSI:  "http://www.w3.org/2001/XMLSchema-instance",

		UBLVersionID:    "2.1",
		CustomizationID: "2.0",
		ProfileID: FacGeneric{
			SchemeName:       "SUNAT:Identificador de Tipo de Operación",
			SchemeAgencyName: "PE:SUNAT",
			SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo17",
			Content:          "0101",
		},
		ID:        "", // ID de la boleta (BC01-3652)
		IssueDate: "", // fecha de la boleta (2017-06-24)
		IssueTime: "", // fecha de la boleta (18:01:29)
		FacInvoiceTypeCode: FacInvoiceTypeCode{
			Content:        "03", // Boleta
			ListAgencyName: "PE:SUNAT",
			ListName:       "SUNAT:Identificador de Tipo de Documento",
			ListURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo01",
		},
		Notes: []FacNote{
			NewFacNote("SON MIL SEISCIENTOS SESENTA Y 60/100", "1000"),
			NewFacNote("050100201706240046", "3000"),
		},
		DocumentCurrencyCode: FacDocumentCurrencyCode{
			Content:        "PEN",
			ListID:         "ISO 4217 Alpha",
			ListName:       "Currency",
			ListAgencyName: "United Europe",
		},
		Signature: FacSignature{
			ID: "IDSignKG",
			SignatoryParty: FacSignatureParty{
				PartyIdentification: FacPartyInfo{
					ID: "10200545523",
				},
				PartyName: FacPartyInfo{
					Name: "VEGA POBLETE CARLOS ENRIQUE",
				},
			},
			DigitalSignatureAtt: FacDigitalSignatureAtt{
				ExternalURI: "#SignatureKG",
			},
		},
		AccountingSupplierParty: FacSupplierParty{
			Party: FacParty{
				PartyName: FacPartyInfo{
					Name: "Electrodomésticos Cruz de Motupe",
				},
			},
		},
		AccountingCustomerParty: FacCustomerParty{
			PartyTaxScheme: FacPartyTaxScheme{
				RegistrationName: FacPartyRegistrationName{
					Content: "Vega Poblete Carlos Enrique",
				},
				EmpresaID: FacGeneric{
					Content:          "10200545523",
					SchemeID:         "6",
					SchemeName:       "SUNAT:Identificador de Documento de Identidad",
					SchemeAgencyName: "PE:SUNAT",
					SchemeURI:        "urn:pe:gob:sunat:cpe:see:gem:catalogos:catalogo0",
				},
			},
		},
		TaxTotal: FacTaxTotal{
			TaxAmount:       FacGeneric{CurrencyID: "PEN", Content: "253.31"},
			StTaxableAmount: FacGeneric{CurrencyID: "PEN", Content: "1407.29"},
			StTaxAmount:     FacGeneric{CurrencyID: "PEN", Content: "253.31"},
			TaxCategory: FacTaxCategory{
				ID: FacGeneric{
					SchemeID:         "UN/ECE 5305",
					SchemeName:       "Tax Category Identifier",
					SchemeAgencyName: "United Nations Economic Commission for Europe",
					Content:          "S",
				},
				SchemeID: FacGeneric{
					SchemeID: "UN/ECE 5153", SchemeAgencyID: "6", Content: "1000",
				},
				SchemeName:        "IGV",
				SchemeTaxTypeCode: "VAT",
			},
		},
		LegalMonetaryTotal: FacLegalMonetaryTotal{
			LineExtensionAmount:  FacGeneric{CurrencyID: "PEN", Content: "1407.29"},
			TaxInclusiveAmount:   FacGeneric{CurrencyID: "PEN", Content: "1660.60"},
			AllowanceTotalAmount: FacGeneric{CurrencyID: "PEN", Content: "0.00"},
			PayableAmount:        FacGeneric{CurrencyID: "PEN", Content: "1660.60"},
		},
		InvoiceLines: []FacInvoiceLine{},
	}

	for i := 0; i < 4; i++ {
		invoice.InvoiceLines = append(invoice.InvoiceLines, NewInvoiceLine())
	}

	return invoice
}

type MakeSignatureArgs struct {
	SignatureID     string
	DigestValue     string
	SignatureValue  string
	X509SubjectName string
	X509Certificate string
}

func MakeSignature(args MakeSignatureArgs) FacExtSignature {

	return FacExtSignature{
		ID: args.SignatureID,
		SignedInfo: FacExtSignedInfo{
			CanonicalizationMethod: FacGeneric{
				Algorithm: "http://www.w3.org/TR/2001/REC-xml-c14n20010315"},
			SignatureMethod: FacGeneric{
				Algorithm: "http://www.w3.org/2000/09/xmldsig#rsa-sha1"},
			Reference: FacExtSignedReference{
				URI: "",
				Transform: FacGeneric{
					Algorithm: "http://www.w3.org/2000/09/xmldsig#envelopedsignature"},
				DigestMethod: FacGeneric{Algorithm: "http://www.w3.org/2000/09/xmldsig#sha1"},
				DigestValue:  args.DigestValue,
			},
		},
		SignatureValue:         args.SignatureValue,
		KeyInfoX509SubjectName: args.X509SubjectName,
		KeyInfoX509Certificate: args.X509Certificate,
	}

}
