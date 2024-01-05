package facturacion

import (
	"encoding/xml"
)

type FacInvoice struct {
	XMLName          xml.Name         `xml:"Invoice"`
	Xmlns            string           `xml:"xmlns,attr"`
	XmlnsCAC         string           `xml:"xmlns:cac,attr"`
	XmlnsCBC         string           `xml:"xmlns:cbc,attr"`
	XmlnsCCTS        string           `xml:"xmlns:ccts,attr"`
	XmlnsDS          string           `xml:"xmlns:ds,attr"`
	XmlnsEXT         string           `xml:"xmlns:ext,attr"`
	XmlnsQDT         string           `xml:"xmlns:qdt,attr"`
	XmlnsUDT         string           `xml:"xmlns:udt,attr"`
	XmlnsXSI         string           `xml:"xmlns:xsi,attr"`
	FacUBLExtensions FacUBLExtensions `xml:"ext:UBLExtensions"`
	UBLVersionID     string           `xml:"cbc:UBLVersionID"`
	CustomizationID  string           `xml:"cbc:CustomizationID"`
	ProfileID        FacGeneric       `xml:"cbc:ProfileID"`

	ID        string `xml:"cbc:ID"`        // Numero de la boleta o factura
	IssueDate string `xml:"cbc:IssueDate"` // Fecha de la factura
	IssueTime string `xml:"cbc:IssueTime"` // Hora de la factura

	FacInvoiceTypeCode FacInvoiceTypeCode `xml:"cbc:InvoiceTypeCode"`
	Notes              []FacNote          `xml:"cbc:Note"`

	DocumentCurrencyCode FacDocumentCurrencyCode `xml:"cbc:DocumentCurrencyCode"`

	Signature               FacSignature     `xml:"cac:Signature"`
	AccountingSupplierParty FacSupplierParty `xml:"cac:AccountingSupplierParty"`
	AccountingCustomerParty FacCustomerParty `xml:"cbc:AccountingCustomerParty>cbc:Party"`
	TaxTotal                FacTaxTotal      `xml:"cac:TaxTotal"`

	LegalMonetaryTotal FacLegalMonetaryTotal `xml:"cac:LegalMonetaryTotal"`
	InvoiceLines       []FacInvoiceLine      `xml:"cac:InvoiceLine"`
}

type FacUBLExtensions struct {
	UBLExtensionContent FacUBLExtensionContent `xml:"ext:UBLExtension"`
}
type FacUBLExtensionContent struct {
	Signature string `xml:",innerxml"`
}

type FacGeneric struct {
	Content          string `xml:",innerxml"`
	SchemeID         string `xml:"schemeID,attr,omitempty"`
	SchemeName       string `xml:"schemeName,attr,omitempty"`
	SchemeAgencyName string `xml:"schemeAgencyName,attr,omitempty"`
	SchemeAgencyID   string `xml:"schemeAgencyID,attr,omitempty"`
	SchemeURI        string `xml:"schemeURI,attr,omitempty"`
	CurrencyID       string `xml:"currencyID,attr,omitempty"`
	UnitCode         string `xml:"unitCode,attr,omitempty"`
	UnitCodeListID   string `xml:"unitCodeListID,attr,omitempty"`
	ListID           string `xml:"listID,attr,omitempty"`
	ListAgencyName   string `xml:"listAgencyName,attr,omitempty"`
	ListName         string `xml:"listName,attr,omitempty"`
	ListURI          string `xml:"listURI,attr,omitempty"`
	Algorithm        string `xml:"Algorithm,attr,omitempty"`

	UnitCodeListAgencyName string `xml:"unitCodeListAgencyName,attr,omitempty"`
}

type FacInvoiceTypeCode struct {
	XMLName        xml.Name `xml:"cbc:InvoiceTypeCode"`
	Content        string   `xml:",innerxml"`
	ListAgencyName string   `xml:"listAgencyName,attr"`
	ListName       string   `xml:"listName,attr"`
	ListURI        string   `xml:"listURI,attr"`
}

type FacNote struct {
	LanguageLocaleID string `xml:"languageLocaleID,attr"`
	Content          string `xml:",innerxml"`
}

type FacDocumentCurrencyCode struct {
	Content        string `xml:",innerxml"`
	ListID         string `xml:"listID,attr"`
	ListName       string `xml:"listName,attr"`
	ListAgencyName string `xml:"listAgencyName,attr"`
}

func NewFacNote(content, languageLocaleID string) FacNote {
	return FacNote{
		LanguageLocaleID: languageLocaleID,
		Content:          content,
	}
}

type FacSignatureParty struct {
	PartyIdentification FacPartyInfo `xml:"cac:PartyIdentification"`
	PartyName           FacPartyInfo `xml:"cac:PartyName"`
}

type FacPartyInfo struct {
	Name string `xml:"cbc:Name,omitempty"`
	ID   string `xml:"cbc:ID,omitempty"`
}

type FacDigitalSignatureAtt struct {
	ExternalURI string `xml:"cac:ExternalReference>cbc:URI"`
}

type FacSignature struct {
	ID                  string                 `xml:"cbc:ID"`
	SignatoryParty      FacSignatureParty      `xml:"cac:SignatoryParty"`
	DigitalSignatureAtt FacDigitalSignatureAtt `xml:"cac:DigitalSignatureAttachment"`
}

