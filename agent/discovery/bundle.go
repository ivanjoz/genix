package discovery

type Bundle struct {
	Plan                    Plan                `json:"plan" cb:"plan,1"`
	DocumentationNavigation FeatureSearchResult `json:"documentation_navigation" cb:"documentation_navigation,2"`
	AgentTools              ToolSearchResult    `json:"agent_tools" cb:"agent_tools,3"`
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
