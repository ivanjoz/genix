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
	// domainChangeCooldownTicks is 20 minutes in the project's 2-second SUnixTime units.
	domainChangeCooldownTicks = int32(20 * 60 / 2)
	// La cuenta de desarrollo del proyecto, exenta del cooldown. El cooldown protege de gastar
	// las cuotas de Cloudflare a base de registrar y soltar hostnames, pero probar ese mismo
	// camino es lo que se hace desde esta cuenta, y esperar 20 minutos por intento lo impide.
	domainCooldownBypassCompanyID = int32(1)
	domainCooldownBypassUserID    = int32(1)
	// domainParameterKey holds the company's active storefront hostname.
	domainParameterKey = "domain"
	// previousDomainParameterKey marks a hostname that is pending release in Cloudflare. It is
	// written when the domain changes and cleared once the release is queued, so the cleanup
	// survives a failed render and the retry that follows it.
	previousDomainParameterKey = "domain_previous"
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

// PostWebsiteDomain reserva el hostname en Cloudflare, lo guarda, y publica la tienda sobre él
// antes de responder.
//
// El render es SÍNCRONO y bloquea la respuesta a propósito. El HTML en R2 está indexado por
// hostname y el Worker no tiene fallback de SPA, así que un dominio recién registrado resuelve y
// devuelve 404 en todas sus rutas hasta que alguien lo renderiza: dejarlo para después significaba
// tener la tienda caída —no degradada— durante esa ventana. El usuario espera unos segundos con el
// spinner que ya muestra WebpageConfig.svelte y no hay ningún estado intermedio roto.
//
// El orden importa y es el que hace segura la falla: se guarda, se renderiza, y solo si el render
// funcionó se libera el dominio anterior. Si el render falla, el hostname viejo sigue vivo en
// Cloudflare sirviendo su HTML, así que la tienda sigue en pie mientras se reintenta.
// canBypassDomainCooldown identifica a la cuenta de desarrollo del proyecto, la única que puede
// cambiar de dominio sin esperar. Se compara contra el token ya verificado, no contra nada que
// venga en el body, así que no es algo que un cliente pueda afirmar de sí mismo.
//
// Lo único que se salta es la espera: el hostname sigue pasando por la validación de formato y
// por la reserva en Cloudflare, que es donde se rechaza un nombre ocupado.
func canBypassDomainCooldown(user *core.UsuarioToken) bool {
	return user != nil &&
		user.CompanyID == domainCooldownBypassCompanyID &&
		user.ID == domainCooldownBypassUserID
}

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
	if currentDomain != nil && isDomainChange && !canBypassDomainCooldown(req.User) {
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

	// Solo se escribe en un cambio real, para que un guardado idempotente no alargue la ventana de
	// cooldown. Renderizar, en cambio, se hace en los dos casos: es lo único que hace que el
	// hostname sirva algo, y volver a guardar el mismo dominio es el único camino que tiene el
	// usuario para reintentar una publicación que falló.
	if isDomainChange {
		parametersToSave := []configTypes.Parameters{{
			CompanyID: req.User.CompanyID,
			Group:     webpageConfigGroup,
			Key:       domainParameterKey,
			Value:     domain,
			Status:    1,
			Updated:   nowTime,
			UpdatedBy: req.User.ID,
		}}
		if currentDomain != nil {
			// El dominio saliente se guarda en su propia fila para que la limpieza sobreviva a un
			// render fallido: en el reintento el guardado ya no es un cambio y currentDomain apunta
			// al nuevo, así que sin esto el anterior se perdería y quedaría huérfano en Cloudflare.
			parametersToSave = append(parametersToSave, configTypes.Parameters{
				CompanyID: req.User.CompanyID,
				Group:     webpageConfigGroup,
				Key:       previousDomainParameterKey,
				Value:     currentDomain.Value,
				Status:    1,
				Updated:   nowTime,
				UpdatedBy: req.User.ID,
			})
		}
		if err := db.Insert(&parametersToSave); err != nil {
			return req.MakeErr("Error al guardar el dominio:", err)
		}
		core.Log("Dominio del sitio guardado::", domain)
	}

	renderResult, renderError := RenderCompanyWebpage(req.User.CompanyID, domain)
	if renderError != nil {
		// El dominio queda guardado y el anterior sin liberar: la tienda sigue sirviéndose por el
		// hostname viejo y volver a guardar reintenta solo el render.
		core.Log("Error publicando la tienda::", domain, renderError)
		return req.MakeErr("El dominio se guardó, pero no se pudo publicar la tienda:", renderError)
	}

	// Recién ahora que el hostname nuevo sirve contenido se libera el anterior. Se lee de su fila y
	// no de currentDomain para cubrir el reintento tras un render fallido, donde este guardado ya
	// no es un cambio.
	releasePreviousDomain(req.User.CompanyID, domain, req.User.ID)

	return req.MakeResponse(map[string]any{
		"domain": domain,
		"pages":  renderResult.Pages,
		"build":  renderResult.BuildID,
	})
}