type FacSupplierParty struct {
	Party          FacParty          `xml:"cac:Party"`
	PartyTaxScheme FacPartyTaxScheme `xml:"cac:PartyTaxScheme"`
}

type FacParty struct {
	PartyName FacPartyInfo
}

type FacPartyRegistrationName struct {
	Content string `xml:",cdata"`
}

type FacPartyTaxScheme struct {
	RegistrationName FacPartyRegistrationName `xml:"cbc:RegistrationName"`
	EmpresaID        FacGeneric               `xml:"cac:EmpresaID"`

	RegddressTypeCode string `xml:"cac:RegistrationAddress>cbc:AddressTypeCode"`
	TaxSchemeID       string `xml:"cac:TaxScheme>cbc:ID"`
}

type CostumerParty struct {
	PartyTaxScheme FacPartyTaxScheme `xml:"cac:PartyTaxScheme"`
}

type FacCustomerParty struct {
	PartyTaxScheme FacPartyTaxScheme `xml:"cac:PartyTaxScheme"`
}

type FacTaxCategory struct {
	ID                FacGeneric `xml:"cbc:ID"`
	SchemeID          FacGeneric `xml:"cbc:TaxScheme>cbc:ID"`
	SchemeName        string     `xml:"cbc:TaxScheme>cbc:Name"`
	SchemeTaxTypeCode string     `xml:"cbc:TaxScheme>cbc:TaxTypeCode"`
}

type FacTaxTotal struct {
	TaxAmount       FacGeneric     `xml:"cbc:TaxAmount"`
	StTaxableAmount FacGeneric     `xml:"cac:TaxSubtotal>cac:TaxableAmount"`
	StTaxAmount     FacGeneric     `xml:"cac:TaxSubtotal>cac:TaxAmount"`
	TaxCategory     FacTaxCategory `xml:"cbc:TaxCategory"`
}

type FacLegalMonetaryTotal struct {
	LineExtensionAmount  FacGeneric `xml:"cbc:LineExtensionAmount"`
	TaxInclusiveAmount   FacGeneric `xml:"cac:TaxInclusiveAmount"`
	AllowanceTotalAmount FacGeneric `xml:"cac:AllowanceTotalAmount"`
	PayableAmount        FacGeneric `xml:"cbc:PayableAmount"`
}

// Invoice Line
type FacInvoiceLine struct {
	ID                  int32               `xml:"cbc:ID"`
	InvoicedQuantity    FacGeneric          `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount FacGeneric          `xml:"cbc:LineExtensionAmount"`
	PricingReference    FacPricingReference `xml:"cac:PricingReference"`
	TaxTotal            FacLineTaxTotal     `xml:"cac:TaxTotal"`
	Item                FacItem             `xml:"cac:Item"`
	PriceAmount         FacGeneric          `xml:"cac:Price>cbc:PriceAmount"`
}

type FacPricingReference struct {
	PriceAmount   FacGeneric `xml:"cbc:AlternativeConditionPrice>cbc:PriceAmount"`
	PriceTypeCode FacGeneric `xml:"cbc:AlternativeConditionPrice>cbc:PriceTypeCode"`
}

type FacLineTaxTotal struct {
	TaxAmount        FacGeneric `xml:"cbc:TaxAmount"`
	SubTaxAmount     FacGeneric `xml:"cbc:TaxSubtotal>cac:TaxAmount"`
	SubTaxCategoryID FacGeneric `xml:"cbc:TaxSubtotal>cac:TaxCategory>cbc:ID"`

	SubTaxCategoryPercent     FacGeneric `xml:"cac:TaxSubtotal>cac:TaxCategory>cbc:Percent"`
	SubTaxExemptionReasonCode FacGeneric `xml:"cac:TaxSubtotal>cac:TaxCategory>cbc:TaxExemptionReasonCode"`

	SubTaxScheme FacTaxCategory `xml:"cac:TaxSubtotal>cac:TaxCategory>cac:TaxScheme"`
}

type FacItem struct {
	Description                 string     `xml:"cbc:Description"`
	SellersItemIdentificationID string     `xml:"cac:SellersItemIdentification>cbc:ID"`
	CommodityClassificationCode FacGeneric `xml:"cac:CommodityClassification>cbc:ItemClassificationCode"`
}

/* SIGNATURE */
type FacExtSignedReference struct {
	URI          string     `xml:"URI,attr"`
	Transform    FacGeneric `xml:"ds:Transforms>ds:Transform"`
	DigestMethod FacGeneric `xml:"ds:DigestMethod"`
	DigestValue  string     `xml:"ds:DigestValue"`
}

type FacExtSignedInfo struct {
	CanonicalizationMethod FacGeneric            `xml:"ds:CanonicalizationMethod"`
	SignatureMethod        FacGeneric            `xml:"ds:SignatureMethod"`
	Reference              FacExtSignedReference `xml:"ds:Reference"`
}

type FacExtSignature struct {
	XMLName        xml.Name         `xml:"ds:Signature"`
	ID             string           `xml:"Id,attr"`
	SignedInfo     FacExtSignedInfo `xml:"ds:SignedInfo"`
	SignatureValue string           `xml:"ds:SignatureValue"`

	KeyInfoX509SubjectName string `xml:"ds:KeyInfo>ds:X509Data>ds:X509SubjectName"`
	KeyInfoX509Certificate string `xml:"ds:KeyInfo>ds:X509Data>ds:X509Certificate"`
}
