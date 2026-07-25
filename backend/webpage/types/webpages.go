package types

import "github.com/ivanjoz/genix-orm/scylla"

// Webpage is one storefront page in the builder. Its ID is reused as the PageID
// of EcommercePageContent, so it must stay within int16. IDs 1-9 are reserved;
// 10-14 are the always-injected system pages (/, /about, /store, /product,
// /cart) which are NOT stored here — the GET handler injects them. User-created
// pages autoincrement from 15.
type Webpage struct {
	scylla.TableStruct[WebpageTable, Webpage]
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
	scylla.TableStruct[WebpageTable, Webpage]
	CompanyID scylla.Col[WebpageTable, int32]
	ID        scylla.Col[WebpageTable, int16]
	Name      scylla.Col[WebpageTable, string]
	Route     scylla.Col[WebpageTable, string]
	Image     scylla.Col[WebpageTable, int32]
	Status    scylla.Col[WebpageTable, int8]
	Updated   scylla.Col[WebpageTable, int32]
	UpdatedBy scylla.Col[WebpageTable, int32]
}

func (e WebpageTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "webpages",
		Partition: e.CompanyID,
		Keys:      scylla.Cols(e.ID.Autoincrement(0)),
		Indexes: []scylla.Index{
			// Two narrow views: the initial fetch filters by Status only, the delta
			// fetch by Updated only — never ANDed — so each gets its own view.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Status)},
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
