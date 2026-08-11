package ragdocs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	docIDPattern        = regexp.MustCompile(`(?m)^<!--\s*DOC-ID:\s*([a-zA-Z0-9._-]+)\s*-->\s*$`)
	sectionTitlePattern = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	htmlCommentPattern  = regexp.MustCompile(`(?s)<!--.*?-->`)
	yamlFencePattern    = regexp.MustCompile("(?s)^\\s*```yaml\\s*\\n(.*?)\\n```\\s*$")
)

var allowedEvidenceRoles = map[string]bool{
	"page": true, "user-interface": true, "frontend-service": true,
	"backend-handler": true, "business-logic": true, "data-model": true,
	"permissions": true, "shared-domain": true, "reference-document": true,
}

// Discover returns only canonical route documentation files in stable path order.
func Discover(repositoryRoot string) ([]string, error) {
	pattern := filepath.Join(repositoryRoot, "frontend", "routes", "**", "DOCUMENTATION.md")
	_ = pattern // filepath.Glob has no recursive ** support; WalkDir enforces the same scope below.

	documentationPaths := []string{}
	routesRoot := filepath.Join(repositoryRoot, "frontend", "routes")
	err := filepath.WalkDir(routesRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "DOCUMENTATION.md" {
			return nil
		}
		documentationPaths = append(documentationPaths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover route documentation: %w", err)
	}
	sort.Strings(documentationPaths)
	return documentationPaths, nil
}

// ParseFile validates identity, retrievable sections, and the exact reviewed source hashes.
func ParseFile(repositoryRoot, documentationPath string) (*Document, error) {
	rawMarkdown, err := os.ReadFile(documentationPath)
	if err != nil {
		return nil, fmt.Errorf("read documentation: %w", err)
	}
	repositoryPath, err := filepath.Rel(repositoryRoot, documentationPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repository path: %w", err)
	}
	repositoryPath = filepath.ToSlash(repositoryPath)
	if strings.HasPrefix(repositoryPath, "../") || filepath.IsAbs(repositoryPath) {
		return nil, fmt.Errorf("documentation path escapes repository: %s", repositoryPath)
	}

	normalizedLineEndings := strings.ReplaceAll(string(rawMarkdown), "\r\n", "\n")
	frontmatter, indexableMarkdown, filesMarkdown, err := splitDocumentParts(normalizedLineEndings)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", repositoryPath, err)
	}

	metadata := Frontmatter{}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, fmt.Errorf("%s: parse frontmatter: %w", repositoryPath, err)
	}
	if err := validateFrontmatter(repositoryPath, metadata); err != nil {
		return nil, err
	}

	evidence := EvidenceManifest{}
	fenceMatch := yamlFencePattern.FindStringSubmatch(filesMarkdown)
	if len(fenceMatch) != 2 {
		return nil, fmt.Errorf("%s: FILES must contain one fenced yaml manifest", repositoryPath)
	}
	if err := yaml.Unmarshal([]byte(fenceMatch[1]), &evidence); err != nil {
		return nil, fmt.Errorf("%s: parse FILES manifest: %w", repositoryPath, err)
	}

	sections, err := parseSections(indexableMarkdown)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", repositoryPath, err)
	}
	if err := validateEvidence(repositoryRoot, repositoryPath, evidence, sections); err != nil {
		return nil, err
	}

	documentationHashInput := canonicalDocumentation(metadata, sections)
	return &Document{
		RepositoryPath:    repositoryPath,
		Frontmatter:       metadata,
		Evidence:          evidence,
		Sections:          sections,
		FileHash:          hashBytes(rawMarkdown),
		DocumentationHash: hashBytes([]byte(documentationHashInput)),
		SourceHashDigest:  sourceHashDigest(evidence.Files),
	}, nil
}

