package discovery

import (
	"context"
	"errors"
	"testing"

	"app/agent/knowledge"
)

type fakeDocumentationSearcher struct {
	results []knowledge.DocumentationResult
	err     error
	options knowledge.SearchOptions
}

func (fake *fakeDocumentationSearcher) SearchDocumentation(_ context.Context, _ string, options knowledge.SearchOptions) ([]knowledge.DocumentationResult, error) {
	fake.options = options
	return fake.results, fake.err
}

func TestFeatureSearchMergesBilingualMenuAndDocumentation(t *testing.T) {
	features := []AccessibleFeature{{
		Name: "Products", Route: "/business/products",
		Description: "Create and edit products | Crear y editar productos",
	}}
	fake := &fakeDocumentationSearcher{results: []knowledge.DocumentationResult{{
		CitationID: "business.products#capability.create", Route: "/business/products", PageTitle: "Products",
		SectionTitle: "Create", Content: "Create products from this page.", Score: 0.8,
	}}}
	result := SearchDocumentationNavigation(context.Background(), FeatureSearchRequest{
		Query: "crear un producto", Domain: "products", Operation: OperationCreate,
	}, features, fake)
	if len(result.Routes) != 1 || result.Routes[0].MatchedBy != "menu_and_documentation" || len(result.Passages) != 1 {
		t.Fatalf("unexpected merged discovery result: %+v", result)
	}
	if len(fake.options.AllowedRoutes) != 1 || fake.options.AllowedRoutes[0] != "/business/products" {
		t.Fatalf("documentation search was not access-filtered: %+v", fake.options.AllowedRoutes)
	}
}

func TestFeatureSearchPreservesMenuWhenDocumentationFails(t *testing.T) {
	features := []AccessibleFeature{{Name: "Products", Route: "/business/products", Description: "Crear productos"}}
	result := SearchDocumentationNavigation(context.Background(), FeatureSearchRequest{
		Query: "crear producto", Operation: OperationCreate,
	}, features, &fakeDocumentationSearcher{err: errors.New("offline")})
	if result.Status != DiscoveryStatusOK || result.Diagnostics.DocumentationStatus != DiscoveryStatusUnavailable || len(result.Routes) != 1 {
		t.Fatalf("menu fallback failed: %+v", result)
	}
}

func TestFeatureSearchNeverReturnsInaccessibleDocumentationRoute(t *testing.T) {
	features := []AccessibleFeature{{Name: "Products", Route: "/business/products", Description: "Crear productos"}}
	fake := &fakeDocumentationSearcher{results: []knowledge.DocumentationResult{{Route: "/finance/private", Content: "private", Score: 1}}}
	result := SearchDocumentationNavigation(context.Background(), FeatureSearchRequest{Query: "crear producto"}, features, fake)
	if len(result.Passages) != 0 || len(result.Routes) != 1 || result.Routes[0].Route != "/business/products" {
		t.Fatalf("inaccessible documentation leaked into discovery: %+v", result)
	}
}

func TestFeatureSearchNeverQueriesDocumentationWithoutAccessibleRoutes(t *testing.T) {
	fake := &fakeDocumentationSearcher{results: []knowledge.DocumentationResult{{Route: "/private", Content: "private"}}}
	result := SearchDocumentationNavigation(context.Background(), FeatureSearchRequest{Query: "ventas"}, nil, fake)
	if fake.options.AllowedRoutes != nil || len(result.Routes) != 0 || len(result.Passages) != 0 {
		t.Fatalf("documentation search ran without an access filter: options=%+v result=%+v", fake.options, result)
	}
}
