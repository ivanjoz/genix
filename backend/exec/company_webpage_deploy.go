package exec

import (
	"app/cloud"
	configTypes "app/config/types"
	"app/core"
	"app/db"
	webpageTypes "app/webpage/types"
	"fmt"
	"strconv"
	"strings"
)

const (
	webpageConfigGroup = int32(10)
	// Páginas de sistema con contenido editable en el builder. Mantener en sync con
	// SYSTEM_PAGES (frontend/services/webpage/pages.svelte.ts) y con defaultPageID.
	webpageHomePageID  = int16(10)
	webpageAboutPageID = int16(11)
	// Los IDs <= 14 están reservados al sistema; las páginas de usuario empiezan en 15.
	lastSystemWebpageID = int16(14)
)

func DeployCompanyWebpage(args *core.ExecArgs) core.FuncResponse {
	companyID, argumentError := parseCompanyIDArgument(args.Message)
	if argumentError != nil {
		return args.MakeErr(argumentError)
	}

	hostname, domainError := getCompanyWebpageDomain(companyID)
	if domainError != nil {
		return args.MakeErr(domainError)
	}

	pages, pagesError := getCompanyWebpagePages(companyID)
	if pagesError != nil {
		return args.MakeErr(pagesError)
	}

	projectRoot, rootError := findGenixProjectRoot()
	if rootError != nil {
		return args.MakeErr(rootError)
	}
	// Los js/css se sirven desde el CDN a un dominio distinto al del sitio.
	if corsError := ensureCompanyWebpageAssetCORS(projectRoot); corsError != nil {
		return args.MakeErr(corsError)
	}

	fmt.Printf("[company-webpage] company=%d hostname=%s páginas=%d\n", companyID, hostname, len(pages))

	// El render vive en una Lambda de Node: el servidor SSR de SvelteKit es JavaScript, y
	// así publicar no depende de tener el monorepo y bun en la máquina que ejecuta esto.
	result, renderError := cloud.InvokeWebpageRenderer(cloud.WebpageRenderRequest{
		CompanyID: companyID,
		Hostname:  hostname,
		Pages:     pages,
	})
	if renderError != nil {
		return args.MakeErr(renderError)
	}

	if provisionError := provisionStorefrontDomain(hostname); provisionError != nil {
		return args.MakeErr(provisionError)
	}

	return core.FuncResponse{
		Message: fmt.Sprintf("Webpage de CompanyID %d desplegada en https://%s (%d página(s), build %s)",
			companyID, hostname, result.Pages, result.BuildID),
	}
}

// getCompanyWebpagePages arma la lista de páginas a renderizar: la raíz y /about —las dos
// páginas de sistema con contenido editable en el builder— más las páginas creadas por el
// usuario que estén activas. Las demás páginas de sistema (/store, /product, /cart) son
// dinámicas y las resuelve el cliente, así que no se prerenderizan.
func getCompanyWebpagePages(companyID int32) ([]cloud.WebpageRenderPage, error) {
	pages := []cloud.WebpageRenderPage{
		{ID: webpageHomePageID, Path: "/"},
		{ID: webpageAboutPageID, Path: "/about"},
	}

	storedPages := []webpageTypes.Webpage{}
	query := db.Query(&storedPages).CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return nil, fmt.Errorf("error consultando las páginas de CompanyID %d: %w", companyID, queryError)
	}

	for _, storedPage := range storedPages {
		// Los IDs <= 14 están reservados a las páginas de sistema; una fila en ese rango
		// solo guarda su miniatura, no es una página propia.
		if storedPage.ID <= lastSystemWebpageID || storedPage.Status <= 0 {
			continue
		}
		route := strings.TrimSpace(storedPage.Route)
		if !strings.HasPrefix(route, "/") || route == "/" {
			return nil, fmt.Errorf("la página %d tiene una ruta inválida: %q", storedPage.ID, storedPage.Route)
		}
		pages = append(pages, cloud.WebpageRenderPage{ID: storedPage.ID, Path: route})
	}

	return pages, nil
}

func parseCompanyIDArgument(rawArgument string) (int32, error) {
	arguments := strings.Fields(rawArgument)
	if len(arguments) != 1 {
		return 0, fmt.Errorf("fn-deploy-company-webpage requiere exactamente un CompanyID")
	}

	companyID64, parseError := strconv.ParseInt(arguments[0], 10, 32)
	if parseError != nil || companyID64 <= 0 {
		return 0, fmt.Errorf("CompanyID debe ser un entero positivo")
	}

	return int32(companyID64), nil
}

func getCompanyWebpageDomain(companyID int32) (string, error) {
	parameters := []configTypes.Parameters{}
	query := db.Query(&parameters).CompanyID.Equals(companyID)
	query.Group.Equals(webpageConfigGroup)
	query.Key.Equals("domain")
	if queryError := query.Exec(); queryError != nil {
		return "", fmt.Errorf("error consultando el dominio de CompanyID %d: %w", companyID, queryError)
	}

	activeDomains := []string{}
	for _, parameter := range parameters {
		if parameter.Status > 0 && strings.TrimSpace(parameter.Value) != "" {
			activeDomains = append(activeDomains, parameter.Value)
		}
	}
	if len(activeDomains) != 1 {
		return "", fmt.Errorf(
			"CompanyID %d debe tener exactamente un dominio activo; encontrados: %d",
			companyID,
			len(activeDomains),
		)
	}

	hostname, hostnameError := normalizeAndValidateHostname(activeDomains[0])
	if hostnameError != nil {
		return "", hostnameError
	}
	return hostname, nil
}