// getCompanyDomain returns the single upserted domain row and its last-change timestamp.
func getCompanyDomain(companyID int32) (*configTypes.Parameters, error) {
	return getCompanyWebpageParameter(companyID, domainParameterKey)
}

// getCompanyWebpageParameter reads one upserted row of the storefront config group, or nil when it
// was never written or was cleared.
func getCompanyWebpageParameter(companyID int32, key string) (*configTypes.Parameters, error) {
	parameters := []configTypes.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(companyID)
	query.Group.Equals(webpageConfigGroup)
	query.Key.Equals(key)
	if queryError := query.Exec(); queryError != nil {
		return nil, queryError
	}
	if len(parameters) == 0 || parameters[0].Status <= 0 || strings.TrimSpace(parameters[0].Value) == "" {
		return nil, nil
	}
	return &parameters[0], nil
}

// releasePreviousDomain programa la limpieza en Cloudflare del dominio que la company dejó de usar y
// borra la marca, de modo que se ejecute exactamente una vez.
//
// Solo se llama después de un render correcto: mientras el hostname nuevo no sirva contenido, el
// viejo es lo único que mantiene la tienda en pie. No devuelve error a propósito —el dominio ya está
// guardado y publicado, y un fallo aquí solo deja un registro huérfano en Cloudflare, que no es
// motivo para que el usuario vea fallar la operación.
func releasePreviousDomain(companyID int32, currentDomain string, userID int32) {
	previousDomain, readError := getCompanyWebpageParameter(companyID, previousDomainParameterKey)
	if readError != nil {
		core.Log("Error leyendo el dominio anterior::", companyID, readError)
		return
	}
	if previousDomain == nil {
		return
	}
	// Una vuelta al mismo dominio no libera nada: es el que la tienda está usando ahora.
	if previousDomain.Value == currentDomain {
		clearPreviousDomainMark(companyID, userID)
		return
	}

	schedulePreviousDomainCleanup(companyID, previousDomain.Value)
	// Se borra la marca al programar y no al ejecutar: la acción de cron ya reintenta por su cuenta,
	// y dejarla puesta haría que el próximo guardado volviera a encolar el mismo borrado.
	clearPreviousDomainMark(companyID, userID)
}

func clearPreviousDomainMark(companyID int32, userID int32) {
	clearedParameter := []configTypes.Parameters{{
		CompanyID: companyID,
		Group:     webpageConfigGroup,
		Key:       previousDomainParameterKey,
		Value:     "",
		Status:    0,
		Updated:   core.SUnixTime(),
		UpdatedBy: userID,
	}}
	if err := db.Insert(&clearedParameter); err != nil {
		core.Log("Error borrando la marca del dominio anterior::", companyID, err)
	}
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
