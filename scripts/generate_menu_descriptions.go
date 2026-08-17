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

// Route documentation lives in exactly one file per route.
const documentationFileName = "DOCUMENTATION.md"

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
	// One DOCUMENTATION.md per directory and a route taken from that directory means a route can
	// never be described twice, so no de-duplication is needed here.
	var menuDescriptions []menuDescription

	err := filepath.WalkDir(routesDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// DOCUMENTATION.md is the single source of route documentation, so no other markdown file
		// can contribute a menu description.
		if entry.IsDir() || entry.Name() != documentationFileName {
			return nil
		}

		menuEntry, found, err := parseMenuDescriptionFile(routesDir, path)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}

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

	// The description lives in the YAML frontmatter, next to route and title.
	spanishDescription, englishDescription, err := parseFrontmatterDescriptions(string(contentBytes))
	if err != nil {
		return menuDescription{}, false, fmt.Errorf("%s: %w", markdownPath, err)
	}
	if spanishDescription == "" && englishDescription == "" {
		return menuDescription{}, false, nil
	}
	if spanishDescription == "" || englishDescription == "" {
		return menuDescription{}, false, fmt.Errorf("%s must include description_en and description_es", markdownPath)
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

// parseFrontmatterDescriptions reads the description fields from a leading YAML frontmatter block.
// A document that omits both fields returns empty strings and is skipped rather than failing, so a
// route may be documented for retrieval before it earns a menu entry.
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

func routeFromMarkdownPath(routesDir string, markdownPath string) (string, error) {
	routeDir := filepath.Dir(markdownPath)

	// A menu description only makes sense for a real page. Without this the generator would publish
	// a route that no menu option can ever match.
	if !isFile(filepath.Join(routeDir, "+page.svelte")) {
		return "", fmt.Errorf("%s describes a menu route but has no +page.svelte beside it", markdownPath)
	}

	relativePath, err := filepath.Rel(routesDir, routeDir)
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
