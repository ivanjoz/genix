package ragdocs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"app/agent/knowledge"
)

type DocumentEmbedder interface {
	EmbedDocuments(context.Context, []string) ([][]float32, error)
}

type PointStore interface {
	ListDocumentPoints(context.Context, string) ([]knowledge.ExistingPoint, error)
	UpsertDocumentationPoints(context.Context, []knowledge.IndexPoint) error
	ReplacePointPayload(context.Context, string, map[string]any) error
	DeleteDocumentationPoints(context.Context, []string) error
}

type IndexOptions struct {
	DryRun     bool
	BatchSize  int
	IndexedAt  time.Time
	Model      string
	Dimensions int
}

type IndexResult struct {
	DocumentationPath string
	Skipped           bool
	Chunks            int
	Embedded          int
	PayloadUpdated    int
	Deleted           int
}

// IndexContractHash invalidates reuse when a setting that changes vectors or IDs changes.
func IndexContractHash(model string, dimensions int) string {
	contract := fmt.Sprintf(
		"collection_schema=%d\nparser_schema=%d\nchunk_target=%d\nchunk_hard_max=%d\nmodel=%s\ndimensions=%d\nbm25=qdrant/bm25\nlanguage=none\ntokenizer=multilingual\nascii_folding=true",
		CollectionSchemaVersion, CollectionSchemaVersion, preferredChunkTokens, hardMaximumTokens,
		strings.TrimSpace(model), dimensions,
	)
	return hashBytes([]byte(contract))
}