func splitDocumentParts(markdown string) (string, string, string, error) {
	if !strings.HasPrefix(markdown, "---\n") {
		return "", "", "", errors.New("missing opening YAML frontmatter delimiter")
	}
	frontmatterEnd := strings.Index(markdown[4:], "\n---\n")
	if frontmatterEnd < 0 {
		return "", "", "", errors.New("missing closing YAML frontmatter delimiter")
	}
	frontmatterEnd += 4
	contentStart := frontmatterEnd + len("\n---\n")
	frontmatter := markdown[4:frontmatterEnd]
	body := markdown[contentStart:]

	filesHeader := "\n### FILES\n"
	filesOffset := strings.LastIndex(body, filesHeader)
	if filesOffset < 0 {
		return "", "", "", errors.New("missing terminal ### FILES section")
	}
	indexableMarkdown := body[:filesOffset]
	filesMarkdown := body[filesOffset+len(filesHeader):]
	return frontmatter, indexableMarkdown, filesMarkdown, nil
}

func validateFrontmatter(repositoryPath string, metadata Frontmatter) error {
	if metadata.Schema != 1 {
		return fmt.Errorf("%s: unsupported documentation schema %d", repositoryPath, metadata.Schema)
	}
	if metadata.PageID == "" || metadata.Route == "" || metadata.Title == "" || metadata.Status == "" || metadata.Visibility == "" {
		return fmt.Errorf("%s: frontmatter requires page_id, route, title, status, and visibility", repositoryPath)
	}
	if metadata.Status != "implemented" {
		return fmt.Errorf("%s: production documentation status must be implemented, got %q", repositoryPath, metadata.Status)
	}
	routeDirectory := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(filepath.Dir(repositoryPath)), "frontend/routes"), "/")
	expectedRoute := routeDirectory
	if expectedRoute == "" {
		expectedRoute = "/"
	}
	if metadata.Route != expectedRoute {
		return fmt.Errorf("%s: route %q does not match directory route %q", repositoryPath, metadata.Route, expectedRoute)
	}
	return nil
}

func parseSections(indexableMarkdown string) ([]Section, error) {
	matches := docIDPattern.FindAllStringSubmatchIndex(indexableMarkdown, -1)
	if len(matches) == 0 {
		return nil, errors.New("no DOC-ID sections found")
	}

	sections := make([]Section, 0, len(matches))
	seenIDs := map[string]bool{}
	for matchIndex, match := range matches {
		sectionID := indexableMarkdown[match[2]:match[3]]
		if seenIDs[sectionID] {
			return nil, fmt.Errorf("duplicate DOC-ID %q", sectionID)
		}
		seenIDs[sectionID] = true

		sectionEnd := len(indexableMarkdown)
		if matchIndex+1 < len(matches) {
			sectionEnd = matches[matchIndex+1][0]
		}
		sectionMarkdown := normalizeMarkdown(htmlCommentPattern.ReplaceAllString(indexableMarkdown[match[1]:sectionEnd], ""))
		titleMatch := sectionTitlePattern.FindStringSubmatch(sectionMarkdown)
		if len(titleMatch) != 2 {
			return nil, fmt.Errorf("DOC-ID %q has no ## section heading", sectionID)
		}
		sections = append(sections, Section{
			ID:       sectionID,
			Title:    strings.TrimSpace(titleMatch[1]),
			Type:     sectionType(sectionID),
			Markdown: sectionMarkdown,
		})
	}
	if !seenIDs["page-purpose"] {
		return nil, errors.New("required DOC-ID page-purpose is missing")
	}
	return sections, nil
}

