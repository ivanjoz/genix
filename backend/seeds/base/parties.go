package base

import (
	"app/core"
	"app/crm"
	crmTypes "app/crm/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
)

//go:embed erp_history_providers.json
var erpProviderSeedJSON []byte

//go:embed erp_history_clients.json
var erpClientSeedJSON []byte

// seedProvidersAndClients writes both parties through POST.client-provider, which deduplicates by
// name+registry hash, so re-running the script reuses the same rows instead of piling up copies.
func seedProvidersAndClients(run *runner.Run) error {
	providerSeeds := []runner.PartySeed{}
	if err := json.Unmarshal(erpProviderSeedJSON, &providerSeeds); err != nil {
		return core.Err("error al leer el JSON de proveedores:", err)
	}
	clientSeeds := []runner.PartySeed{}
	if err := json.Unmarshal(erpClientSeedJSON, &clientSeeds); err != nil {
		return core.Err("error al leer el JSON de clientes:", err)
	}

	providers, err := SaveAndLoadParties(run, providerSeeds, crmTypes.ClientProviderTypeProvider)
	if err != nil {
		return err
	}
	run.ProviderIDs = []int32{}
	for _, provider := range providers {
		run.ProviderIDs = append(run.ProviderIDs, provider.ID)
	}

	clients, err := SaveAndLoadParties(run, clientSeeds, crmTypes.ClientProviderTypeClient)
	if err != nil {
		return err
	}
	run.ClientIDs, run.BusinessClientIDs, run.DNIClientIDs = []int32{}, []int32{}, []int32{}
	for _, client := range clients {
		// An 11-digit RUC is a company buying with a factura; everyone else buys with a boleta.
		switch len(client.RegistryNumber) {
		case 11:
			run.BusinessClientIDs = append(run.BusinessClientIDs, client.ID)
		case 8:
			run.DNIClientIDs = append(run.DNIClientIDs, client.ID)
			run.ClientIDs = append(run.ClientIDs, client.ID)
		default:
			run.ClientIDs = append(run.ClientIDs, client.ID)
		}
	}

	core.Log("Seed:: terceros listos", "proveedores", len(run.ProviderIDs),
		"clientes", len(run.ClientIDs), "empresas", len(run.BusinessClientIDs))
	return nil
}

// saveAndLoadParties writes the seeds and returns the stored rows they resolved to. Only the seeds'
// own rows come back: other scripts seed parties of the same type (supply providers, expense
// suppliers), and a grocery purchase order must not land on the electricity company.
func SaveAndLoadParties(run *runner.Run, seeds []runner.PartySeed, partyType int8) ([]crmTypes.ClientProvider, error) {
	if len(seeds) == 0 {
		return nil, core.Err("la lista de terceros a sembrar está vacía")
	}

	payload := make([]crmTypes.ClientProvider, 0, len(seeds))
	for seedIndex, seed := range seeds {
		if strings.TrimSpace(seed.Name) == "" {
			return nil, core.Err("el tercero en posición", seedIndex, "no tiene Name")
		}
		payload = append(payload, crmTypes.ClientProvider{
			Type:           partyType,
			Name:           seed.Name,
			PersonType:     seed.PersonType,
			RegistryNumber: strings.TrimSpace(seed.RegistryNumber),
			CountryID:      seed.CountryID,
			CityID:         seed.CityID,
		})
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request := run.MakeRequest("POST.client-provider", nil, string(bodyBytes))
	if response := run.Send(request, crm.PostClientProviders); response.StatusCode != 200 {
		return nil, core.Err(response.Error)
	}

	// Read the IDs back instead of trusting the payload: the handler may have matched existing rows.
	listRequest := run.MakeRequest("GET.client-provider", map[string]string{
		"type": strconv.Itoa(int(partyType)),
	}, "")
	storedParties := []crmTypes.ClientProvider{}
	if err := runner.DecodeResponse(run.Send(listRequest, crm.GetClientProviders), &storedParties); err != nil {
		return nil, err
	}

	seedNames := core.SliceSet[string]{}
	for _, seed := range payload {
		seedNames.Add(seed.Name)
	}
	seededParties := []crmTypes.ClientProvider{}
	for _, party := range storedParties {
		if party.ID > 0 && party.Status != 0 && party.Type == partyType && seedNames.Include(party.Name) {
			seededParties = append(seededParties, party)
		}
	}
	if len(seededParties) == 0 {
		return nil, core.Err("no se pudo resolver ningún tercero de tipo", partyType)
	}
	return seededParties, nil
}
