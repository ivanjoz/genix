package webpage

import (
	"app/cloud"
	"app/core"
	"app/db"
	s "app/webpage/types"
	"encoding/json"
	"fmt"
	"strings"
)

// showcaseImageFolder is the CDN/bucket folder for page thumbnail images. One
// object per page: "<companyID>_<imageID>.avif" (single resolution).
const showcaseImageFolder = "img-webpage"

// systemPageRoutes maps each injected system page ID to its route. System pages
// (10-14) are normally not stored, but a showcase upload persists a minimal row
// so the page's thumbnail Image reference can live somewhere. Keep in sync with
// systemRoutes / the frontend SYSTEM_PAGES.
var systemPageRoutes = map[int16]string{
	10: "/", 11: "/about", 12: "/store", 13: "/product", 14: "/cart",
}

// showcaseImageBody is the request payload: a single client-converted AVIF as a
// base64 data-url. The conversion (screenshot -> ~0.4 Mpx AVIF) happens on the
// client, so the server stores the bytes as-is (no re-conversion).
type showcaseImageBody struct {
	Content string
}

// PostWebpageShowcaseImage stores a page's showcase thumbnail. The client sends a
// single pre-converted AVIF; this handler reserves an imageID ending in 0 (single
// image, vs the product scheme's 7), uploads the bytes to the CDN, and writes the
// imageID into the page's Image column (the cache-buster — a fresh id per save
// changes the URL). For a system page (10-14, not normally stored) it creates a
// minimal row so the reference has a home.
func PostWebpageShowcaseImage(req *core.HandlerArgs) core.HandlerResponse {
	pageID := int16(req.GetQueryInt("page-id"))
	if pageID <= 0 {
		return req.MakeErr("Falta el parámetro [page-id] o es inválido.")
	}

	body := showcaseImageBody{}
	if err := json.Unmarshal([]byte(*req.Body), &body); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	// Drop an optional "...;base64," data-url prefix, then validate there is content.
	content := body.Content
	if idx := strings.Index(content, "base64,"); idx >= 0 {
		content = content[idx+len("base64,"):]
	}
	if len(content) < 40 {
		return req.MakeErr("No se recibió el contenido de la imagen.")
	}
	imageBytes := core.Base64ToBytes(content)
	if len(imageBytes) == 0 {
		return req.MakeErr("Error al convertir el contenido de la imagen a bytes.")
	}

	// Reserve a per-company image autoincrement; the last digit 0 marks a single
	// image (the product upload uses 7 for its multi-resolution set).
	autoincrement, err := db.GetAutoincrementID(fmt.Sprintf("images_%v", req.User.CompanyID), 1)
	if err != nil {
		return req.MakeErr("Error al reservar el id de imagen:", err)
	}
	imageID := int32(autoincrement*10) + 0

	// Upload the AVIF directly (the client already converted it).
	saveErr := cloud.SaveFile(cloud.SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        showcaseImageFolder,
		Name:        fmt.Sprintf("%v_%v.avif", req.User.CompanyID, imageID),
		FileContent: imageBytes,
		ContentType: "image/avif",
	})
	if saveErr != nil {
		return req.MakeErr("Error al guardar la imagen:", saveErr)
	}

	// Load the existing page row to update its Image without wiping Name/Route.
	pages := []s.Webpage{}
	query := db.Query(&pages).CompanyID.Equals(req.User.CompanyID)
	query.ID.Equals(pageID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al leer la página:", err)
	}

	var page s.Webpage
	if len(pages) > 0 {
		page = pages[0]
	} else {
		// System page (10-14): not stored yet — create a minimal active row so its
		// thumbnail reference has a home. Other pages must already exist.
		route, isSystem := systemPageRoutes[pageID]
		if !isSystem {
			return req.MakeErr("No se encontró la página con ID:", pageID)
		}
		page = s.Webpage{ID: pageID, Name: route, Route: route, Status: 1}
	}
	page.CompanyID = req.User.CompanyID
	page.Image = imageID
	page.Updated = core.SUnixTime()
	page.UpdatedBy = req.User.ID

	if err := db.Insert(&[]s.Webpage{page}); err != nil {
		return req.MakeErr("Error al actualizar la página:", err)
	}

	core.Log("Imagen de vista previa guardada:: página", pageID, "imageID", imageID)
	return req.MakeResponse(map[string]any{"ImageID": imageID})
}
