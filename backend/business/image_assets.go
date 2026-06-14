package business

import (
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"encoding/base64"

	"golang.org/x/sync/errgroup"
)

// ImageAssetSearchRecord is the minimal delta payload consumed by frontend search.
type ImageAssetSearchRecord struct {
	ID         int32  `json:",omitempty"`
	CategoryID int16  `json:",omitempty"`
	Bigrams    string `json:",omitempty"`
	Updated    int32  `json:"upd,omitempty"`
}

type ImageAssetCategoryRecord struct {
	ID      int16  `json:",omitempty"`
	Name    string `json:",omitempty"`
	Updated int32  `json:"upd,omitempty"`
}

type ImageAssetsResult struct {
	Images     []ImageAssetSearchRecord   `json:"images"`
	Categories []ImageAssetCategoryRecord `json:"categories"`
}

func GetImageAssets(req *core.HandlerArgs) core.HandlerResponse {
	imagesUpdated := req.GetQueryInt("images")
	categoriesUpdated := req.GetQueryInt("categories")
	result := ImageAssetsResult{
		Images:     []ImageAssetSearchRecord{},
		Categories: []ImageAssetCategoryRecord{},
	}

	core.Log("[image-assets] delta query started; images=", imagesUpdated, " categories=", categoriesUpdated)
	queryGroup := errgroup.Group{}
	queryGroup.Go(func() error {
		storedAssets := []businessTypes.ImageAsset{}
		query := db.Query(&storedAssets)
		query.Select(query.ID, query.CategoryID, query.Bigrams, query.Updated)
		if imagesUpdated > 0 {
			query.Updated.GreaterThan(imagesUpdated)
		}
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
			}
		}
		return nil
	})
	queryGroup.Go(func() error {
		storedCategories := []businessTypes.ImageAssetCategory{}
		query := db.Query(&storedCategories)
		query.Select(query.ID, query.Name, query.Updated).
			GroupID.Equals(imageAssetCategoryGroupID)
		if categoriesUpdated > 0 {
			query.Updated.GreaterThan(categoriesUpdated)
		}
		if err := query.Exec(); err != nil {
			return err
		}

		result.Categories = make([]ImageAssetCategoryRecord, len(storedCategories))
		for recordIndex, storedCategory := range storedCategories {
			result.Categories[recordIndex] = ImageAssetCategoryRecord{
				ID:      storedCategory.ID,
				Name:    storedCategory.Name,
				Updated: storedCategory.Updated,
			}
		}
		return nil
	})

	if err := queryGroup.Wait(); err != nil {
		core.Log("[image-assets] delta query failed; images=", imagesUpdated, " categories=", categoriesUpdated, " error=", err)
		return req.MakeErr("Error al obtener los recursos de imágenes.", err)
	}
	core.Log("[image-assets] delta query completed; images=", len(result.Images), " categories=", len(result.Categories))
	return core.MakeResponse(req, &result)
}

func encodeImageAssetBigrams(bigrams []int8) string {
	// Preserve each stored signed byte's original 8-bit representation.
	bigramBytes := make([]byte, len(bigrams))
	for bigramIndex, bigram := range bigrams {
		bigramBytes[bigramIndex] = byte(bigram)
	}
	return base64.StdEncoding.EncodeToString(bigramBytes)
}