// IndexDocument performs an idempotent per-document update and writes its completion marker last.
func IndexDocument(ctx context.Context, store PointStore, embedder DocumentEmbedder, document *Document, chunks []Chunk, options IndexOptions) (IndexResult, error) {
	result := IndexResult{DocumentationPath: document.RepositoryPath, Chunks: len(chunks)}
	if options.BatchSize <= 0 {
		options.BatchSize = 16
	}
	if options.IndexedAt.IsZero() {
		options.IndexedAt = time.Now().UTC()
	}
	if options.Model == "" || options.Dimensions <= 0 {
		return result, errors.New("embedding model and dimensions are required for indexing")
	}
	if len(chunks) == 0 {
		return result, errors.New("document has no chunks")
	}

	markerIndex := -1
	for chunkIndex, chunk := range chunks {
		if chunk.SectionID == "page-purpose" && chunk.PartIndex == 0 {
			markerIndex = chunkIndex
			break
		}
	}
	if markerIndex < 0 {
		return result, errors.New("document has no page-purpose completion marker chunk")
	}

	existingPoints := []knowledge.ExistingPoint{}
	var err error
	if store != nil {
		existingPoints, err = store.ListDocumentPoints(ctx, document.RepositoryPath)
		if err != nil {
			return result, err
		}
	}
	existingByID := make(map[string]knowledge.ExistingPoint, len(existingPoints))
	for _, existingPoint := range existingPoints {
		existingByID[existingPoint.ID] = existingPoint
	}

	contractHash := IndexContractHash(options.Model, options.Dimensions)
	markerID := chunks[markerIndex].PointID
	if marker, exists := existingByID[markerID]; exists &&
		marker.IndexState == "complete" &&
		marker.DocumentChunkCount == len(chunks) &&
		len(existingPoints) == len(chunks) &&
		marker.DocumentationFileHash == document.FileHash &&
		marker.SourceHashDigest == document.SourceHashDigest &&
		marker.IndexContractHash == contractHash {
		result.Skipped = true
		log.Printf("[agent.ragdocs] document_skipped path=%s chunks=%d file_hash=%s", document.RepositoryPath, len(chunks), document.FileHash)
		return result, nil
	}

	newIDs := make(map[string]bool, len(chunks))
	changedChunkIndexes := []int{}
	unchangedChunkIndexes := []int{}
	for chunkIndex, chunk := range chunks {
		newIDs[chunk.PointID] = true
		existing, exists := existingByID[chunk.PointID]
		if exists && existing.ContentHash == chunk.ContentHash && existing.IndexContractHash == contractHash {
			unchangedChunkIndexes = append(unchangedChunkIndexes, chunkIndex)
		} else {
			changedChunkIndexes = append(changedChunkIndexes, chunkIndex)
		}
	}
	obsoleteIDs := []string{}
	for _, existing := range existingPoints {
		if !newIDs[existing.ID] {
			obsoleteIDs = append(obsoleteIDs, existing.ID)
		}
	}
	result.Embedded = len(changedChunkIndexes)
	result.PayloadUpdated = len(unchangedChunkIndexes)
	result.Deleted = len(obsoleteIDs)
	if options.DryRun {
		log.Printf("[agent.ragdocs] document_dry_run path=%s chunks=%d embed=%d payload=%d delete=%d",
			document.RepositoryPath, result.Chunks, result.Embedded, result.PayloadUpdated, result.Deleted)
		return result, nil
	}
	if store == nil || embedder == nil {
		return result, errors.New("Qdrant store and embedder are required outside dry-run")
	}

	vectorsByChunkIndex := map[int][]float32{}
	for batchStart := 0; batchStart < len(changedChunkIndexes); batchStart += options.BatchSize {
		batchEnd := min(batchStart+options.BatchSize, len(changedChunkIndexes))
		batchIndexes := changedChunkIndexes[batchStart:batchEnd]
		batchTexts := make([]string, 0, len(batchIndexes))
		for _, chunkIndex := range batchIndexes {
			batchTexts = append(batchTexts, chunks[chunkIndex].EmbeddingText)
		}
		vectors, err := embedder.EmbedDocuments(ctx, batchTexts)
		if err != nil {
			return result, fmt.Errorf("embed %s batch %d-%d: %w", document.RepositoryPath, batchStart, batchEnd, err)
		}
		for vectorIndex, vector := range vectors {
			vectorsByChunkIndex[batchIndexes[vectorIndex]] = vector
		}
	}

	// Every non-marker mutation completes first; the marker's final payload is the commit record.
	for batchStart := 0; batchStart < len(changedChunkIndexes); batchStart += options.BatchSize {
		batchEnd := min(batchStart+options.BatchSize, len(changedChunkIndexes))
		points := []knowledge.IndexPoint{}
		for _, chunkIndex := range changedChunkIndexes[batchStart:batchEnd] {
			if chunkIndex == markerIndex {
				continue
			}
			points = append(points, indexPoint(document, chunks, chunkIndex, vectorsByChunkIndex[chunkIndex], contractHash, options, false))
		}
		if err := store.UpsertDocumentationPoints(ctx, points); err != nil {
			return result, err
		}
	}
	for _, chunkIndex := range unchangedChunkIndexes {
		if chunkIndex == markerIndex {
			continue
		}
		if err := store.ReplacePointPayload(ctx, chunks[chunkIndex].PointID, chunkPayload(document, chunks, chunkIndex, contractHash, options, false)); err != nil {
			return result, err
		}
	}
	if err := store.DeleteDocumentationPoints(ctx, obsoleteIDs); err != nil {
		return result, err
	}

	markerChanged := false
	for _, chunkIndex := range changedChunkIndexes {
		if chunkIndex == markerIndex {
			markerChanged = true
			break
		}
	}
	if markerChanged {
		markerPoint := indexPoint(document, chunks, markerIndex, vectorsByChunkIndex[markerIndex], contractHash, options, true)
		if err := store.UpsertDocumentationPoints(ctx, []knowledge.IndexPoint{markerPoint}); err != nil {
			return result, err
		}
	} else if err := store.ReplacePointPayload(ctx, markerID, chunkPayload(document, chunks, markerIndex, contractHash, options, true)); err != nil {
		return result, err
	}

	log.Printf("[agent.ragdocs] document_indexed path=%s chunks=%d embedded=%d payload=%d deleted=%d file_hash=%s",
		document.RepositoryPath, result.Chunks, result.Embedded, result.PayloadUpdated, result.Deleted, document.FileHash)
	return result, nil
}

