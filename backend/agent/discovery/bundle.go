package discovery

type Bundle struct {
	Plan                    Plan                `json:"plan"`
	DocumentationNavigation FeatureSearchResult `json:"documentation_navigation"`
	AgentTools              ToolSearchResult    `json:"agent_tools"`
}

func EmptyFeatureResult() FeatureSearchResult {
	return FeatureSearchResult{
		Status: DiscoveryStatusOK, Routes: []RouteCandidate{}, Passages: []DocumentationPassage{},
		Diagnostics: FeatureSearchDiagnostics{DocumentationStatus: DiscoveryStatusOK},
	}
}

func EmptyToolResult() ToolSearchResult {
	return ToolSearchResult{Status: DiscoveryStatusOK, CatalogVersion: ToolCatalogVersion, Tools: []ToolDescriptor{}}
}
