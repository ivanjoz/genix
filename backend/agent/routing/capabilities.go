package routing

// Registry is the backend authority; classifier output cannot change availability.
var Registry = []CapabilityState{
	{Name: CapabilityDocumentationSearch, Available: true},
	{Name: CapabilityCurrentPage, Available: true},
	{Name: CapabilityMenu, Available: true},
	{Name: CapabilityBrowserAction, Available: true},
	{Name: CapabilityWebpageBuilder, Available: true},
	{Name: CapabilitySalesSearch, Available: false},
	{Name: CapabilityPurchaseSearch, Available: false},
	{Name: CapabilityInventorySearch, Available: false},
	{Name: CapabilityFinanceSearch, Available: false},
	{Name: CapabilityCustomerSearch, Available: false},
	{Name: CapabilitySupplierSearch, Available: false},
}

func KnownCapability(name CapabilityName) bool {
	for _, capability := range Registry {
		if capability.Name == name {
			return true
		}
	}
	return false
}

func CapabilityAvailable(name CapabilityName) bool {
	for _, capability := range Registry {
		if capability.Name == name {
			return capability.Available
		}
	}
	return false
}

func CapabilitySnapshot() []CapabilityState {
	return append([]CapabilityState(nil), Registry...)
}
