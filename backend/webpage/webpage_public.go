package webpage

import (
	"app/core"
	"app/db"
	s "app/webpage/types"
)

// WebpagePublicResult is the unauthenticated payload for a single storefront page:
// its SEO metatags (Config) plus its content sections (Sections, with the
// section-1 whole-page CSS). Consumed by the prerender build and deployed storefronts.
type WebpagePublicResult struct {
	Config   map[string]string        `json:"Config"`
	Sections []s.EcommercePageContent `json:"Sections"`
}

// GetWebpagePublic is the single public (no-auth, "p-" prefix) read for a storefront
// page: GET.p-webpage?company-id=<id>&id=<pageID>. It is scoped by the company-id
// (alias cid) query param — never a session, so one tenant can't read another's —
// and the id param selects the page (defaults to the root/Inicio page). Only active
// sections and the known SEO keys are exposed. *Never trust the client.*
func GetWebpagePublic(req *core.HandlerArgs) core.HandlerResponse {
	companyID := core.Coalesce(req.GetQueryInt("company-id"), req.GetQueryInt("cid"))
	if companyID <= 0 {
		return req.MakeErr("company-id inválido:", companyID)
	}

	pageID := int16(req.GetQueryInt("id"))
	if pageID == 0 {
		pageID = defaultPageID
	}

	result := WebpagePublicResult{
		Config:   map[string]string{},
		Sections: []s.EcommercePageContent{},
	}

	seoConfig, err := publicSeoMetatags(companyID)
	if err != nil {
		return req.MakeErr("Error al obtener la configuración pública del sitio:", err)
	}
	result.Config = seoConfig

	rows := []s.EcommercePageContent{}
	query := db.Query(&rows)
	query.Select().CompanyID.Equals(companyID)
	query.PageID.Equals(pageID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el contenido público de la página:", err)
	}
	for _, row := range rows {
		if row.Status >= 1 {
			result.Sections = append(result.Sections, row)
		}
	}

	core.Log("Webpage pública obtenida::", companyID, pageID, len(result.Sections))
	return req.MakeResponse(result)
}
