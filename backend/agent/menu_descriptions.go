package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"app/core"
)

type menuDescriptionFileEntry struct {
	Route              string `json:"route"`
	Description        string `json:"description"`
	DescriptionSpanish string `json:"descriptionSpanish,omitempty"`
}

func AttachMenuDescriptions(menuGroups []AgentMenuGroup) error {
	descriptionPath, err := findMenuDescriptionPath()
	if err != nil {
		return err
	}

	descriptionsByRoute, err := loadMenuDescriptions(descriptionPath)
	if err != nil {
		return err
	}

	matchedCount := 0
	missingRoutes := []string{}
	for groupIndex := range menuGroups {
		for optionIndex := range menuGroups[groupIndex].Options {
			option := &menuGroups[groupIndex].Options[optionIndex]
			description, exists := descriptionsByRoute[normalizeMenuRoute(option.Route)]
			if !exists {
				missingRoutes = append(missingRoutes, option.Route)
				continue
			}
			option.Description = description
			matchedCount++
		}
	}

	// Log mismatches without hiding the accessible menu from the agent.
	if len(missingRoutes) > 0 {
		core.Log("agent.menu descriptions missing routes::", strings.Join(missingRoutes, ", "))
	}
	core.Log("agent.menu descriptions attached::", matchedCount, " source::", descriptionPath)

	return nil
}

