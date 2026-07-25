package webpage

import (
	configTypes "app/config/types"
	"app/core"
	"app/db"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// webpageConfigGroup stores all storefront configuration for a company.
	webpageConfigGroup = int32(10)
	// domainChangeCooldownTicks is 60 minutes in the project's 2-second SUnixTime units.
	domainChangeCooldownTicks = int32(60 * 60 / 2)
)

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

// PostWebsiteDomain reserves the hostname in Cloudflare before storing it for the company.
func PostWebsiteDomain(req *core.HandlerArgs) core.HandlerResponse {
	body := struct {
		Domain string `json:"Domain"`
	}{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el dominio:", err)
	}

	domain, domainError := normalizeStorefrontDomain(body.Domain)
	if domainError != nil {
		return req.MakeErr(domainError.Error())
	}

	currentDomain, readError := getCompanyDomain(req.User.CompanyID)
	if readError != nil {
		return req.MakeErr("Error al obtener el dominio actual:", readError)
	}

	nowTime := core.SUnixTime()
	isDomainChange := currentDomain == nil || currentDomain.Value != domain
	if currentDomain != nil && isDomainChange {
		elapsedTicks := nowTime - currentDomain.Updated
		if elapsedTicks < domainChangeCooldownTicks {
			remainingMinutes := (domainChangeCooldownTicks - elapsedTicks + 29) / 30
			return req.MakeErr(fmt.Sprintf(
				"Debe esperar %d minuto(s) antes de cambiar nuevamente el dominio.",
				remainingMinutes,
			))
		}
	}

	core.Log("Verificando dominio Cloudflare::", domain)
	if provisionError := provisionStorefrontDomain(domain, !isDomainChange); provisionError != nil {
		return req.MakeErr("No se pudo registrar el dominio:", provisionError)
	}

	// Keep an idempotent save from extending the cooldown window.
	if !isDomainChange {
		return req.MakeResponse(map[string]string{"domain": domain})
	}

	domainParameter := []configTypes.Parameters{{
		CompanyID: req.User.CompanyID,
		Group:     webpageConfigGroup,
		Key:       "domain",
		Value:     domain,
		Status:    1,
		Updated:   nowTime,
		UpdatedBy: req.User.ID,
	}}
	if err := db.Insert(&domainParameter); err != nil {
		return req.MakeErr("Error al guardar el dominio:", err)
	}

	core.Log("Dominio del sitio guardado::", domain)
	return req.MakeResponse(map[string]string{"domain": domain})
}

// getCompanyDomain returns the single upserted domain row and its last-change timestamp.
func getCompanyDomain(companyID int32) (*configTypes.Parameters, error) {
	parameters := []configTypes.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(companyID)
	query.Group.Equals(webpageConfigGroup)
	query.Key.Equals("domain")
	if queryError := query.Exec(); queryError != nil {
		return nil, queryError
	}
	if len(parameters) == 0 || parameters[0].Status <= 0 {
		return nil, nil
	}
	return &parameters[0], nil
}

// normalizeStorefrontDomain accepts only direct subdomains of the configured Cloudflare zone.
func normalizeStorefrontDomain(rawDomain string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(rawDomain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(strings.Split(domain, "/")[0], ".")

	zoneName := strings.ToLower(strings.TrimSpace(core.Env.ZONE_NAME))
	if zoneName == "" {
		zoneName = "un.pe"
	}
	subdomain := strings.TrimSuffix(domain, "."+zoneName)
	if subdomain == domain || subdomain == "" || strings.Contains(subdomain, ".") {
		return "", fmt.Errorf("el dominio debe tener el formato nombre.%s", zoneName)
	}
	for _, character := range subdomain {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return "", fmt.Errorf("el subdominio solo puede contener letras, números y guiones")
		}
	}
	if len(subdomain) > 63 || subdomain[0] == '-' || subdomain[len(subdomain)-1] == '-' {
		return "", fmt.Errorf("el subdominio no es válido")
	}
	return domain, nil
}
