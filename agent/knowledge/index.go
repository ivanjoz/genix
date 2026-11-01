package knowledge

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// ExistingPoint contains only the payload fields needed for incremental comparison.
type ExistingPoint struct {
	ID                    string `json:"-"`
	ContentHash         string `json:"content_hash"`
	DocumentationFileHash string `json:"documentation_file_hash"`
	DocumentationHash     string `json:"documentation_hash"`
	SourceHashDigest      string `json:"source_hash_digest"`
	IndexContractHash     string `json:"index_contract_hash"`
	IndexState            string `json:"index_state"`
	DocumentChunkCount    int    `json:"document_chunk_count"`
}

// IndexPoint is a complete changed chunk ready for dense and lexical upsert.
type IndexPoint struct {
	ID          string
	DenseVector []float32
	LexicalText string
	Payload     map[string]any
}

type scrollPointsRequest struct {
	Filter      pointFilter `json:"filter"`
	Limit       int         `json:"limit"`
	Offset      *string     `json:"offset,omitempty"`
	WithPayload []string    `json:"with_payload"`
	WithVector  bool        `json:"with_vector"`
}

type upsertPoint struct {
	ID      string         `json:"id"`
	Vector  map[string]any `json:"vector"`
	Payload map[string]any `json:"payload"`
}

// ListDocumentPoints scrolls payload only; stored vectors never cross the network for comparison.
func (store *Store) ListDocumentPoints(ctx context.Context, documentationPath string) ([]ExistingPoint, error) {
	request := scrollPointsRequest{
		Filter: pointFilter{Must: []fieldCondition{
			{Key: "documentation_path", Match: matchCondition{Value: documentationPath}},
		}},
		Limit:       256,
		WithPayload: []string{"content_hash", "documentation_file_hash", "documentation_hash", "source_hash_digest", "index_contract_hash", "index_state", "document_chunk_count"},
	}

	existingPoints := []ExistingPoint{}
	for {
		// Point IDs are always UUID strings written by this indexer, so the page offset is too.
		scrollResult := struct {
			Points []struct {
				ID      string        `json:"id"`
				Payload ExistingPoint `json:"payload"`
			} `json:"points"`
			NextPageOffset *string `json:"next_page_offset"`
		}{}
		if err := store.callQdrant(ctx, http.MethodPost, "/points/scroll", request, &scrollResult); err != nil {
			return nil, fmt.Errorf("list Qdrant points for %s: %w", documentationPath, err)
		}
		for _, point := range scrollResult.Points {
			existingPoint := point.Payload
			existingPoint.ID = point.ID
			existingPoints = append(existingPoints, existingPoint)
		}
		if scrollResult.NextPageOffset == nil {
			break
		}
		request.Offset = scrollResult.NextPageOffset
	}
	log.Printf("[agent.knowledge] document_points_listed path=%s count=%d", documentationPath, len(existingPoints))
	return existingPoints, nil
}

// UpsertDocumentationPoints asks Qdrant 1.19+ to generate the lexical BM25 vector server-side.
func (store *Store) UpsertDocumentationPoints(ctx context.Context, points []IndexPoint) error {
	if len(points) == 0 {
		return nil
	}
	upsertPoints := make([]upsertPoint, 0, len(points))
	for _, point := range points {
		upsertPoints = append(upsertPoints, upsertPoint{
			ID: point.ID,
			Vector: map[string]any{
				DenseVectorName:   point.DenseVector,
				LexicalVectorName: bm25Document{Text: point.LexicalText, Model: bm25Model, Options: bm25DocumentOptions},
			},
			Payload: point.Payload,
		})
	}
	upsertRequest := struct {
		Points []upsertPoint `json:"points"`
	}{Points: upsertPoints}
	if err := store.callQdrant(ctx, http.MethodPut, "/points?wait=true", upsertRequest, nil); err != nil {
		return fmt.Errorf("upsert %d documentation points: %w", len(points), err)
	}
	log.Printf("[agent.knowledge] documentation_points_upserted collection=%s count=%d", store.config.Collection, len(points))
	return nil
}

// ReplacePointPayload refreshes hashes/navigation metadata while preserving existing vectors.
func (store *Store) ReplacePointPayload(ctx context.Context, pointID string, payload map[string]any) error {
	overwriteRequest := struct {
		Payload map[string]any `json:"payload"`
		Points  []string       `json:"points"`
	}{Payload: payload, Points: []string{pointID}}
	// PUT /points/payload overwrites the whole payload; POST would merge into the old one.
	if err := store.callQdrant(ctx, http.MethodPut, "/points/payload?wait=true", overwriteRequest, nil); err != nil {
		return fmt.Errorf("replace payload for point %s: %w", pointID, err)
	}
	return nil
}

func (store *Store) DeleteDocumentationPoints(ctx context.Context, pointIDs []string) error {
	if len(pointIDs) == 0 {
		return nil
	}
	deleteRequest := struct {
		Points []string `json:"points"`
	}{Points: pointIDs}
	if err := store.callQdrant(ctx, http.MethodPost, "/points/delete?wait=true", deleteRequest, nil); err != nil {
		return fmt.Errorf("delete %d obsolete documentation points: %w", len(pointIDs), err)
	}
	log.Printf("[agent.knowledge] documentation_points_deleted collection=%s count=%d", store.config.Collection, len(pointIDs))
	return nil
}