func indexPoint(document *Document, chunks []Chunk, chunkIndex int, denseVector []float32, contractHash string, options IndexOptions, complete bool) knowledge.IndexPoint {
	return knowledge.IndexPoint{
		ID:          chunks[chunkIndex].PointID,
		DenseVector: denseVector,
		LexicalText: chunks[chunkIndex].EmbeddingText,
		Payload:     chunkPayload(document, chunks, chunkIndex, contractHash, options, complete),
	}
}

func chunkPayload(document *Document, chunks []Chunk, chunkIndex int, contractHash string, options IndexOptions, complete bool) map[string]any {
	chunk := chunks[chunkIndex]
	previousPointID := ""
	nextPointID := ""
	if chunkIndex > 0 && chunks[chunkIndex-1].SectionID == chunk.SectionID {
		previousPointID = chunks[chunkIndex-1].PointID
	}
	if chunkIndex+1 < len(chunks) && chunks[chunkIndex+1].SectionID == chunk.SectionID {
		nextPointID = chunks[chunkIndex+1].PointID
	}

	sourceFiles := make([]any, 0, len(document.Evidence.Files))
	sourceHashes := make([]any, 0, len(document.Evidence.Files))
	for _, evidenceFile := range document.Evidence.Files {
		sourceFiles = append(sourceFiles, evidenceFile.Path)
		sourceHashes = append(sourceHashes, evidenceFile.Hash)
	}
	keywords := make([]any, 0, len(chunk.Keywords))
	for _, keyword := range chunk.Keywords {
		keywords = append(keywords, keyword)
	}

	indexState := "chunk"
	if complete {
		indexState = "complete"
	}
	return map[string]any{
		"point_type":              "documentation_chunk",
		"schema_version":          int64(CollectionSchemaVersion),
		"document_id":             document.Frontmatter.PageID,
		"section_id":              chunk.SectionID,
		"point_key":               chunk.PointKey,
		"part_index":              int64(chunk.PartIndex),
		"part_count":              int64(chunk.PartCount),
		"document_chunk_count":    int64(len(chunks)),
		"previous_point_id":       previousPointID,
		"next_point_id":           nextPointID,
		"route":                   document.Frontmatter.Route,
		"module":                  moduleName(document.Frontmatter.Route),
		"title":                   document.Frontmatter.Title,
		"section_title":           chunk.SectionTitle,
		"section_type":            chunk.SectionType,
		"content":                 chunk.Content,
		"context_header":          chunk.ContextHeader,
		"keywords":                keywords,
		"status":                  document.Frontmatter.Status,
		"visibility":              document.Frontmatter.Visibility,
		"documentation_path":      document.RepositoryPath,
		"documentation_file_hash": document.FileHash,
		"documentation_hash":      document.DocumentationHash,
		"content_hash":            chunk.ContentHash,
		"source_hash_digest":      document.SourceHashDigest,
		"source_files":            sourceFiles,
		"source_hashes":           sourceHashes,
		"embedding_model":         options.Model,
		"embedding_dimensions":    int64(options.Dimensions),
		"index_contract_hash":     contractHash,
		"index_state":             indexState,
		"documentation_current":   true,
		"indexed_at":              options.IndexedAt.Unix(),
	}
}

func moduleName(route string) string {
	firstSegment := strings.Split(strings.Trim(route, "/"), "/")[0]
	switch firstSegment {
	case "business":
		return "Business"
	case "sales":
		return "Sales"
	case "logistics":
		return "Logistics"
	case "finance":
		return "Finance"
	case "security":
		return "Security"
	case "configuration":
		return "Configuration"
	case "website":
		return "Website"
	default:
		return "System"
	}
}
