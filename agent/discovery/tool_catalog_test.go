package discovery

import (
	"context"
	"testing"
)

func TestEmptyToolCatalogIsSuccessful(t *testing.T) {
	result := NewToolCatalog().Search(context.Background(), ToolSearchRequest{
		Query: "ventas por producto", Domain: "sales", Operation: OperationRead, DeliveryPreference: DeliveryInline,
	})
	if result.Status != DiscoveryStatusOK || result.CatalogVersion != ToolCatalogVersion || result.Tools == nil || len(result.Tools) != 0 {
		t.Fatalf("unexpected empty catalog result: %+v", result)
	}
}
