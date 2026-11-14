package types

import (
	config "app/config/types"
	"app/core"
	crm "app/crm/types"
	"app/db"
	sales "app/sales/types"
	"strings"
)

// ResolveSaleClientID is the client a sale names by its details (ClientInfo): an existing client with
// that identity, or a new one. It never updates an existing client from those details, which come
// from a till or a store form, to prevent corrupting a client/provider row other documents share.
func ResolveSaleClientID(clientInfo *sales.SaleOrderClientInfo, companyID int32, userID int32) (int32, error) {
	clientName := strings.TrimSpace(clientInfo.Name)
	clientRegistryNumber := strings.TrimSpace(clientInfo.RegistryNumber)
	if clientName == "" {
		return 0, core.Err("ClientInfo.Name es obligatorio.")
	}

	clientPersonType := crm.PersonTypeNatural
	if clientRegistryNumber != "" {
		// Preserve the provided registry number so identity matching can reuse existing client rows.
		clientPersonType = crm.PersonTypeCompany
	}

	clientProviders := []crm.ClientProvider{{
		Type:           crm.ClientProviderTypeClient,
		Name:           clientName,
		RegistryNumber: clientRegistryNumber,
		PersonType:     clientPersonType,
		// Validated (or derived when 0) by SaveClientProviders, which also pins it on the snapshot.
		IdentityDocType: clientInfo.IdentityDocType,
	}}
	// Under the company row's lock, like every client save, so two sales naming one new buyer
	// resolve to one client.
	saveError := db.RunTransaction(func(transaction *db.Transaction) error {
		if _, err := config.LockCompanyIn(transaction, companyID); err != nil {
			return err
		}
		saved, err := crm.SaveClientProvidersIn(transaction, clientProviders, companyID, userID, true)
		clientProviders = core.If(err == nil, saved, clientProviders)
		return err
	})
	if saveError != nil {
		return 0, saveError
	}
	if len(clientProviders) == 0 || clientProviders[0].ID <= 0 {
		return 0, core.Err("No se pudo resolver el ClientID de la venta.")
	}

	return clientProviders[0].ID, nil
}
