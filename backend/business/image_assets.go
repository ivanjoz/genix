package business

import (
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"encoding/base64"

	"golang.org/x/sync/errgroup"
)

// ImageAssetSearchRecord is the minimal delta payload consumed by frontend search. UpdatedVersion
// travels with it because the client watermarks this route on "upv", not on the timestamp.
type ImageAssetSearchRecord struct {
	ID             int32  `json:",omitempty"`
	CategoryID     int16  `json:",omitempty"`
	Bigrams        string `json:",omitempty"`
	Updated        int32  `json:"upd,omitempty"`
	UpdatedVersion int32  `json:"upv,omitempty"`
}

type ImageAssetCategoryRecord struct {
	ID             int16  `json:",omitempty"`
	Name           string `json:",omitempty"`
	Updated        int32  `json:"upd,omitempty"`
	UpdatedVersion int32  `json:"upv,omitempty"`
}

type ImageAssetsResult struct {
	Images     []ImageAssetSearchRecord   `json:"images"`
	Categories []ImageAssetCategoryRecord `json:"categories"`
}

func GetImageAssets(req *core.HandlerArgs) core.HandlerResponse {
	// Each table advances its own "updated_version" sequence, so the two response keys carry
	// independent watermarks; the frontend sends one query param per key, named after it.
	imagesUpdatedSince := req.GetQueryInt("images")
	categoriesUpdatedSince := req.GetQueryInt("categories")
	result := ImageAssetsResult{
		Images:     []ImageAssetSearchRecord{},
		Categories: []ImageAssetCategoryRecord{},
	}

	core.Log("[image-assets] delta query started; images=", imagesUpdatedSince, " categories=", categoriesUpdatedSince)
	queryGroup := errgroup.Group{}
	queryGroup.Go(func() error {
		storedAssets := []businessTypes.ImageAsset{}
		query := db.Query(&storedAssets)
		// Image assets have no status, so Delta() constrains nothing but the watermark.
		query.Select(query.ID, query.CategoryID, query.Bigrams, query.Updated, query.UpdatedVersion).
			GroupID.Equals(imageAssetCategoryGroupID).
			Delta(imagesUpdatedSince)
		if err := query.Exec(); err != nil {
			return err
		}

		result.Images = make([]ImageAssetSearchRecord, len(storedAssets))
		for recordIndex, storedAsset := range storedAssets {
			result.Images[recordIndex] = ImageAssetSearchRecord{
				ID:             storedAsset.ID,
				CategoryID:     storedAsset.CategoryID,
				Bigrams:        encodeImageAssetBigrams(storedAsset.Bigrams),
				Updated:        storedAsset.Updated,
				UpdatedVersion: storedAsset.UpdatedVersion,
			}
		}
		return nil
	})
	queryGroup.Go(func() error {
		storedCategories := []businessTypes.ImageAssetCategory{}
		query := db.Query(&storedCategories)
		query.Select(query.ID, query.Name, query.Updated, query.UpdatedVersion).
			GroupID.Equals(imageAssetCategoryGroupID).
			Delta(categoriesUpdatedSince)
		if err := query.Exec(); err != nil {
			return err
		}

		result.Categories = make([]ImageAssetCategoryRecord, len(storedCategories))
		for recordIndex, storedCategory := range storedCategories {
			result.Categories[recordIndex] = ImageAssetCategoryRecord{
				ID:             storedCategory.ID,
				Name:           storedCategory.Name,
				Updated:        storedCategory.Updated,
				UpdatedVersion: storedCategory.UpdatedVersion,
			}
		}
		return nil
	})

	if err := queryGroup.Wait(); err != nil {
		core.Log("[image-assets] delta query failed; images=", imagesUpdatedSince, " categories=", categoriesUpdatedSince, " error=", err)
		return req.MakeErr("Error al obtener los recursos de imágenes.", err)
	}
	core.Log("[image-assets] delta query completed; images=", len(result.Images), " categories=", len(result.Categories))
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

	// Image assets share the single group partition and carry no Status column,
	// so they index into status group 0.
	matches, err := db.SearchTextIDs[businessTypes.ImageAsset](imageAssetCategoryGroupID, query, 0, limit)
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