func FormatMenuTSV(menuGroups []AgentMenuGroup) string {
	var builder strings.Builder
	builder.WriteString("group_id\tgroup_name\toption_name\troute\tdescription\n")

	for _, group := range menuGroups {
		groupID := strconv.Itoa(group.ID)
		for _, option := range group.Options {
			// TSV keeps route discovery compact for agents while preserving all fields needed for navigate.
			builder.WriteString(tsvCell(groupID))
			builder.WriteByte('\t')
			builder.WriteString(tsvCell(group.Name))
			builder.WriteByte('\t')
			builder.WriteString(tsvCell(option.Name))
			builder.WriteByte('\t')
			builder.WriteString(tsvCell(option.Route))
			builder.WriteByte('\t')
			builder.WriteString(tsvCell(option.Description))
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

func loadMenuDescriptions(descriptionPath string) (map[string]string, error) {
	fileBytes, err := os.ReadFile(descriptionPath)
	if err != nil {
		return nil, fmt.Errorf("read menu descriptions %s: %w", descriptionPath, err)
	}

	var descriptionEntries []menuDescriptionFileEntry
	if err := json.Unmarshal(fileBytes, &descriptionEntries); err != nil {
		return nil, fmt.Errorf("parse menu descriptions %s: %w", descriptionPath, err)
	}

	descriptionsByRoute := make(map[string]string, len(descriptionEntries))
	for _, entry := range descriptionEntries {
		route := normalizeMenuRoute(entry.Route)
		if route == "" || entry.Description == "" {
			continue
		}
		descriptionsByRoute[route] = entry.Description
	}

	if len(descriptionsByRoute) == 0 {
		return nil, fmt.Errorf("menu descriptions %s has no usable route descriptions", descriptionPath)
	}

	return descriptionsByRoute, nil
}

func findMenuDescriptionPath() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("read working directory: %w", err)
	}
	originalDir := currentDir

	for {
		descriptionPath := filepath.Join(currentDir, "tmp", "menu_description.json")
		if isReadableFile(descriptionPath) {
			return descriptionPath, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	generatedPath, err := generateMenuDescriptionFile(originalDir)
	if err != nil {
		return "", fmt.Errorf("tmp/menu_description.json not found from working directory %s and generation failed: %w", originalDir, err)
	}
	return generatedPath, nil
}

func generateMenuDescriptionFile(workingDir string) (string, error) {
	repoRoot, err := findMenuDescriptionRepoRoot(workingDir)
	if err != nil {
		return "", err
	}
	routesDir := filepath.Join(repoRoot, "frontend", "routes")
	// The systemd service grants write access to WorkingDirectory only.
	outputPath := filepath.Join(workingDir, "tmp", "menu_description.json")
	core.Log("agent.menu generating descriptions repo::", repoRoot, " output::", outputPath)

	menuDescriptions, err := collectMenuDescriptions(routesDir)
	if err != nil {
		return "", err
	}
	if len(menuDescriptions) == 0 {
		return "", fmt.Errorf("no DESCRIPTION blocks found under %s", routesDir)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("create menu description output dir: %w", err)
	}
	jsonContent, err := json.MarshalIndent(menuDescriptions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode menu descriptions: %w", err)
	}
	if err := os.WriteFile(outputPath, append(jsonContent, '\n'), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", outputPath, err)
	}
	core.Log("agent.menu generated descriptions count::", len(menuDescriptions), " output::", outputPath)
	return outputPath, nil
}

func findMenuDescriptionRepoRoot(workingDir string) (string, error) {
	candidates := []string{
		strings.TrimSpace(os.Getenv("GENIX_REPOSITORY_ROOT")),
		filepath.Dir(strings.TrimSpace(os.Getenv("GENIX_CREDENTIALS_FILE"))),
		workingDir,
	}
	for _, candidate := range candidates {
		if root := walkUpToRoutesDir(candidate); root != "" {
			return root, nil
		}
	}
	return "", fmt.Errorf("could not find repo root containing frontend/routes; set GENIX_REPOSITORY_ROOT")
}

func walkUpToRoutesDir(startDir string) string {
	currentDir := strings.TrimSpace(startDir)
	if currentDir == "" || currentDir == "." {
		return ""
	}
	if info, err := os.Stat(currentDir); err == nil && !info.IsDir() {
		currentDir = filepath.Dir(currentDir)
	}
	for {
		if isReadableDir(filepath.Join(currentDir, "frontend", "routes")) {
			return currentDir
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return ""
		}
		currentDir = parentDir
	}
}

func collectMenuDescriptions(routesDir string) ([]menuDescriptionFileEntry, error) {
	menuDescriptions := []menuDescriptionFileEntry{}
	err := filepath.WalkDir(routesDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Only markdown files with bilingual DESCRIPTION blocks are indexed.
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		menuEntry, found, err := parseMenuDescriptionFile(routesDir, path)
		if err != nil {
			return err
		}
		if found {
			menuDescriptions = append(menuDescriptions, menuEntry)
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

func parseMenuDescriptionFile(routesDir string, markdownPath string) (menuDescriptionFileEntry, bool, error) {
	contentBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		return menuDescriptionFileEntry{}, false, fmt.Errorf("read %s: %w", markdownPath, err)
	}
	descriptions := parseDescriptionBlocks(string(contentBytes))
	spanishDescription := descriptions["ES"]
	englishDescription := descriptions["EN"]
	if spanishDescription == "" && englishDescription == "" {
		return menuDescriptionFileEntry{}, false, nil
	}
	if spanishDescription == "" || englishDescription == "" {
		return menuDescriptionFileEntry{}, false, fmt.Errorf("%s must include DESCRIPTION::ES and DESCRIPTION::EN", markdownPath)
	}
	route, err := routeFromMarkdownPath(routesDir, markdownPath)
	if err != nil {
		return menuDescriptionFileEntry{}, false, err
	}
	return menuDescriptionFileEntry{
		Route:              route,
		Description:        englishDescription,
		DescriptionSpanish: spanishDescription,
	}, true, nil
}

func parseDescriptionBlocks(markdownContent string) map[string]string {
	descriptions := map[string]string{}
	activeLanguage := ""
	activeLines := []string{}

	flushActiveBlock := func() {
		if activeLanguage == "" {
			return
		}
		// Preserve paragraphs while trimming markdown spacing around each block.
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
	if isReadableFile(filepath.Join(filepath.Dir(markdownPath), "+page.svelte")) {
		// Route descriptions next to +page.svelte describe the folder route.
		routePath = filepath.Dir(markdownPath)
	}
	relativePath, err := filepath.Rel(routesDir, routePath)
	if err != nil {
		return "", fmt.Errorf("build route for %s: %w", markdownPath, err)
	}
	return "/" + filepath.ToSlash(relativePath), nil
}

func normalizeMenuRoute(route string) string {
	trimmedRoute := strings.TrimSpace(route)
	if trimmedRoute == "" {
		return ""
	}
	if !strings.HasPrefix(trimmedRoute, "/") {
		trimmedRoute = "/" + trimmedRoute
	}
	if len(trimmedRoute) > 1 {
		trimmedRoute = strings.TrimRight(trimmedRoute, "/")
	}
	return trimmedRoute
}

func isReadableFile(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && !fileInfo.IsDir()
}

func isReadableDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return err == nil && fileInfo.IsDir()
}

func tsvCell(value string) string {
	replacer := strings.NewReplacer("\t", " ", "\r\n", " ", "\n", " ", "\r", " ")
	return replacer.Replace(strings.TrimSpace(value))
}
