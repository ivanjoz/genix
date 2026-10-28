package discovery

import (
	"context"
	"strings"
)

const ToolCatalogVersion = 1

type ToolSearchRequest struct {
	Query              string             `json:"query" cb:"query,1"`
	Domain             string             `json:"domain" cb:"domain,2"`
	Operation          Operation          `json:"operation" cb:"operation,3"`
	DeliveryPreference DeliveryPreference `json:"delivery_preference" cb:"delivery_preference,4"`
	ResultLimit        int                `json:"result_limit" cb:"result_limit,5"`
}

type ToolDescriptor struct {
	Name              string    `json:"name" cb:"name,1"`
	Description       string    `json:"description" cb:"description,2"`
	Domain            string    `json:"domain" cb:"domain,3"`
	Operation         Operation `json:"operation" cb:"operation,4"`
	OutputType        string    `json:"output_type" cb:"output_type,5"`
	RequiredArguments []string  `json:"required_arguments" cb:"required_arguments,6"`
	OptionalArguments []string  `json:"optional_arguments" cb:"optional_arguments,7"`
	ReadOnly          bool      `json:"read_only" cb:"read_only,8"`
}

type ToolSearchResult struct {
	Status         string           `json:"status" cb:"status,1"`
	CatalogVersion int              `json:"catalog_version" cb:"catalog_version,2"`
	Tools          []ToolDescriptor `json:"tools" cb:"tools,3"`
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
