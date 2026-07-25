package business

import (
	businessTypes "app/business/types"
	"app/db"
	"net/url"
	"strconv"
	"strings"
)

// Public base for repository images, grouped by category slug. Mirrors the
// frontend resolver (ImageAssetsService.getImageURL in
// frontend/services/business/image-assets.svelte.ts).
const imageAssetsPublicBaseURL = "https://ivanjoz.github.io/genix-assets/images/"

// AgentImageCandidate is one image option offered to the page-builder agent's
// find_image subagent: enough to pick by meaning and aspect ratio, plus the
// ready-to-use public src URL the section HTML will embed.
type AgentImageCandidate struct {
	ID          int32   `json:"ID"`
	URL         string  `json:"url"`
	Description string  `json:"description,omitempty"`
	Ratio       float32 `json:"ratio,omitempty"` // width/height; 0 ⇒ unknown (treat as 1:1)
}

// FindImageCandidates returns image options for the page-builder agent. It runs
// the English-keyword text search first and falls back to the first images in
// the group when the search is empty or the query is too short — so the caller
// ALWAYS gets candidates (the builder agent must place some image). Each result
// carries a public URL resolved from its category slug.
func FindImageCandidates(keywords string, limit int) ([]AgentImageCandidate, error) {
	if limit <= 0 {
		limit = 10
	}
	assets := []businessTypes.ImageAsset{}
	keywords = strings.TrimSpace(keywords)
	if len(keywords) >= 2 {
		// Image assets share one group partition and have no Status column, so
		// they index into status group 0. SearchText hydrates `assets` ordered
		// by relevance (best match first).
		if _, err := db.SearchText[businessTypes.ImageAsset](&assets, imageAssetCategoryGroupID, keywords, 0, limit); err != nil {
			return nil, err
		}
	}
	if len(assets) == 0 {
		// Fallback: return any images so find_image never comes back empty.
		query := db.Query(&assets)
		if err := query.GroupID.Equals(imageAssetCategoryGroupID).Limit(int32(limit)).Exec(); err != nil {
			return nil, err
		}
	}

	categoryNames, err := imageCategoryNames()
	if err != nil {
		return nil, err
	}

	candidates := make([]AgentImageCandidate, 0, len(assets))
	for _, asset := range assets {
		candidates = append(candidates, AgentImageCandidate{
			ID:          asset.ID,
			URL:         imageAssetURL(categoryNames[asset.CategoryID], asset.ID),
			Description: asset.Description,
			Ratio:       asset.Ratio,
		})
	}
	return candidates, nil
}

// imageCategoryNames loads the CategoryID→Name map used to build public URLs.
func imageCategoryNames() (map[int16]string, error) {
	categories := []businessTypes.ImageAssetCategory{}
	query := db.Query(&categories)
	if err := query.Select(query.ID, query.Name).GroupID.Equals(imageAssetCategoryGroupID).Exec(); err != nil {
		return nil, err
	}
	names := make(map[int16]string, len(categories))
	for _, category := range categories {
		names[category.ID] = category.Name
	}
	return names, nil
}

// imageAssetURL builds the public src for an image from its category slug + id.
// Empty when the category is unknown (caller should skip such candidates).
func imageAssetURL(categoryName string, id int32) string {
	if categoryName == "" {
		return ""
	}
	return imageAssetsPublicBaseURL + url.PathEscape(categoryName) + "/" + strconv.Itoa(int(id)) + ".avif"
}
