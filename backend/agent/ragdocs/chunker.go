package ragdocs

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	preferredChunkTokens = 600
	hardMaximumTokens    = 900
)

var backtickTermPattern = regexp.MustCompile("`([^`\\n]+)`")

// BuildChunks keeps DOC-ID boundaries stable and splits only sections above the hard limit.
func BuildChunks(document *Document) ([]Chunk, error) {
	chunks := []Chunk{}
	for _, section := range document.Sections {
		parts := splitSection(section.Markdown)
		for partIndex, content := range parts {
			contextHeader := fmt.Sprintf("Page: %s\nRoute: %s\nSection: %s",
				document.Frontmatter.Title, document.Frontmatter.Route, section.Title)
			embeddingText := contextHeader + "\n\n" + content
			pointKey := fmt.Sprintf("%s:%s:%d", document.Frontmatter.PageID, section.ID, partIndex)
			chunks = append(chunks, Chunk{
				PointID:       StablePointID(CollectionSchemaVersion, document.Frontmatter.PageID, section.ID, partIndex),
				PointKey:      pointKey,
				SectionID:     section.ID,
				SectionTitle:  section.Title,
				SectionType:   section.Type,
				PartIndex:     partIndex,
				PartCount:     len(parts),
				Content:       content,
				ContextHeader: contextHeader,
				EmbeddingText: embeddingText,
				ContentHash:   hashBytes([]byte(embeddingText)),
				Keywords:      extractKeywords(content),
			})
		}
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("document %s produced no chunks", document.RepositoryPath)
	}
	return chunks, nil
}

// CollectionSchemaVersion participates in every point ID; changing parsing semantics creates new IDs.
const CollectionSchemaVersion = 1

func splitSection(markdown string) []string {
	if estimateTokens(markdown) <= hardMaximumTokens {
		return []string{markdown}
	}

	paragraphs := strings.Split(markdown, "\n\n")
	parts := []string{}
	currentParagraphs := []string{}
	currentTokens := 0
	flushCurrent := func() {
		if len(currentParagraphs) == 0 {
			return
		}
		parts = append(parts, strings.Join(currentParagraphs, "\n\n"))
		currentParagraphs = nil
		currentTokens = 0
	}

	for _, paragraph := range paragraphs {
		paragraphTokens := estimateTokens(paragraph)
		if paragraphTokens > hardMaximumTokens {
			flushCurrent()
			parts = append(parts, splitLongParagraph(paragraph)...)
			continue
		}
		if currentTokens > 0 && currentTokens+paragraphTokens > preferredChunkTokens {
			flushCurrent()
		}
		currentParagraphs = append(currentParagraphs, paragraph)
		currentTokens += paragraphTokens
	}
	flushCurrent()
	return parts
}

func splitLongParagraph(paragraph string) []string {
	words := strings.Fields(paragraph)
	parts := []string{}
	currentWords := []string{}
	for _, word := range words {
		candidate := strings.Join(append(currentWords, word), " ")
		if len(currentWords) > 0 && estimateTokens(candidate) > preferredChunkTokens {
			parts = append(parts, strings.Join(currentWords, " "))
			currentWords = nil
		}
		currentWords = append(currentWords, word)
	}
	if len(currentWords) > 0 {
		parts = append(parts, strings.Join(currentWords, " "))
	}
	return parts
}

// Qwen tokenization is provider-side; rune/4 is a conservative deterministic planning estimate.
func estimateTokens(text string) int {
	runeCount := utf8.RuneCountInString(text)
	return (runeCount + 3) / 4
}

func extractKeywords(content string) []string {
	matches := backtickTermPattern.FindAllStringSubmatch(content, -1)
	keywords := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		keyword := strings.TrimSpace(match[1])
		if keyword == "" || seen[keyword] {
			continue
		}
		seen[keyword] = true
		keywords = append(keywords, keyword)
		if len(keywords) == 32 {
			break
		}
	}
	return keywords
}

// StablePointID is UUID-shaped so Qdrant can address a chunk directly without a lookup.
func StablePointID(schemaVersion int, pageID, sectionID string, partIndex int) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%d", schemaVersion, pageID, sectionID, partIndex)))
	uuidBytes := digest[:16]
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x50
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])
}
