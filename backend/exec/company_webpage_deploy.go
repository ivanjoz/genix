package exec

import (
	configTypes "app/config/types"
	"app/core"
	"app/db"
	"app/webpage"
	"fmt"
	"strconv"
	"strings"
)

// La lista de páginas del sistema vive en el paquete webpage, junto al render que la usa.
const webpageConfigGroup = int32(10)

func DeployCompanyWebpage(args *core.ExecArgs) core.FuncResponse {
	companyID, argumentError := parseCompanyIDArgument(args.Message)
	if argumentError != nil {
		return args.MakeErr(argumentError)
	}

	hostname, domainError := getCompanyWebpageDomain(companyID)
	if domainError != nil {
		return args.MakeErr(domainError)
	}

	projectRoot, rootError := findGenixProjectRoot()
	if rootError != nil {
		return args.MakeErr(rootError)
	}
	// Los js/css se sirven desde el CDN a un dominio distinto al del sitio. Solo el CLI lo hace:
	// es una regla a nivel de bucket y necesita leer r2-cors.json del repo.
	if corsError := ensureCompanyWebpageAssetCORS(projectRoot); corsError != nil {
		return args.MakeErr(corsError)
	}

	fmt.Printf("[company-webpage] company=%d hostname=%s\n", companyID, hostname)

	// Misma publicación que hace el guardado de dominio: la lista de páginas y la llamada al
	// renderer viven en el paquete webpage para que los dos caminos no puedan divergir.
	result, renderError := webpage.RenderCompanyWebpage(companyID, hostname)
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
