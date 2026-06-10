package types

import "app/db"

// Webpage is one storefront page in the builder. Its ID is reused as the PageID
// of EcommercePageContent, so it must stay within int16. IDs 1-9 are reserved;
// 10-14 are the always-injected system pages (/, /about, /store, /product,
// /cart) which are NOT stored here — the GET handler injects them. User-created
// pages autoincrement from 15.
type Webpage struct {
	db.TableStruct[WebpageTable, Webpage]
	CompanyID int32  `json:",omitempty"`
	ID        int16  `json:",omitempty"`
	Name      string `json:",omitempty"`
	Route     string `json:",omitempty"`
	// Image is the numeric ID of the page's thumbnail image. Image upload/resolution
	// is handled separately; this column only stores the reference.
	Image int32 `json:",omitempty"`
	// Status: 0 removed, 1 active, 2 published.
	Status    int8  `json:"ss,omitempty"`
	Updated   int32 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
}

type WebpageTable struct {
	db.TableStruct[WebpageTable, Webpage]
	CompanyID db.Col[WebpageTable, int32]
	ID        db.Col[WebpageTable, int16]
	Name      db.Col[WebpageTable, string]
	Route     db.Col[WebpageTable, string]
	Image     db.Col[WebpageTable, int32]
	Status    db.Col[WebpageTable, int8]
	Updated   db.Col[WebpageTable, int32]
	UpdatedBy db.Col[WebpageTable, int32]
}

func (e WebpageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "webpages",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID.Autoincrement(0)},
		Indexes: []db.Index{
			// Two narrow views: the initial fetch filters by Status only, the delta
			// fetch by Updated only — never ANDed — so each gets its own view.
			{Type: db.TypeView, Keys: []db.Coln{e.Status}, KeepPart: true},
			{Type: db.TypeView, Keys: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
