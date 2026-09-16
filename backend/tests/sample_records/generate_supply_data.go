package sample_records

import (
	"app/core"
	"app/crm"
	crmTypes "app/crm/types"
	"app/db"
	"app/logistics"
	logisticsTypes "app/logistics/types"
	production "app/production/types"
	_ "embed"
	"encoding/json"
	mrand "math/rand"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	supplyCompanyID int32 = 1
	supplyUserID    int32 = 1
	// One request per batch, under the handler's own 1000-record cap.
	supplyPostBatchSize = 500
)

//go:embed supply_providers.json
var supplyProviderSeedJSON []byte

// supplyDataConfig is the full argument surface; every field has a default so the script runs with
// no arguments at all.
type supplyDataConfig struct {
	productsCount        int
	providersPerProduct  int
	minStockMin          int
	minStockMax          int
	salesPerDayMin       int
	salesPerDayMax       int
	supplyMaterialsCount int
	assetsCount          int
	dryRun               bool
}

func makeSupplyDataDefaults() supplyDataConfig {
	return supplyDataConfig{
		productsCount:        2000,
		providersPerProduct:  2,
		minStockMin:          2,
		minStockMax:          50,
		salesPerDayMin:       0,
		salesPerDayMax:       20,
		supplyMaterialsCount: 100,
		assetsCount:          100,
	}
}

type supplyDataGenerator struct {
	random    *mrand.Rand
	config    supplyDataConfig
	userToken core.UsuarioToken

	providerIDs []int32
	products    []production.Product
}

// GenerateSupplyData is the entry point behind `fn-generate-supply-data`. Unlike
// generate_erp_history it never moves the clock: a supply configuration is current state, not
// history, so it belongs at the real timestamp. It is also idempotent — product_supply is keyed by
// ProductID and written through db.Merge — so rerunning overwrites the same rows instead of
// duplicating them, and no resume file is needed.
func GenerateSupplyData(args *core.ExecArgs) core.FuncResponse {
	config, configError := parseSupplyDataArgs(args.Message)
	if configError != nil {
		return args.MakeErr("Argumentos inválidos:", configError)
	}

	generator := supplyDataGenerator{
		random: mrand.New(mrand.NewSource(time.Now().UnixNano())),
		config: config,
		userToken: core.UsuarioToken{
			CompanyID: supplyCompanyID,
			ID:        supplyUserID,
		},
	}

	core.Log("GenerateSupplyData:: config", "products", config.productsCount,
		"providersPerProduct", config.providersPerProduct, "dryRun", config.dryRun)

	if err := generator.seedProviders(); err != nil {
		return args.MakeErr("No se pudieron sembrar los proveedores:", err)
	}
	if err := generator.loadProducts(); err != nil {
		return args.MakeErr("No se pudieron cargar los productos:", err)
	}

	supplyRecords := generator.makeSupplyRecords()

	if config.dryRun {
		previewCount := minInt(5, len(supplyRecords))
		previewBytes, _ := json.MarshalIndent(supplyRecords[:previewCount], "", "  ")
		core.Log("GenerateSupplyData:: dry-run preview\n" + string(previewBytes))
		return core.FuncResponse{
			Message: "Dry run: no se escribió nada.",
			Content: map[string]any{
				"providers":     len(generator.providerIDs),
				"supplyRecords": len(supplyRecords),
				"preview":       supplyRecords[:previewCount],
			},
		}
	}

	if err := generator.postSupplyBatches(supplyRecords); err != nil {
		return args.MakeErr("No se pudo guardar el abastecimiento:", err)
	}

	// Supplies come before assets: an asset is acquired from a depreciable supply, so the catalog
	// has to exist before anything can point at it.
	if generator.config.supplyMaterialsCount > 0 {
		if err := generator.seedSupplyMaterials(); err != nil {
			return args.MakeErr("No se pudieron generar los insumos:", err)
		}
	}
	if generator.config.assetsCount > 0 {
		if err := generator.seedAssets(); err != nil {
			return args.MakeErr("No se pudieron generar los activos:", err)
		}
	}

	supplyMaterialsTotal, _ := generator.countSupplyMaterials()
	assetsTotal, _ := generator.countAssets()

	core.Log("GenerateSupplyData:: terminado", "proveedores", len(generator.providerIDs),
		"productos", len(supplyRecords), "insumos", supplyMaterialsTotal, "activos", assetsTotal)

	return core.FuncResponse{
		Message: "Datos de abastecimiento generados correctamente.",
		Content: map[string]any{
			"providers":       len(generator.providerIDs),
			"supplyRecords":   len(supplyRecords),
			"supplyMaterials": supplyMaterialsTotal,
			"assets":          assetsTotal,
		},
	}
}