func validateEvidence(repositoryRoot, repositoryPath string, evidence EvidenceManifest, sections []Section) error {
	if evidence.Schema != 1 || strings.ToLower(evidence.HashAlgorithm) != "sha256" {
		return fmt.Errorf("%s: FILES requires schema 1 and hash_algorithm sha256", repositoryPath)
	}
	if len(evidence.Files) == 0 {
		return fmt.Errorf("%s: FILES contains no source evidence", repositoryPath)
	}
	sectionIDs := map[string]bool{}
	for _, section := range sections {
		sectionIDs[section.ID] = true
	}

	seenPaths := map[string]bool{}
	for _, evidenceFile := range evidence.Files {
		if evidenceFile.Path == "" || filepath.IsAbs(evidenceFile.Path) || strings.Contains(filepath.ToSlash(evidenceFile.Path), "../") {
			return fmt.Errorf("%s: invalid evidence path %q", repositoryPath, evidenceFile.Path)
		}
		if seenPaths[evidenceFile.Path] {
			return fmt.Errorf("%s: duplicate evidence path %q", repositoryPath, evidenceFile.Path)
		}
		seenPaths[evidenceFile.Path] = true
		if !allowedEvidenceRoles[evidenceFile.Role] {
			return fmt.Errorf("%s: unknown evidence role %q for %s", repositoryPath, evidenceFile.Role, evidenceFile.Path)
		}
		if len(evidenceFile.Supports) == 0 {
			return fmt.Errorf("%s: evidence %s supports no DOC-ID", repositoryPath, evidenceFile.Path)
		}
		for _, supportedID := range evidenceFile.Supports {
			if !sectionIDs[supportedID] {
				return fmt.Errorf("%s: evidence %s references missing DOC-ID %q", repositoryPath, evidenceFile.Path, supportedID)
			}
		}
		if evidenceFile.Hash == "pending" {
			return fmt.Errorf("%s: evidence hash is pending for %s", repositoryPath, evidenceFile.Path)
		}
		if !validSHA256(evidenceFile.Hash) {
			return fmt.Errorf("%s: invalid SHA-256 for %s", repositoryPath, evidenceFile.Path)
		}

		sourceBytes, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(evidenceFile.Path)))
		if err != nil {
			return fmt.Errorf("%s: read evidence %s: %w", repositoryPath, evidenceFile.Path, err)
		}
		actualHash := hashBytes(sourceBytes)
		if actualHash != evidenceFile.Hash {
			return fmt.Errorf("%s: stale evidence %s: stored %s, current %s", repositoryPath, evidenceFile.Path, evidenceFile.Hash, actualHash)
		}
	}
	return nil
}

func normalizeMarkdown(markdown string) string {
	lines := strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n")
	normalizedLines := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
		} else {
			blankCount = 0
		}
		normalizedLines = append(normalizedLines, line)
	}
	return strings.TrimSpace(strings.Join(normalizedLines, "\n"))
}

func canonicalDocumentation(metadata Frontmatter, sections []Section) string {
	var canonical strings.Builder
	fmt.Fprintf(&canonical, "schema=%d\npage_id=%s\nroute=%s\ntitle=%s\nstatus=%s\nvisibility=%s\n",
		metadata.Schema, metadata.PageID, metadata.Route, metadata.Title, metadata.Status, metadata.Visibility)
	for _, section := range sections {
		fmt.Fprintf(&canonical, "\nDOC-ID:%s\n%s\n", section.ID, section.Markdown)
	}
	return canonical.String()
}

func sourceHashDigest(files []EvidenceFile) string {
	entries := make([]string, 0, len(files))
	for _, evidenceFile := range files {
		entries = append(entries, evidenceFile.Path+"="+evidenceFile.Hash)
	}
	sort.Strings(entries)
	return hashBytes([]byte(strings.Join(entries, "\n")))
}

func hashBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func sectionType(sectionID string) string {
	switch {
	case sectionID == "page-purpose":
		return "purpose"
	case sectionID == "concepts":
		return "concept"
	case strings.HasPrefix(sectionID, "capability."):
		return "capability"
	case sectionID == "rules":
		return "rule"
	case sectionID == "troubleshooting":
		return "troubleshooting"
	case sectionID == "related-pages":
		return "related-page"
	default:
		return "section"
	}
}
