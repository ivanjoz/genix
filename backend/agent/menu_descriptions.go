package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"app/core"
)

type menuDescriptionFileEntry struct {
	Route       string `json:"route"`
	Description string `json:"description"`
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
			return "", fmt.Errorf("tmp/menu_description.json not found from working directory %s", originalDir)
		}
		currentDir = parentDir
	}
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

func tsvCell(value string) string {
	replacer := strings.NewReplacer("\t", " ","\r\n", " ","\n", " ","\r", " ")
	return replacer.Replace(strings.TrimSpace(value))
}
