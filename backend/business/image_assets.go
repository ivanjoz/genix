package business

import (
	"app/business/types"
	"app/core"
	"app/db"
	"encoding/base64"

	"golang.org/x/sync/errgroup"
)

// ImageAssetSearchRecord is the minimal delta payload consumed by frontend search. Version and
// Metadata travel with it because the client follows this route's delta log.
type ImageAssetSearchRecord struct {
	ID         int32   `json:",omitempty" cb:"ID,1"`
	CategoryID int16   `json:",omitempty" cb:"CategoryID,2"`
	Bigrams    string  `json:",omitempty" cb:"Bigrams,3"`
	Updated    int32   `json:"upd,omitempty" cb:"upd,4"`
	Version    int16   `json:"ver,omitempty" cb:"ver,5"`
	Metadata   *string `json:"mdt,omitempty" cb:"mdt,6"`
}

type ImageAssetCategoryRecord struct {
	ID       int16   `json:",omitempty" cb:"ID,1"`
	Name     string  `json:",omitempty" cb:"Name,2"`
	Updated  int32   `json:"upd,omitempty" cb:"upd,3"`
	Version  int16   `json:"ver,omitempty" cb:"ver,4"`
	Metadata *string `json:"mdt,omitempty" cb:"mdt,5"`
}

type ImageAssetsResult struct {
	Images     []ImageAssetSearchRecord   `json:"images" cb:"images,1"`
	Categories []ImageAssetCategoryRecord `json:"categories" cb:"categories,2"`
}

func GetImageAssets(req *core.HandlerArgs) core.HandlerResponse {
	// Each table has its own delta log, so the two response keys carry independent cursors; the
	// frontend sends one query param per key, named after it.
	result := ImageAssetsResult{
		Images:     []ImageAssetSearchRecord{},
		Categories: []ImageAssetCategoryRecord{},
	}

	queryGroup := errgroup.Group{}
	queryGroup.Go(func() error {
		storedAssets := []types.ImageAsset{}
		query := db.Query(&storedAssets)
		// Keyless delta index: image assets have no status to filter.
		query.Select(query.ID, query.CategoryID, query.Bigrams, query.Updated, query.Version).
			GroupID.Equals(types.ImageAssetCategoryGroupID).
			Delta(req.GetDeltaCursor("images"))
		if err := query.Exec(); err != nil {
			return err
		}

		result.Images = make([]ImageAssetSearchRecord, len(storedAssets))
		for recordIndex, storedAsset := range storedAssets {
			result.Images[recordIndex] = ImageAssetSearchRecord{
				ID:         storedAsset.ID,
				CategoryID: storedAsset.CategoryID,
				Bigrams:    encodeImageAssetBigrams(storedAsset.Bigrams),
				Updated:    storedAsset.Updated,
				Version:    storedAsset.Version,
				Metadata:   storedAsset.Metadata,
			}
		}
		return nil
	})
	queryGroup.Go(func() error {
		storedCategories := []types.ImageAssetCategory{}
		query := db.Query(&storedCategories)
		query.Select(query.ID, query.Name, query.Updated, query.Version).
			GroupID.Equals(types.ImageAssetCategoryGroupID).
			Delta(req.GetDeltaCursor("categories"))
		if err := query.Exec(); err != nil {
			return err
		}

		result.Categories = make([]ImageAssetCategoryRecord, len(storedCategories))
		for recordIndex, storedCategory := range storedCategories {
			result.Categories[recordIndex] = ImageAssetCategoryRecord{
				ID:       storedCategory.ID,
				Name:     storedCategory.Name,
				Updated:  storedCategory.Updated,
				Version:  storedCategory.Version,
				Metadata: storedCategory.Metadata,
			}
		}
		return nil
	})

	if err := queryGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los recursos de imágenes.", err)
	}
	return core.MakeResponse(req, &result)
}

// GetImageAssetTextSearch runs the Sonic GenixSearch index over image asset Keywords
// (English) and returns the top ids + relevance weights ordered by score.
func GetImageAssetTextSearch(req *core.HandlerArgs) core.HandlerResponse {
	query := req.GetQuery("q")
	if len(query) < 2 {
		return req.MakeErr("La búsqueda debe tener al menos 2 caracteres.")
	}
	limit := int(req.GetQueryInt("limit"))
	if limit <= 0 {
		limit = 10
	}

	// Image assets share the single group partition.
	matches, err := db.SearchTextIDs[types.ImageAsset](types.ImageAssetCategoryGroupID, query, limit, 0)
	if err != nil {
		return req.MakeErr("Error en la búsqueda de imágenes:", err)
	}
	return core.MakeResponse(req, &matches)
}

func encodeImageAssetBigrams(bigrams []int8) string {
	// Preserve each stored signed byte's original 8-bit representation.
	bigramBytes := make([]byte, len(bigrams))
	for bigramIndex, bigram := range bigrams {
		bigramBytes[bigramIndex] = byte(bigram)
	}
	return base64.StdEncoding.EncodeToString(bigramBytes)
}
