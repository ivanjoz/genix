package webpage

import (
	"app/core"
	"app/db"
	s "app/webpage/types"
	"encoding/json"
	"hash/fnv"
)

// defaultPageID is the "Inicio" page (ID 10 in the webpages table). It is used
// when no explicit page-id query param is provided (the bare /webpage-builder
// route). All other pages pass their PageID explicitly.
const defaultPageID = int16(10)

// sectionHash is the FNV-1a 64-bit hash of a section's JSON. It detects whether a
// section's content changed since the last save and is computed server-side only
// — the client never sends a hash.
func sectionHash(content s.SectionContent) int64 {
	raw, _ := json.Marshal(content)
	hasher := fnv.New64a()
	hasher.Write(raw)
	return int64(hasher.Sum64())
}

// resolvePageID reads the page-id query param, defaulting to the Inicio page.
func resolvePageID(req *core.HandlerArgs) int16 {
	pageID := int16(req.GetQueryInt("page-id"))
	if pageID == 0 {
		pageID = defaultPageID
	}
	return pageID
}

// GetPageContent returns a page's active sections in position order so the
// builder can reload them. (CompanyID, PageID) scopes the read.
func GetPageContent(req *core.HandlerArgs) core.HandlerResponse {
	pageID := resolvePageID(req)

	rows := []s.EcommercePageContent{}
	query := db.Query(&rows)
	query.Select().CompanyID.Equals(req.User.CompanyID)
	query.PageID.Equals(pageID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener el contenido de la página:", err)
	}

	// Drop soft-deleted positions; rows come back in clustering (SectionID) order.
	activeSections := []s.EcommercePageContent{}
	for _, row := range rows {
		if row.Status >= 1 {
			activeSections = append(activeSections, row)
		}
	}

	core.Log("Secciones obtenidas::", len(activeSections))
	return req.MakeResponse(activeSections)
}

// PostPageContent upserts a page's sections. The client sends every current
// section in order (each carrying its pre-generated PageCss); the server assigns
// the 1-based SectionID, hashes each, and writes only new/changed rows,
// soft-deleting positions that no longer exist.
func PostPageContent(req *core.HandlerArgs) core.HandlerResponse {
	pageID := resolvePageID(req)

	incomingSections := []s.SectionContent{}
	if err := json.Unmarshal([]byte(*req.Body), &incomingSections); err != nil {
		return req.MakeErr("Error al procesar las secciones:", err)
	}

	// Load current rows for this page to compare hashes and detect removed positions.
	currentRows := []s.EcommercePageContent{}
	currentQuery := db.Query(&currentRows)
	currentQuery.Select().CompanyID.Equals(req.User.CompanyID)
	currentQuery.PageID.Equals(pageID)
	if err := currentQuery.Exec(); err != nil {
		return req.MakeErr("Error al leer el contenido actual:", err)
	}
	currentBySection := map[int16]s.EcommercePageContent{}
	for _, row := range currentRows {
		currentBySection[row.SectionID] = row
	}

	now := core.SUnixTime()
	sectionsToWrite := []s.EcommercePageContent{}

	for index, content := range incomingSections {
		sectionID := int16(index + 1)

		// The whole-page stylesheet rides in section 1's PageCss only. Move it to the
		// top-level Css column (cheap to serve) and clear it from the content blob so
		// it is not stored twice.
		pageCss := content.PageCss
		content.PageCss = ""

		// Hash the content AFTER clearing PageCss so it matches what is actually
		// stored. The Css is compared separately: a whole-page CSS change (e.g. a
		// different section was edited) doesn't alter section 1's own content, so the
		// hash alone wouldn't catch it.
		hash := sectionHash(content)

		// Skip sections that are unchanged and still active — the dedup goal.
		if prev, exists := currentBySection[sectionID]; exists && prev.Status == 1 && prev.Hash == hash && prev.Css == pageCss {
			continue
		}

		sectionsToWrite = append(sectionsToWrite, s.EcommercePageContent{
			CompanyID: req.User.CompanyID,
			PageID:    pageID,
			SectionID: sectionID,
			Content:   content,
			Css:       pageCss,
			Hash:      hash,
			Status:    1,
			Updated:   now,
			UpdatedBy: req.User.ID,
		})
	}

	// Soft-delete positions beyond the new length that are still active
	// (e.g. the page shrank from 12 to 10 sections).
	sectionCount := int16(len(incomingSections))
	sectionsToDelete := []s.EcommercePageContent{}
	for sectionID, row := range currentBySection {
		if sectionID > sectionCount && row.Status == 1 {
			row.Status = 0
			row.Updated = now
			row.UpdatedBy = req.User.ID
			sectionsToDelete = append(sectionsToDelete, row)
		}
	}

	if len(sectionsToWrite) > 0 {
		if err := db.Insert(&sectionsToWrite); err != nil {
			return req.MakeErr("Error al guardar las secciones:", err)
		}
	}
	if len(sectionsToDelete) > 0 {
		table := db.Table[s.EcommercePageContent]()
		if err := db.Update(&sectionsToDelete, table.Status, table.Updated, table.UpdatedBy); err != nil {
			return req.MakeErr("Error al eliminar las secciones removidas:", err)
		}
	}

	core.Log("Secciones guardadas::", len(sectionsToWrite), "eliminadas::", len(sectionsToDelete))
	return req.MakeResponse(map[string]any{"saved": len(sectionsToWrite), "deleted": len(sectionsToDelete)})
}
