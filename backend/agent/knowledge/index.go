package knowledge

import (
	"context"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

// ExistingPoint contains only the payload fields needed for incremental comparison.
type ExistingPoint struct {
	ID                    string
	ContentHash           string
	DocumentationFileHash string
	DocumentationHash     string
	SourceHashDigest      string
	IndexContractHash     string
	IndexState            string
	DocumentChunkCount    int
}

// IndexPoint is a complete changed chunk ready for dense and lexical upsert.
type IndexPoint struct {
	ID          string
	DenseVector []float32
	LexicalText string
	Payload     map[string]any
}

// ListDocumentPoints scrolls payload only; stored vectors never cross the network for comparison.
func (store *Store) ListDocumentPoints(ctx context.Context, documentationPath string) ([]ExistingPoint, error) {
	request := &qdrant.ScrollPoints{
		CollectionName: store.config.Collection,
		Filter: &qdrant.Filter{Must: []*qdrant.Condition{
			qdrant.NewMatchKeyword("documentation_path", documentationPath),
		}},
		Limit:       qdrant.PtrOf(uint32(256)),
		WithPayload: qdrant.NewWithPayloadInclude("content_hash", "documentation_file_hash", "documentation_hash", "source_hash_digest", "index_contract_hash", "index_state", "document_chunk_count"),
		WithVectors: qdrant.NewWithVectors(false),
	}

	existingPoints := []ExistingPoint{}
	for {
		points, nextOffset, err := store.client.ScrollAndOffset(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("list Qdrant points for %s: %w", documentationPath, err)
		}
		for _, point := range points {
			payload := point.GetPayload()
			existingPoints = append(existingPoints, ExistingPoint{
				ID:                    point.GetId().GetUuid(),
				ContentHash:           payloadString(payload, "content_hash"),
				DocumentationFileHash: payloadString(payload, "documentation_file_hash"),
				DocumentationHash:     payloadString(payload, "documentation_hash"),
				SourceHashDigest:      payloadString(payload, "source_hash_digest"),
				IndexContractHash:     payloadString(payload, "index_contract_hash"),
				IndexState:            payloadString(payload, "index_state"),
				DocumentChunkCount:    payloadInteger(payload, "document_chunk_count"),
			})
		}
		if nextOffset == nil {
			break
		}
		request.Offset = nextOffset
	}
	log.Printf("[agent.knowledge] document_points_listed path=%s count=%d", documentationPath, len(existingPoints))
	return existingPoints, nil
}

// UpsertDocumentationPoints asks Qdrant 1.19+ to generate the lexical BM25 vector server-side.
func (store *Store) UpsertDocumentationPoints(ctx context.Context, points []IndexPoint) error {
	if len(points) == 0 {
		return nil
	}
	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, point := range points {
		payload, err := qdrant.TryValueMap(point.Payload)
		if err != nil {
			return fmt.Errorf("encode payload for point %s: %w", point.ID, err)
		}
		qdrantPoints = append(qdrantPoints, &qdrant.PointStruct{
			Id: qdrant.NewIDUUID(point.ID),
			Vectors: qdrant.NewVectorsMap(map[string]*qdrant.Vector{
				DenseVectorName: qdrant.NewVectorDense(point.DenseVector),
				LexicalVectorName: qdrant.NewVectorDocument(&qdrant.Document{
					Text:  point.LexicalText,
					Model: "qdrant/bm25",
					Options: qdrant.NewValueMap(map[string]any{
						"language":      "none",
						"tokenizer":     "multilingual",
						"ascii_folding": true,
					}),
				}),
			}),
			Payload: payload,
		})
	}
	if _, err := store.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: store.config.Collection,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrantPoints,
	}); err != nil {
		return fmt.Errorf("upsert %d documentation points: %w", len(points), err)
	}
	log.Printf("[agent.knowledge] documentation_points_upserted collection=%s count=%d", store.config.Collection, len(points))
	return nil
}

// ReplacePointPayload refreshes hashes/navigation metadata while preserving existing vectors.
func (store *Store) ReplacePointPayload(ctx context.Context, pointID string, payload map[string]any) error {
	encodedPayload, err := qdrant.TryValueMap(payload)
	if err != nil {
		return fmt.Errorf("encode payload for point %s: %w", pointID, err)
	}
	if _, err := store.client.OverwritePayload(ctx, &qdrant.SetPayloadPoints{
		CollectionName: store.config.Collection,
		Wait:           qdrant.PtrOf(true),
		Payload:        encodedPayload,
		PointsSelector: qdrant.NewPointsSelector(qdrant.NewIDUUID(pointID)),
	}); err != nil {
		return fmt.Errorf("replace payload for point %s: %w", pointID, err)
	}
	return nil
}

func (store *Store) DeleteDocumentationPoints(ctx context.Context, pointIDs []string) error {
	if len(pointIDs) == 0 {
		return nil
	}
	qdrantIDs := make([]*qdrant.PointId, 0, len(pointIDs))
	for _, pointID := range pointIDs {
		qdrantIDs = append(qdrantIDs, qdrant.NewIDUUID(pointID))
	}
	if _, err := store.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: store.config.Collection,
		Wait:           qdrant.PtrOf(true),
		Points:         qdrant.NewPointsSelector(qdrantIDs...),
	}); err != nil {
		return fmt.Errorf("delete %d obsolete documentation points: %w", len(pointIDs), err)
	}
	log.Printf("[agent.knowledge] documentation_points_deleted collection=%s count=%d", store.config.Collection, len(pointIDs))
	return nil
}

func payloadString(payload map[string]*qdrant.Value, key string) string {
	if value := payload[key]; value != nil {
		return value.GetStringValue()
	}
	return ""
}

func payloadInteger(payload map[string]*qdrant.Value, key string) int {
	if value := payload[key]; value != nil {
		return int(value.GetIntegerValue())
	}
	return 0
}
