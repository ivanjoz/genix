package types

import "github.com/ivanjoz/genix-orm/scylla"

type ImageAsset struct {
	scylla.TableStruct[ImageAssetTable, ImageAsset]
	GroupID            int32    `json:"-"`
	ID                 int32    `json:",omitempty"`
	CategoryID         int16    `json:",omitempty" db:"category_id"`
	Description        string   `json:",omitempty"`            // English description (IMAGES_LIST.md).
	SpanishDescription string   `json:",omitempty"`            // Spanish description (IMAGES_LIST.ES.md).
	Keywords           string   `json:",omitempty"`            // Deduplicated English words; feeds the TextSearchColumn AI search.
	SpanishKeywords    []string `json:",omitempty" db:",list"` // Spanish object list; feeds the bigram local search.
	Bigrams            []int8   `json:",omitempty" db:",list"` // Encoded from the Spanish keywords for frontend local search.
	Ratio              float32  `json:",omitempty"`            // Aspect ratio = width/height (1.0=1:1, 1.777=16:9, 0.75=3:4). 0 ⇒ unknown, treated as 1:1 by find_image.
	Updated            int32    `json:"upd,omitempty"`
}

type ImageAssetTable struct {
	scylla.TableStruct[ImageAssetTable, ImageAsset]
	GroupID            scylla.Col[ImageAssetTable, int32]
	ID                 scylla.Col[ImageAssetTable, int32]
	CategoryID         scylla.Col[ImageAssetTable, int16]
	Description        scylla.Col[ImageAssetTable, string]
	SpanishDescription scylla.Col[ImageAssetTable, string]
	Keywords           scylla.Col[ImageAssetTable, string]
	SpanishKeywords    scylla.ColSlice[ImageAssetTable, string]
	Bigrams            scylla.ColSlice[ImageAssetTable, int8]
	Ratio              scylla.Col[ImageAssetTable, float32]
	Updated            scylla.Col[ImageAssetTable, int32]
}

func (e ImageAssetTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "image_assets",
		Partition: e.GroupID,
		// Keywords holds the deduplicated English text indexed by the Sonic AI search.
		TextSearchColumn: e.Keywords,
		Keys:             scylla.Cols(e.ID),
		Indexes: []scylla.Index{
			// Keep Updated as the first clustering column for global frontend deltas.
			{Type: scylla.TypeView, Keys: scylla.Cols(e.Updated)},
		},
	}
}
