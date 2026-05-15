package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	err := filepath.WalkDir(routesDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Only route markdown files with DESCRIPTION blocks are part of the menu description index.
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		menuEntry, found, err := parseMenuDescriptionFile(routesDir, path)
		if err != nil {
			return err
		}
		if found {
			menuDescriptions = append(menuDescriptions, menuEntry)
			fmt.Printf("Added %s from %s\n", menuEntry.Route, path)
		}

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

	descriptions := parseDescriptionBlocks(string(contentBytes))
	spanishDescription := descriptions["ES"]
	englishDescription := descriptions["EN"]
	if spanishDescription == "" && englishDescription == "" {
		return menuDescription{}, false, nil
	}
	if spanishDescription == "" || englishDescription == "" {
		return menuDescription{}, false, fmt.Errorf("%s must include DESCRIPTION::ES and DESCRIPTION::EN", markdownPath)
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
