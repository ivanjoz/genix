package webpage

import (
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	s "app/webpage/types"
	"encoding/json"
	"strings"
)

// firstUserPageID is the first ID available to user-created pages. IDs 1-9 are
// reserved and 10-14 belong to the injected system pages, so user pages start at 15.
const firstUserPageID = int16(15)

// systemRoutes are the routes of the fixed system pages (/, /about, /store,
// /product, /cart). They are injected client-side (not stored), so here they only
// serve to reserve those routes against user pages. Keep in sync with the
// frontend SYSTEM_PAGES.
var systemRoutes = []string{"/", "/about", "/store", "/product", "/cart"}

// GetWebpages returns the company's stored (user-created) pages via delta cache.
// The fixed system pages are injected by the frontend, not here, so they always
// exist without depending on a server round-trip.
func GetWebpages(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))

	pages := []s.Webpage{}
	query := db.Query(&pages).CompanyID.Equals(req.User.CompanyID)
	if updated > 0 {
		query.Updated.GreaterThan(updated) // delta: include Status=0 rows so the client can evict them
	} else {
		query.Status.GreaterEqual(1) // initial: active + published only
	}
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener las páginas web:", err)
	}

	core.Log("Páginas web obtenidas::", len(pages))
	return core.MakeResponse(req, &pages)
}

// PostWebpage upserts the company's user-defined pages. It enforces: reserved IDs
// (<= 14) cannot be written; new pages get sequential IDs starting at 15; routes
// must start with "/", be unique, and not collide with a system route.
func PostWebpage(req *core.HandlerArgs) core.HandlerResponse {
	incomingPages := []s.Webpage{}
	if err := json.Unmarshal([]byte(*req.Body), &incomingPages); err != nil {
		return req.MakeErr("Error al deserializar las páginas:", err)
	}
	if len(incomingPages) == 0 {
		return req.MakeErr("No se recibieron páginas para guardar.")
	}

	// Reserve the system routes so user pages can never shadow them.
	reservedRoutes := map[string]bool{}
	for _, route := range systemRoutes {
		reservedRoutes[route] = true
	}

	// Load current stored pages to validate route uniqueness and compute the next ID.
	currentPages := []s.Webpage{}
	currentQuery := db.Query(&currentPages).CompanyID.Equals(req.User.CompanyID)
	if err := currentQuery.Exec(); err != nil {
		return req.MakeErr("Error al leer las páginas actuales:", err)
	}

	// Map active stored routes to their owning ID so we can detect collisions, and
	// track the highest used ID to keep new IDs monotonic and >= firstUserPageID.
	routeOwnerID := map[string]int16{}
	maxUsedID := firstUserPageID - 1
	for _, storedPage := range currentPages {
		if storedPage.ID > maxUsedID {
			maxUsedID = storedPage.ID
		}
		if storedPage.Status > 0 {
			routeOwnerID[storedPage.Route] = storedPage.ID
		}
	}

	nowTime := core.SUnixTime()
	newIDs := []businessTypes.NewIDToID{}

	for index := range incomingPages {
		page := &incomingPages[index]

		// A page in the reserved/system range is injected, not stored — reject writes.
		if page.ID > 0 && page.ID < firstUserPageID {
			return req.MakeErr("Las páginas del sistema no se pueden editar (ID:", page.ID, ").")
		}
		if len(strings.TrimSpace(page.Name)) < 3 {
			return req.MakeErr("El nombre de la página debe tener al menos 3 caracteres.")
		}

		page.Route = strings.TrimSpace(page.Route)
		if !strings.HasPrefix(page.Route, "/") || page.Route == "/" {
			return req.MakeErr("La ruta debe iniciar con \"/\" y no puede ser la raíz. Ruta:", page.Route)
		}
		if reservedRoutes[page.Route] {
			return req.MakeErr("La ruta", page.Route, "está reservada por una página del sistema.")
		}
		// Route must be unique among stored pages (a route owned by another ID collides).
		if ownerID, exists := routeOwnerID[page.Route]; exists && ownerID != page.ID {
			return req.MakeErr("La ruta", page.Route, "ya está en uso por otra página.")
		}

		// Preserve the incoming (possibly temporary/negative) ID for the response map.
		newIDs = append(newIDs, businessTypes.NewIDToID{TempID: int32(page.ID)})

		// Assign a fresh sequential ID to new pages instead of relying on the ORM
		// autoincrement, so reserved IDs (<= 14) are never produced.
		if page.ID <= 0 {
			maxUsedID++
			page.ID = maxUsedID
		}

		page.CompanyID = req.User.CompanyID
		page.Updated = nowTime
		page.UpdatedBy = req.User.ID
		routeOwnerID[page.Route] = page.ID
	}

	if err := db.Insert(&incomingPages); err != nil {
		return req.MakeErr("Error al guardar las páginas:", err)
	}

	// Complete the TempID -> final ID mapping for the client cache.
	for index := range incomingPages {
		newIDs[index].ID = int32(incomingPages[index].ID)
	}

	core.Log("Páginas web guardadas::", len(incomingPages))
	return req.MakeResponse(newIDs)
}
