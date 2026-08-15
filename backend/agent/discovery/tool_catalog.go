package discovery

import (
	"context"
	"strings"
)

const ToolCatalogVersion = 1

type ToolSearchRequest struct {
	Query              string             `json:"query"`
	Domain             string             `json:"domain,omitempty"`
	Operation          Operation          `json:"operation"`
	DeliveryPreference DeliveryPreference `json:"delivery_preference"`
	ResultLimit        int                `json:"result_limit,omitempty"`
}

type ToolDescriptor struct {
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Domain            string    `json:"domain"`
	Operation         Operation `json:"operation"`
	OutputType        string    `json:"output_type"`
	RequiredArguments []string  `json:"required_arguments"`
	OptionalArguments []string  `json:"optional_arguments"`
	ReadOnly          bool      `json:"read_only"`
}

type ToolSearchResult struct {
	Status         string           `json:"status"`
	CatalogVersion int              `json:"catalog_version"`
	Tools          []ToolDescriptor `json:"tools"`
}

type ToolCatalog struct {
	descriptors []ToolDescriptor
}

// NewToolCatalog starts empty intentionally. Future read-only tools enter only
// through validated registration, without changing discovery orchestration.
func NewToolCatalog() *ToolCatalog {
	return &ToolCatalog{descriptors: []ToolDescriptor{}}
}

func (catalog *ToolCatalog) Search(_ context.Context, request ToolSearchRequest) ToolSearchResult {
	result := ToolSearchResult{Status: DiscoveryStatusOK, CatalogVersion: ToolCatalogVersion, Tools: []ToolDescriptor{}}
	if catalog == nil || len(catalog.descriptors) == 0 {
		return result
	}
	queryTokens := searchTokens(request.Query + " " + request.Domain)
	limit := request.ResultLimit
	if limit <= 0 || limit > 12 {
		limit = 6
	}
	for _, descriptor := range catalog.descriptors {
		if request.Domain != "" && descriptor.Domain != "" && !strings.EqualFold(request.Domain, descriptor.Domain) {
			continue
		}
		if request.Operation != OperationNone && descriptor.Operation != request.Operation {
			continue
		}
		candidateTokens := searchTokens(descriptor.Name + " " + descriptor.Description + " " + descriptor.Domain)
		matches := len(queryTokens) == 0
		for token := range queryTokens {
			matches = matches || candidateTokens[token]
		}
		if matches {
			result.Tools = append(result.Tools, descriptor)
		}
		if len(result.Tools) == limit {
			break
		}
	}
	return result
}
