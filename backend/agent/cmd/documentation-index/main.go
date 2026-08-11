package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"app/agent/embedding"
	"app/agent/knowledge"
	"app/agent/ragdocs"
	"app/core"
)

type parsedDocument struct {
	document *ragdocs.Document
	chunks   []ragdocs.Chunk
}

func main() {
	if err := run(); err != nil {
		log.Printf("[agent.ragdocs] index_failed error=%v", err)
		os.Exit(1)
	}
}

func run() error {
	mode := flag.String("mode", "validate", "validate or index")
	dryRun := flag.Bool("dry-run", false, "read current Qdrant state but perform no embedding or writes")
	repositoryRoot := flag.String("root", defaultRepositoryRoot(), "absolute or relative repository root")
	documentPath := flag.String("document", "", "optional repository-relative DOCUMENTATION.md path")
	batchSize := flag.Int("batch-size", 16, "maximum OpenRouter inputs and Qdrant points per batch")
	qdrantHost := flag.String("qdrant-host", "", "optional qdrant.host override without modifying config.toml")
	flag.Parse()

	absoluteRoot, err := filepath.Abs(*repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	parsedDocuments, err := parseRepositoryDocuments(absoluteRoot, *documentPath)
	if err != nil {
		return err
	}
	log.Printf("[agent.ragdocs] validation_complete documents=%d chunks=%d", len(parsedDocuments), totalChunks(parsedDocuments))
	if *mode == "validate" {
		return nil
	}
	if *mode != "index" {
		return fmt.Errorf("unsupported mode %q; use validate or index", *mode)
	}
	if *batchSize <= 0 {
		return errors.New("batch-size must be greater than zero")
	}

	core.PopulateVariables()
	if strings.TrimSpace(*qdrantHost) != "" {
		core.Env.QDRANT_HOST = strings.TrimSpace(*qdrantHost)
	}
	qdrantConfig, err := knowledge.ConfigFromEnv()
	if err != nil {
		return err
	}
	qdrantStore, err := knowledge.NewStore(qdrantConfig)
	if err != nil {
		return err
	}
	defer qdrantStore.Close()

	collectionExists, err := qdrantStore.CollectionExists(context.Background())
	if err != nil {
		return err
	}
	var pointStore ragdocs.PointStore
	if *dryRun {
		if collectionExists {
			if err := qdrantStore.ValidateExistingCollection(context.Background()); err != nil {
				return err
			}
			pointStore = qdrantStore
		} else {
			log.Printf("[agent.ragdocs] dry_run_collection_missing collection=%s action=would_create", qdrantConfig.Collection)
		}
	} else {
		if err := qdrantStore.EnsureCollection(context.Background()); err != nil {
			return err
		}
		pointStore = qdrantStore
	}

	var documentEmbedder ragdocs.DocumentEmbedder
	if !*dryRun {
		documentEmbedder, err = embedding.NewClientFromEnv()
		if err != nil {
			return err
		}
	}

	totalIndexedChunks := 0
	totalEmbeddedChunks := 0
	totalPayloadUpdates := 0
	totalDeletedChunks := 0
	skippedDocuments := 0
	for _, parsed := range parsedDocuments {
		result, err := ragdocs.IndexDocument(context.Background(), pointStore, documentEmbedder, parsed.document, parsed.chunks, ragdocs.IndexOptions{
			DryRun:     *dryRun,
			BatchSize:  *batchSize,
			IndexedAt:  time.Now().UTC(),
			Model:      core.Env.EMBEDDING_MODEL_ID,
			Dimensions: core.Env.EMBEDDING_DIMENSIONS,
		})
		if err != nil {
			return err
		}
		totalIndexedChunks += result.Chunks
		totalEmbeddedChunks += result.Embedded
		totalPayloadUpdates += result.PayloadUpdated
		totalDeletedChunks += result.Deleted
		if result.Skipped {
			skippedDocuments++
		}
	}
	log.Printf("[agent.ragdocs] index_complete documents=%d skipped_documents=%d chunks=%d embedded=%d payload_updated=%d deleted=%d dry_run=%t",
		len(parsedDocuments), skippedDocuments, totalIndexedChunks, totalEmbeddedChunks, totalPayloadUpdates, totalDeletedChunks, *dryRun)
	return nil
}

func parseRepositoryDocuments(repositoryRoot, selectedPath string) ([]parsedDocument, error) {
	paths, err := ragdocs.Discover(repositoryRoot)
	if err != nil {
		return nil, err
	}
	if selectedPath != "" {
		selectedAbsolutePath := filepath.Join(repositoryRoot, filepath.FromSlash(selectedPath))
		paths = []string{selectedAbsolutePath}
	}
	if len(paths) == 0 {
		return nil, errors.New("no frontend/routes/**/DOCUMENTATION.md files found")
	}

	parsedDocuments := make([]parsedDocument, 0, len(paths))
	seenPageIDs := map[string]string{}
	seenPointIDs := map[string]string{}
	for _, documentationPath := range paths {
		document, err := ragdocs.ParseFile(repositoryRoot, documentationPath)
		if err != nil {
			return nil, err
		}
		if previousPath := seenPageIDs[document.Frontmatter.PageID]; previousPath != "" {
			return nil, fmt.Errorf("duplicate page_id %q in %s and %s", document.Frontmatter.PageID, previousPath, document.RepositoryPath)
		}
		seenPageIDs[document.Frontmatter.PageID] = document.RepositoryPath
		chunks, err := ragdocs.BuildChunks(document)
		if err != nil {
			return nil, err
		}
		for _, chunk := range chunks {
			if previousKey := seenPointIDs[chunk.PointID]; previousKey != "" {
				return nil, fmt.Errorf("duplicate point ID %s for %s and %s", chunk.PointID, previousKey, chunk.PointKey)
			}
			seenPointIDs[chunk.PointID] = chunk.PointKey
		}
		parsedDocuments = append(parsedDocuments, parsedDocument{document: document, chunks: chunks})
		log.Printf("[agent.ragdocs] document_valid path=%s page_id=%s route=%s sections=%d chunks=%d file_hash=%s",
			document.RepositoryPath, document.Frontmatter.PageID, document.Frontmatter.Route, len(document.Sections), len(chunks), document.FileHash)
	}
	return parsedDocuments, nil
}

func totalChunks(documents []parsedDocument) int {
	total := 0
	for _, document := range documents {
		total += len(document.chunks)
	}
	return total
}

func defaultRepositoryRoot() string {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "."
	}
	if directoryExists(filepath.Join(workingDirectory, "frontend", "routes")) {
		return workingDirectory
	}
	parent := filepath.Dir(workingDirectory)
	if directoryExists(filepath.Join(parent, "frontend", "routes")) {
		return parent
	}
	return workingDirectory
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