func parseSupplyDataArgs(message string) (supplyDataConfig, error) {
	config := makeSupplyDataDefaults()

	for _, argument := range strings.Fields(message) {
		name, rawValue, hasValue := strings.Cut(strings.TrimPrefix(argument, "--"), "=")

		if name == "dry-run" {
			config.dryRun = true
			continue
		}
		if !hasValue {
			return config, core.Err("el argumento", argument, "necesita un valor")
		}

		parsedValue, parseError := strconv.Atoi(rawValue)
		if parseError != nil {
			return config, core.Err("el argumento", argument, "no es un número")
		}
		switch name {
		case "products":
			config.productsCount = parsedValue
		case "providers-per-product":
			config.providersPerProduct = parsedValue
		case "min-stock-min":
			config.minStockMin = parsedValue
		case "min-stock-max":
			config.minStockMax = parsedValue
		case "sales-per-day-min":
			config.salesPerDayMin = parsedValue
		case "sales-per-day-max":
			config.salesPerDayMax = parsedValue
		case "supply-materials":
			config.supplyMaterialsCount = parsedValue
		case "assets":
			config.assetsCount = parsedValue
		default:
			return config, core.Err("argumento desconocido:", argument)
		}
	}

	// Fail on inconsistent combinations up front rather than thousands of records in.
	switch {
	case config.productsCount < 1:
		return config, core.Err("--products debe ser >= 1")
	case config.providersPerProduct < 1:
		return config, core.Err("--providers-per-product debe ser >= 1")
	case config.minStockMin < 0 || config.minStockMax < config.minStockMin:
		return config, core.Err("--min-stock-min/--min-stock-max forman un rango inválido")
	case config.salesPerDayMin < 0 || config.salesPerDayMax < config.salesPerDayMin:
		return config, core.Err("--sales-per-day-min/--sales-per-day-max forman un rango inválido")
	case config.supplyMaterialsCount < 0:
		return config, core.Err("--supply-materials no puede ser negativo")
	case config.assetsCount < 0:
		return config, core.Err("--assets no puede ser negativo")
	}

	return config, nil
}

