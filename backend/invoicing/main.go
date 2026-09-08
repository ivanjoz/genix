package invoicing

import "app/core"

// ModuleHandlers routes the electronic-invoicing APIs.
//
// The paths are the ones access.toml grants (ids 35 and 36) and
// core/api_routes.generated.go numbers, so adding one here means adding it in
// both of those too, or the route exists and nobody can reach it.
var ModuleHandlers = core.AppRouterType{
	// Issuing. POST.invoice returns as soon as the document has a number;
	// SUNAT answers later and the state moves on its own.
	"POST.invoice":       PostInvoice,
	"POST.invoice-retry": PostInvoiceRetry,
	"GET.invoices":       GetInvoices,
	"GET.invoice-xml":    GetInvoiceXML,

	// Series are not routed here: they travel inline on the company record, so
	// GET/POST.company-parametros reads and writes them.

	// SUNAT credentials: the SOL user and the signing certificate, both stored
	// encrypted and never returned to the client.
	"GET.company-secrets":       GetCompanySecrets,
	"POST.company-secrets":      PostCompanySecrets,
	"POST.company-secrets-test": PostCompanySecretsTest,
}
