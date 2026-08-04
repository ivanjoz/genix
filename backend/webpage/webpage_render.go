package webpage

import (
	"app/cloud"
	"app/db"
	s "app/webpage/types"
	"fmt"
	"strings"
)

const (
	// Páginas de sistema con contenido editable en el builder. Mantener en sync con
	// SYSTEM_PAGES (frontend/services/webpage/pages.svelte.ts).
	webpageHomePageID  = int16(10)
	webpageAboutPageID = int16(11)
	// Los IDs <= 14 están reservados al sistema; las páginas de usuario empiezan en 15.
	lastSystemWebpageID = int16(14)
)

// RenderCompanyWebpage renderiza y publica en Cloudflare todas las páginas prerenderizadas de una
// company sobre hostname. Es SÍNCRONA: quien la llama espera a que termine.
//
// Vive aquí y no en el CLI de deploy porque hay dos llamadores que no pueden divergir: el deploy
// manual y el guardado de dominio. El HTML en R2 está indexado por hostname, así que un hostname
// sin render no sirve nada —el Worker no tiene fallback de SPA— y publicar es parte de cambiar el
// dominio, no un paso posterior opcional.
//
// No configura el CORS del bucket a propósito: es una regla a nivel de bucket, se fija una sola vez
// en el deploy de infraestructura y necesita leer r2-cors.json del repo, que en Lambda no existe.
func RenderCompanyWebpage(companyID int32, hostname string) (cloud.WebpageRenderResult, error) {
	pages, pagesError := companyWebpageRenderPages(companyID)
	if pagesError != nil {
		return cloud.WebpageRenderResult{}, pagesError
	}

	return cloud.InvokeWebpageRenderer(cloud.WebpageRenderRequest{
		CompanyID: companyID,
		Hostname:  hostname,
		Pages:     pages,
	})
}

// companyWebpageRenderPages arma la lista de páginas a renderizar: la raíz y /about —las dos
// páginas de sistema con contenido editable en el builder— más las páginas creadas por el usuario
// que estén activas. Las demás páginas de sistema (/store, /product, /cart) son dinámicas y las
// resuelve el cliente, así que no se prerenderizan.
func companyWebpageRenderPages(companyID int32) ([]cloud.WebpageRenderPage, error) {
	pages := []cloud.WebpageRenderPage{
		{ID: webpageHomePageID, Path: "/"},
		{ID: webpageAboutPageID, Path: "/about"},
	}

	storedPages := []s.Webpage{}
	query := db.Query(&storedPages).CompanyID.Equals(companyID)
	if queryError := query.Exec(); queryError != nil {
		return nil, fmt.Errorf("error consultando las páginas de CompanyID %d: %w", companyID, queryError)
	}

	for _, storedPage := range storedPages {
		// Los IDs <= 14 están reservados a las páginas de sistema; una fila en ese rango solo
		// guarda su miniatura, no es una página propia.
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