// seedProviders writes the 100 static providers and keeps the IDs the handler resolved. The save
// deduplicates on RegistryNumber, so a second run matches the existing rows instead of creating a
// second set, and the response already carries their IDs.
func (generator *supplyDataGenerator) seedProviders() error {
	providerSeeds := []partySeed{}
	if err := json.Unmarshal(supplyProviderSeedJSON, &providerSeeds); err != nil {
		return core.Err("error al leer el JSON de proveedores:", err)
	}
	if len(providerSeeds) == 0 {
		return core.Err("el JSON de proveedores está vacío")
	}

	// A dry run must not write anything at all, so it builds its preview from the providers that
	// already exist instead of seeding new ones.
	if generator.config.dryRun {
		return generator.loadExistingProviders()
	}

	payload := make([]crmTypes.ClientProvider, 0, len(providerSeeds))
	for seedIndex, seed := range providerSeeds {
		if strings.TrimSpace(seed.Name) == "" {
			return core.Err("el proveedor en posición", seedIndex, "no tiene Name")
		}
		payload = append(payload, crmTypes.ClientProvider{
			Type:           crmTypes.ClientProviderTypeProvider,
			Name:           seed.Name,
			PersonType:     seed.PersonType,
			RegistryNumber: strings.TrimSpace(seed.RegistryNumber),
			CountryID:      seed.CountryID,
			CityID:         seed.CityID,
		})
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request := generator.makeRequest("POST.client-provider", nil, string(bodyBytes))
	savedProviders := []crmTypes.ClientProvider{}
	if err := decodeResponse(crm.PostClientProviders(&request), &savedProviders); err != nil {
		return err
	}

	generator.providerIDs = make([]int32, 0, len(savedProviders))
	for _, savedProvider := range savedProviders {
		if savedProvider.ID > 0 {
			generator.providerIDs = append(generator.providerIDs, savedProvider.ID)
		}
	}
	if len(generator.providerIDs) < generator.config.providersPerProduct {
		return core.Err("se resolvieron sólo", len(generator.providerIDs), "proveedores")
	}

	core.Log("GenerateSupplyData:: proveedores listos", "total", len(generator.providerIDs))
	return nil
}

// loadExistingProviders backs the dry-run preview with real provider IDs already in the company.
func (generator *supplyDataGenerator) loadExistingProviders() error {
	request := generator.makeRequest("GET.client-provider", map[string]string{
		"type": strconv.Itoa(int(crmTypes.ClientProviderTypeProvider)),
		"upv":  "0",
	}, "")

	storedProviders := []crmTypes.ClientProvider{}
	if err := decodeResponse(crm.GetClientProviders(&request), &storedProviders); err != nil {
		return err
	}

	generator.providerIDs = []int32{}
	for _, storedProvider := range storedProviders {
		if storedProvider.ID > 0 && storedProvider.Status != 0 && storedProvider.Type == crmTypes.ClientProviderTypeProvider {
			generator.providerIDs = append(generator.providerIDs, storedProvider.ID)
		}
	}
	if len(generator.providerIDs) < generator.config.providersPerProduct {
		return core.Err("el dry-run necesita al menos", generator.config.providersPerProduct,
			"proveedores ya existentes; corre el script sin --dry-run para sembrarlos")
	}

	core.Log("GenerateSupplyData:: dry-run con proveedores existentes", "total", len(generator.providerIDs))
	return nil
}

// loadProducts takes the first N active products by ID, so a rerun configures the same products
// and the page shows a stable block of configured rows.
func (generator *supplyDataGenerator) loadProducts() error {
	products := []production.Product{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.ID, productQuery.Name, productQuery.Price,
		productQuery.FinalPrice, productQuery.Status).
		CompanyID.Equals(supplyCompanyID).Status.Equals(1)

	if err := productQuery.Exec(); err != nil {
		return err
	}

	activeProducts := make([]production.Product, 0, len(products))
	for _, productRecord := range products {
		if productRecord.ID > 0 {
			activeProducts = append(activeProducts, productRecord)
		}
	}
	if len(activeProducts) == 0 {
		return core.Err("no hay productos activos en la company")
	}

	slices.SortFunc(activeProducts, func(leftProduct, rightProduct production.Product) int {
		return int(leftProduct.ID - rightProduct.ID)
	})
	if len(activeProducts) > generator.config.productsCount {
		activeProducts = activeProducts[:generator.config.productsCount]
	}

	generator.products = activeProducts
	core.Log("GenerateSupplyData:: productos cargados", "total", len(generator.products))
	return nil
}

// makeSupplyRecords draws the replenishment configuration for every selected product. Provider
// prices derive from the product's own sale price so the two quotes of a product stay in the same
// order of magnitude as what it sells for.
func (generator *supplyDataGenerator) makeSupplyRecords() []logisticsTypes.ProductSupply {
	supplyRecords := make([]logisticsTypes.ProductSupply, 0, len(generator.products))

	for _, productRecord := range generator.products {
		salePrice := productRecord.FinalPrice
		if salePrice <= 0 {
			salePrice = productRecord.Price
		}
		if salePrice <= 0 {
			salePrice = 1000
		}

		// Base cost between 55% and 75% of the sale price: the margin a distributor would leave.
		costPrice := salePrice * int32(55+generator.random.Intn(21)) / 100

		providerRows := make([]logisticsTypes.ProductSupplyProviderRow, 0, generator.config.providersPerProduct)
		for _, providerID := range generator.pickDistinctProviders(generator.config.providersPerProduct) {
			// Each provider quotes within ±10% of the base cost, so one of them is always cheaper.
			quotedPrice := costPrice * int32(90+generator.random.Intn(21)) / 100
			if quotedPrice < 1 {
				quotedPrice = 1
			}
			providerRows = append(providerRows, logisticsTypes.ProductSupplyProviderRow{
				ProviderID:   providerID,
				Capacity:     int32(50 + generator.random.Intn(451)),
				DeliveryTime: int16(1 + generator.random.Intn(15)),
				Price:        quotedPrice,
			})
		}

		supplyRecords = append(supplyRecords, logisticsTypes.ProductSupply{
			ProductID:            productRecord.ID,
			MinimunStock:         int32(generator.randomInRange(generator.config.minStockMin, generator.config.minStockMax)),
			SalesPerDayEstimated: int32(generator.randomInRange(generator.config.salesPerDayMin, generator.config.salesPerDayMax)),
			ProviderSupply:       providerRows,
		})
	}

	return supplyRecords
}

// pickDistinctProviders draws providers without repetition, because the handler rejects the same
// provider twice on one product.
func (generator *supplyDataGenerator) pickDistinctProviders(count int) []int32 {
	if count > len(generator.providerIDs) {
		count = len(generator.providerIDs)
	}

	shuffledProviderIDs := slices.Clone(generator.providerIDs)
	generator.random.Shuffle(len(shuffledProviderIDs), func(i, j int) {
		shuffledProviderIDs[i], shuffledProviderIDs[j] = shuffledProviderIDs[j], shuffledProviderIDs[i]
	})
	return shuffledProviderIDs[:count]
}

func (generator *supplyDataGenerator) postSupplyBatches(supplyRecords []logisticsTypes.ProductSupply) error {
	for batchStart := 0; batchStart < len(supplyRecords); batchStart += supplyPostBatchSize {
		batchEnd := minInt(batchStart+supplyPostBatchSize, len(supplyRecords))
		currentBatch := supplyRecords[batchStart:batchEnd]

		bodyBytes, err := json.Marshal(currentBatch)
		if err != nil {
			return err
		}

		request := generator.makeRequest("POST.product-supply", nil, string(bodyBytes))
		if response := logistics.PostProductSupply(&request); response.StatusCode != 200 {
			return core.Err(response.Error)
		}

		core.Log("GenerateSupplyData:: lote guardado", "desde", batchStart, "hasta", batchEnd)
	}
	return nil
}

func (generator *supplyDataGenerator) randomInRange(minValue int, maxValue int) int {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + generator.random.Intn(maxValue-minValue+1)
}

func (generator *supplyDataGenerator) makeRequest(route string, query map[string]string, body string) core.HandlerArgs {
	method := route
	if separatorIndex := strings.IndexByte(route, '.'); separatorIndex > 0 {
		method = route[:separatorIndex]
	}
	return core.HandlerArgs{
		Body:   &body,
		Query:  query,
		Route:  route,
		Method: method,
		User:   &generator.userToken,
	}
}

func minInt(valueA, valueB int) int {
	if valueA < valueB {
		return valueA
	}
	return valueB
}
