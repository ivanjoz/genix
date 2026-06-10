package webpage

import (
	configTypes "app/config/types"
	"app/core"
	"app/db"
	"encoding/json"
	"strings"
)

// webpageConfigGroup is the parameters Group that stores all storefront config
// (SEO metatags + domain) for a company.
const webpageConfigGroup = int32(10)

// seoMetatagKeys are the SEO parameter keys persisted under the config group. The
// domain is stored separately under the "domain" key by PostWebsiteDomain.
var seoMetatagKeys = []string{"title", "description", "keywords", "ogTitle", "ogDescription", "ogImage", "favicon"}

// GetWebsiteConfig returns the company's storefront config (domain + SEO metatags)
// as a flat key -> value map read from the parameters table (Group 10). It is
// company-scoped server-side so no tenant data leaks.
func GetWebsiteConfig(req *core.HandlerArgs) core.HandlerResponse {
	parameters := []configTypes.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(req.User.CompanyID)
	query.Group.Equals(webpageConfigGroup)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la configuración del sitio:", err)
	}

	config := map[string]string{}
	for _, parameter := range parameters {
		if parameter.Status > 0 {
			config[parameter.Key] = parameter.Value
		}
	}
	return req.MakeResponse(config)
}

// publicSeoMetatags reads the company's SEO metatags (group 10) and returns only the
// known SEO keys — never the domain or any other parameter. Shared by the public
// webpage read.
func publicSeoMetatags(companyID int32) (map[string]string, error) {
	parameters := []configTypes.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(companyID)
	query.Group.Equals(webpageConfigGroup)
	if err := query.Exec(); err != nil {
		return nil, err
	}

	seoKeys := map[string]bool{}
	for _, key := range seoMetatagKeys {
		seoKeys[key] = true
	}

	config := map[string]string{}
	for _, parameter := range parameters {
		if parameter.Status > 0 && seoKeys[parameter.Key] {
			config[parameter.Key] = parameter.Value
		}
	}
	return config, nil
}

// PostWebsiteSeo upserts the SEO metatags into the parameters table (Group 10),
// one row per known key. CompanyID and audit fields are set server-side.
func PostWebsiteSeo(req *core.HandlerArgs) core.HandlerResponse {
	incoming := map[string]string{}
	if err := json.Unmarshal([]byte(*req.Body), &incoming); err != nil {
		return req.MakeErr("Error al deserializar los metatags:", err)
	}

	nowTime := core.SUnixTime()
	parameters := []configTypes.Parameters{}
	// Only persist the known SEO keys so the client can't write arbitrary parameters.
	for _, key := range seoMetatagKeys {
		parameters = append(parameters, configTypes.Parameters{
			CompanyID: req.User.CompanyID,
			Group:     webpageConfigGroup,
			Key:       key,
			Value:     strings.TrimSpace(incoming[key]),
			Status:    1,
			Updated:   nowTime,
			UpdatedBy: req.User.ID,
		})
	}

	if err := db.Insert(&parameters); err != nil {
		return req.MakeErr("Error al guardar los metatags SEO:", err)
	}

	core.Log("Metatags SEO guardados::", len(parameters))
	return req.MakeResponse(map[string]bool{"saved": true})
}

// PostWebsiteDomain validates and stores the storefront domain in the parameters
// table under Group 10, Key "domain". It has its own endpoint because the domain
// will later drive DNS / Cloudflare provisioning.
func PostWebsiteDomain(req *core.HandlerArgs) core.HandlerResponse {
	body := struct {
		Domain string `json:"Domain"`
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el dominio:", err)
	}

	// Normalize: drop protocol/path and surrounding whitespace so we store a bare host.
	domain := strings.ToLower(strings.TrimSpace(body.Domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(strings.Split(domain, "/")[0], ".")

	// Minimal host validation: a label.tld shape, no spaces.
	if len(domain) < 4 || strings.Contains(domain, " ") || !strings.Contains(domain, ".") {
		return req.MakeErr("El dominio no es válido:", body.Domain)
	}

	domainParameter := []configTypes.Parameters{{
		CompanyID: req.User.CompanyID,
		Group:     webpageConfigGroup,
		Key:       "domain",
		Value:     domain,
		Status:    1,
		Updated:   core.SUnixTime(),
		UpdatedBy: req.User.ID,
	}}
	if err := db.Insert(&domainParameter); err != nil {
		return req.MakeErr("Error al guardar el dominio:", err)
	}

	core.Log("Dominio del sitio guardado::", domain)
	return req.MakeResponse(map[string]string{"domain": domain})
}
