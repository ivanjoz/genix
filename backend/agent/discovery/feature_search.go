package discovery

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"app/agent/knowledge"
	"golang.org/x/text/unicode/norm"
)

const (
	DiscoveryStatusOK          = "ok"
	DiscoveryStatusUnavailable = "unavailable"
	defaultFeatureResultLimit  = 6
)

type FeatureSearchRequest struct {
	Query       string    `json:"query"`
	Domain      string    `json:"domain,omitempty"`
	Operation   Operation `json:"operation"`
	ResultLimit int       `json:"result_limit,omitempty"`
}

type AccessibleFeature struct {
	Group       string `json:"group,omitempty"`
	Name        string `json:"name"`
	Route       string `json:"route"`
	Description string `json:"description,omitempty"`
}

type RouteCandidate struct {
	Route       string  `json:"route"`
	PageName    string  `json:"page_name"`
	Description string  `json:"description,omitempty"`
	MatchedBy   string  `json:"matched_by"`
	Score       float32 `json:"score"`
}

type DocumentationPassage struct {
	CitationID   string `json:"citation_id"`
	Route        string `json:"route"`
	PageTitle    string `json:"page_title"`
	SectionTitle string `json:"section_title"`
	Content      string `json:"content"`
}

type FeatureSearchDiagnostics struct {
	DocumentationStatus  string `json:"documentation_status"`
	DocumentationMatches int    `json:"documentation_matches"`
	MenuMatches          int    `json:"menu_matches"`
}

type FeatureSearchResult struct {
	Status      string                   `json:"status"`
	Routes      []RouteCandidate         `json:"routes"`
	Passages    []DocumentationPassage   `json:"passages"`
	Diagnostics FeatureSearchDiagnostics `json:"diagnostics"`
}

type DocumentationSearcher interface {
	SearchDocumentation(context.Context, string, knowledge.SearchOptions) ([]knowledge.DocumentationResult, error)
}

// SearchDocumentationNavigation merges verified passages with accessible-menu
// matching. Documentation failure preserves menu-only navigation candidates.
func SearchDocumentationNavigation(ctx context.Context, request FeatureSearchRequest, features []AccessibleFeature, searcher DocumentationSearcher) FeatureSearchResult {
	limit := request.ResultLimit
	if limit <= 0 || limit > 12 {
		limit = defaultFeatureResultLimit
	}
	result := FeatureSearchResult{
		Status: DiscoveryStatusOK, Routes: []RouteCandidate{}, Passages: []DocumentationPassage{},
		Diagnostics: FeatureSearchDiagnostics{DocumentationStatus: DiscoveryStatusOK},
	}
	menuMatches := matchAccessibleFeatures(request.Query, request.Domain, features, limit)
	result.Diagnostics.MenuMatches = len(menuMatches)

	allowedRoutes := make([]string, 0, len(features))
	featureByRoute := make(map[string]AccessibleFeature, len(features))
	for _, feature := range features {
		route := strings.TrimSpace(feature.Route)
		if route == "" {
			continue
		}
		allowedRoutes = append(allowedRoutes, route)
		feature.Route = route
		featureByRoute[route] = feature
	}

	documentationResults := []knowledge.DocumentationResult{}
	if len(allowedRoutes) == 0 || searcher == nil {
		result.Diagnostics.DocumentationStatus = DiscoveryStatusUnavailable
	} else {
		var err error
		documentationResults, err = searcher.SearchDocumentation(ctx, request.Query, knowledge.SearchOptions{
			CandidateLimit: 25, ResultLimit: uint64(limit), AllowedRoutes: allowedRoutes,
		})
		if err != nil {
			result.Diagnostics.DocumentationStatus = DiscoveryStatusUnavailable
			documentationResults = nil
		}
	}
	result.Diagnostics.DocumentationMatches = len(documentationResults)

	routesByPath := map[string]RouteCandidate{}
	for _, candidate := range menuMatches {
		routesByPath[candidate.Route] = candidate
	}
	for _, documentation := range documentationResults {
		feature, accessible := featureByRoute[strings.TrimSpace(documentation.Route)]
		if !accessible {
			continue
		}
		result.Passages = append(result.Passages, DocumentationPassage{
			CitationID: documentation.CitationID, Route: feature.Route, PageTitle: documentation.PageTitle,
			SectionTitle: documentation.SectionTitle, Content: strings.TrimSpace(documentation.Content),
		})
		candidate, menuMatched := routesByPath[feature.Route]
		if menuMatched {
			candidate.MatchedBy = "menu_and_documentation"
			candidate.Score += documentation.Score
		} else {
			candidate = RouteCandidate{
				Route: feature.Route, PageName: feature.Name, Description: feature.Description,
				MatchedBy: "documentation", Score: documentation.Score,
			}
		}
		routesByPath[feature.Route] = candidate
	}

	result.Routes = make([]RouteCandidate, 0, len(routesByPath))
	for _, candidate := range routesByPath {
		result.Routes = append(result.Routes, candidate)
	}
	sort.SliceStable(result.Routes, func(left, right int) bool {
		if result.Routes[left].Score == result.Routes[right].Score {
			return result.Routes[left].Route < result.Routes[right].Route
		}
		return result.Routes[left].Score > result.Routes[right].Score
	})
	if len(result.Routes) > limit {
		result.Routes = result.Routes[:limit]
	}
	return result
}

func matchAccessibleFeatures(query, domain string, features []AccessibleFeature, limit int) []RouteCandidate {
	queryTokens := searchTokens(query + " " + domain)
	if len(queryTokens) == 0 {
		return nil
	}
	candidates := make([]RouteCandidate, 0, len(features))
	for _, feature := range features {
		route := strings.TrimSpace(feature.Route)
		if route == "" {
			continue
		}
		candidateTokens := searchTokens(feature.Group + " " + feature.Name + " " + feature.Description + " " + route)
		matched := 0
		for token := range queryTokens {
			if candidateTokens[token] {
				matched++
			}
		}
		if matched == 0 {
			continue
		}
		score := float32(matched) / float32(len(queryTokens))
		candidates = append(candidates, RouteCandidate{
			Route: route, PageName: strings.TrimSpace(feature.Name), Description: strings.TrimSpace(feature.Description),
			MatchedBy: "menu", Score: score,
		})
	}
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].Score == candidates[right].Score {
			return candidates[left].Route < candidates[right].Route
		}
		return candidates[left].Score > candidates[right].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func searchTokens(value string) map[string]bool {
	normalized := strings.ToLower(norm.NFD.String(value))
	fields := strings.FieldsFunc(normalized, func(character rune) bool {
		return unicode.Is(unicode.Mn, character) || (!unicode.IsLetter(character) && !unicode.IsDigit(character))
	})
	stopWords := map[string]bool{
		"a": true, "al": true, "and": true, "de": true, "del": true, "el": true, "en": true,
		"for": true, "i": true, "la": true, "las": true, "los": true, "me": true, "of": true,
		"para": true, "por": true, "que": true, "the": true, "to": true, "un": true, "una": true,
		"quiero": true, "want": true,
	}
	tokens := map[string]bool{}
	for _, field := range fields {
		if len(field) >= 2 && !stopWords[field] {
			tokens[field] = true
		}
	}
	return tokens
}
