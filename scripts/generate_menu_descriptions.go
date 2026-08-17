package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type menuDescription struct {
	Route              string `json:"route"`
	Description        string `json:"description"`
	DescriptionSpanish string `json:"descriptionSpanish"`
}

func GenerateMenuDescriptions() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		exitWithMenuDescriptionError(err)
	}

	routesDir := filepath.Join(repoRoot, "frontend", "routes")
	outputPath := filepath.Join(repoRoot, "tmp", "menu_description.json")

	fmt.Printf("Scanning markdown route descriptions in %s\n", routesDir)

	menuDescriptions, err := collectMenuDescriptions(routesDir)
	if err != nil {
		exitWithMenuDescriptionError(err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		exitWithMenuDescriptionError(fmt.Errorf("create output folder: %w", err))
	}

	jsonContent, err := json.MarshalIndent(menuDescriptions, "", "  ")
	if err != nil {
		exitWithMenuDescriptionError(fmt.Errorf("encode menu descriptions: %w", err))
	}

	if err := os.WriteFile(outputPath, append(jsonContent, '\n'), 0644); err != nil {
		exitWithMenuDescriptionError(fmt.Errorf("write %s: %w", outputPath, err))
	}

	fmt.Printf("Generated %d menu descriptions at %s\n", len(menuDescriptions), outputPath)
}

func collectMenuDescriptions(routesDir string) ([]menuDescription, error) {
	var menuDescriptions []menuDescription
	sourceByRoute := map[string]string{}

	err := filepath.WalkDir(routesDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Only route markdown files carrying a description are part of the menu description index.
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		menuEntry, found, err := parseMenuDescriptionFile(routesDir, path)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}

		// AttachMenuDescriptions keys descriptions by route, so a second file for the same route
		// would silently overwrite the first. Fail loudly instead of shipping an arbitrary winner.
		if previousPath, duplicated := sourceByRoute[menuEntry.Route]; duplicated {
			return fmt.Errorf("route %s is described twice: %s and %s", menuEntry.Route, previousPath, path)
		}
		sourceByRoute[menuEntry.Route] = path

		menuDescriptions = append(menuDescriptions, menuEntry)
		fmt.Printf("Added %s from %s\n", menuEntry.Route, path)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan route markdown files: %w", err)
	}

	sort.Slice(menuDescriptions, func(leftIndex, rightIndex int) bool {
		return menuDescriptions[leftIndex].Route < menuDescriptions[rightIndex].Route
	})

	return menuDescriptions, nil
}

func parseMenuDescriptionFile(routesDir string, markdownPath string) (menuDescription, bool, error) {
	contentBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		return menuDescription{}, false, fmt.Errorf("read %s: %w", markdownPath, err)
	}

	markdownContent := string(contentBytes)

	// DOCUMENTATION.md carries its description in YAML frontmatter next to route and title. Routes
	// without a DOCUMENTATION.md still use the legacy "## DESCRIPTION::" stub file.
	spanishDescription, englishDescription, err := parseFrontmatterDescriptions(markdownContent)
	if err != nil {
		return menuDescription{}, false, fmt.Errorf("%s: %w", markdownPath, err)
	}
	requiredFields := "description_en and description_es"
	if spanishDescription == "" && englishDescription == "" {
		descriptions := parseDescriptionBlocks(markdownContent)
		spanishDescription = descriptions["ES"]
		englishDescription = descriptions["EN"]
		requiredFields = "DESCRIPTION::ES and DESCRIPTION::EN"
	}

	if spanishDescription == "" && englishDescription == "" {
		return menuDescription{}, false, nil
	}
	if spanishDescription == "" || englishDescription == "" {
		return menuDescription{}, false, fmt.Errorf("%s must include %s", markdownPath, requiredFields)
	}

	route, err := routeFromMarkdownPath(routesDir, markdownPath)
	if err != nil {
		return menuDescription{}, false, err
	}

	return menuDescription{
		Route:              route,
		Description:        englishDescription,
		DescriptionSpanish: spanishDescription,
	}, true, nil
}

// parseFrontmatterDescriptions reads the optional description fields from a leading YAML
// frontmatter block. Files without frontmatter, or whose frontmatter omits both fields, return
// empty strings so the caller can fall back to the legacy description blocks.
func parseFrontmatterDescriptions(markdownContent string) (string, string, error) {
	normalizedContent := strings.ReplaceAll(markdownContent, "\r\n", "\n")
	if !strings.HasPrefix(normalizedContent, "---\n") {
		return "", "", nil
	}
	frontmatterEnd := strings.Index(normalizedContent[len("---\n"):], "\n---\n")
	if frontmatterEnd < 0 {
		return "", "", nil
	}
	frontmatter := normalizedContent[len("---\n") : len("---\n")+frontmatterEnd]

	var descriptionFields struct {
		DescriptionEnglish string `yaml:"description_en"`
		DescriptionSpanish string `yaml:"description_es"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &descriptionFields); err != nil {
		return "", "", fmt.Errorf("parse frontmatter: %w", err)
	}

	return strings.TrimSpace(descriptionFields.DescriptionSpanish),
		strings.TrimSpace(descriptionFields.DescriptionEnglish), nil
}

func parseDescriptionBlocks(markdownContent string) map[string]string {
	descriptions := map[string]string{}
	var activeLanguage string
	var activeLines []string

	flushActiveBlock := func() {
		if activeLanguage == "" {
			return
		}

		// Preserve paragraph content while trimming markdown spacing around the block.
		descriptions[activeLanguage] = strings.TrimSpace(strings.Join(activeLines, "\n"))
		activeLanguage = ""
		activeLines = nil
	}

	for _, line := range strings.Split(markdownContent, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "## DESCRIPTION::") {
			flushActiveBlock()
			activeLanguage = strings.TrimPrefix(trimmedLine, "## DESCRIPTION::")
			activeLines = nil
			continue
		}

		if strings.HasPrefix(trimmedLine, "## ") {
			flushActiveBlock()
			continue
		}

		if activeLanguage != "" {
			activeLines = append(activeLines, line)
		}
	}

	flushActiveBlock()

	return descriptions
}

func routeFromMarkdownPath(routesDir string, markdownPath string) (string, error) {
	routePath := strings.TrimSuffix(markdownPath, ".md")
	if isFile(filepath.Join(filepath.Dir(markdownPath), "+page.svelte")) {
		// Route descriptions live next to +page.svelte, so the URL is the folder path.
		routePath = filepath.Dir(markdownPath)
	}

	relativePath, err := filepath.Rel(routesDir, routePath)
	if err != nil {
		return "", fmt.Errorf("build route for %s: %w", markdownPath, err)
	}

	return "/" + filepath.ToSlash(relativePath), nil
}

func findRepoRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("read working directory: %w", err)
	}

	for {
		routesDir := filepath.Join(currentDir, "frontend", "routes")
		if isDir(routesDir) {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("could not find repo root containing frontend/routes")
		}
		currentDir = parentDir
	}
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && fileInfo.IsDir()
}

func isFile(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && !fileInfo.IsDir()
}

func exitWithMenuDescriptionError(err error) {
	fmt.Fprintf(os.Stderr, "generate_menu_descriptions failed: %v\n", err)
	os.Exit(1)
}
