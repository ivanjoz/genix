package sample_records

import (
	"app/business"
	businessTypes "app/business/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/finance"
	financeTypes "app/finance/types"
	"app/logistics"
	logisticsTypes "app/logistics/types"
	"app/sales"
	salesTypes "app/sales/types"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/ivanjoz/minijson"
	mrand "math/rand"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// GenerateErpHistory replays N days of real ERP operation — purchase orders, supplier payments,
// goods reception (free / lot-tracked / serial-tracked) and sales — writing every record dated on
// the simulated day instead of today.
//
// The backdating is not simulated at the record level: core.SetHistoricalUnix freezes the whole
// process clock on the simulated instant, so the business handlers, the stock ledger, the cash
// register and the ORM's own created/updated columns all agree on the date. Every write goes
// through the production handler with the exact JSON payload its route expects.

const (
	erpCompanyID        int32 = 1
	erpUserID           int32 = 1
	erpStateFileName          = "genix_erp_history_state.json"
	erpStateVersion           = 1
	erpLotCodePrefix          = "L"
	erpSerialPrefix           = "SN"
	erpCashTopUpPadding int32 = 100000 // extra cents injected on top of the shortfall, so one
	// injection covers several orders instead of one per purchase.
)

//go:embed erp_history_providers.json
var erpProviderSeedJSON []byte

// erpPartySeed is the seed shape for both parties. CountryID only matters for providers, which
// SaveClientProviders rejects without one.
type erpPartySeed struct {
	Name           string `json:"name"`
	PersonType     int8   `json:"personType"`
	RegistryNumber string `json:"registryNumber"`
	CountryID      int16  `json:"countryId"`
	CityID         string `json:"cityId"`
}

// erpHistoryConfig is the full argument surface; every field has a default so the script runs
// with no arguments at all.
type erpHistoryConfig struct {
	days                  int
	productPoolSize       int
	purchaseOrderProducts int
	lotProducts           int
	serialProducts        int
	salesMin              int
	salesMax              int
	saleLinesMin          int
	saleLinesMax          int
	unpaidMin             int
	unpaidMax             int
	undeliveredMin        int
	undeliveredMax        int
	warehouseIDs          []int32
	reset                 bool
	dryRun                bool
}

func makeErpHistoryDefaults() erpHistoryConfig {
	return erpHistoryConfig{
		days:                  15,
		productPoolSize:       400,
		purchaseOrderProducts: 50,
		lotProducts:           15,
		serialProducts:        10,
		salesMin:              100,
		salesMax:              150,
		saleLinesMin:          3,
		saleLinesMax:          8,
		unpaidMin:             5,
		unpaidMax:             20,
		undeliveredMin:        5,
		undeliveredMax:        20,
		warehouseIDs:          []int32{1, 2},
	}
}

// stockBucketKind separates the three physical places a unit can live, because each one is
// validated against a different row by validateSaleStock: the free bucket against
// ProductStockV2.Quantity, the other two against their ProductStockDetail row.
type stockBucketKind int8

const (
	stockBucketFree stockBucketKind = iota
	stockBucketLot
	stockBucketSerial
)

type stockBucket struct {
	kind           stockBucketKind
	productID      int32
	presentationID int16
	lotID          int32
	serialNumber   string
	remaining      int32
}

// erpProductClass is how a product is received in one purchase order. The same product can be
// free on one day and lot-tracked on another, exactly like a real catalog.
type erpProductClass int8

const (
	erpProductFree erpProductClass = iota
	erpProductLotted
	erpProductSerialized
)

// Exported fields because the whole struct is persisted in the resume file.
type erpHistoryStats struct {
	PurchaseOrders   int
	ReceivedUnits    int32
	SalesCreated     int
	SalesUnpaid      int
	SalesUndelivered int
	CashInjections   int
	CashInjected     int32
	StockRetries     int
}

// erpHistoryState is the resume point, rewritten after every purchase order and after every
// single sale. A closed terminal or a killed process therefore loses at most one record, and the
// rerun continues from the exact day and step instead of replaying — which would duplicate the
// history, since nothing here is idempotent.
//
// The product pool travels in the state on purpose: it is drawn at random once, and the whole
// point of the generator is that the same products repeat across days. Redrawing it on resume
// would split one run into two unrelated catalogs.
type erpHistoryState struct {
	Version         int
	ConfigSignature string
	ProductPool     []int32

	// Day in progress and how far into it the run got.
	UnixDay                 int16
	CompletedPurchaseOrders int
	// The day's sales plan is drawn once and stored, so a resumed day keeps the exact
	// unpaid/undelivered counts that were promised when it started.
	SalesPlanCount        int
	SalesUnpaidFlags      []bool
	SalesUndeliveredFlags []bool
	CompletedSales        int

	Stats            erpHistoryStats
	AffectedUnixDays []int16
}

type erpHistoryGenerator struct {
	random  *mrand.Rand
	realNow time.Time
	config  erpHistoryConfig

	userToken   core.UsuarioToken
	cashBankID  int32
	providerIDs []int32
	clientIDs   []int32

	productPool        []int32
	productNameByID    map[int32]string
	salePriceByProduct map[int32]int32
	costPriceByProduct map[int32]int32

	// bucketsByWarehouse[warehouseID][productID] holds every place that product has units.
	bucketsByWarehouse map[int32]map[int32][]*stockBucket

	statePath string
	state     erpHistoryState
}

// GenerateErpHistory is the entry point behind `fn-generate-erp-history`.
func GenerateErpHistory(args *core.ExecArgs) core.FuncResponse {
	config, configError := parseErpHistoryArgs(args.Message)
	if configError != nil {
		return args.MakeErr("Argumentos inválidos:", configError)
	}

	generator := erpHistoryGenerator{
		random:  mrand.New(mrand.NewSource(time.Now().UnixNano())),
		realNow: time.Now(),
		config:  config,
		userToken: core.UsuarioToken{
			CompanyID: erpCompanyID,
			ID:        erpUserID,
		},
		productNameByID:    map[int32]string{},
		salePriceByProduct: map[int32]int32{},
		costPriceByProduct: map[int32]int32{},
		bucketsByWarehouse: map[int32]map[int32][]*stockBucket{},
	}

	// The clock is process-wide: leaving it frozen would date every later write in this process,
	// including anything the deferred cron goroutines do.
	defer core.SetHistoricalUnix(0)

	core.Log("GenerateErpHistory:: config", "days", config.days, "products", config.productPoolSize,
		"warehouses", config.warehouseIDs, "dryRun", config.dryRun)

	// Loaded before anything else so an interrupted run restores its product pool instead of
	// drawing a new one.
	if err := generator.loadState(); err != nil {
		return args.MakeErr("No se pudo cargar el estado de la corrida:", err)
	}
	if err := generator.validateContext(); err != nil {
		return args.MakeErr("No se pudo validar el contexto:", err)
	}
	if err := generator.seedProvidersAndClients(); err != nil {
		return args.MakeErr("No se pudieron sembrar proveedores/clientes:", err)
	}
	if err := generator.loadProductPool(); err != nil {
		return args.MakeErr("No se pudo armar el pool de productos:", err)
	}
	for _, warehouseID := range config.warehouseIDs {
		if err := generator.reloadWarehouseLedger(warehouseID); err != nil {
			return args.MakeErr("No se pudo cargar el stock del almacén", warehouseID, ":", err)
		}
	}

	if config.dryRun {
		return generator.previewPayloads(args)
	}

	historicalDates, err := generator.resolvePendingDates()
	if err != nil {
		return args.MakeErr("No se pudo resolver el rango de días:", err)
	}

	for _, targetUnixDay := range historicalDates {
		if err := generator.startDay(targetUnixDay); err != nil {
			return args.MakeErr("No se pudo guardar el estado:", err)
		}
		if err := generator.runOneDay(targetUnixDay); err != nil {
			return args.MakeErr("Error en el día", targetUnixDay, ":", err)
		}
	}

	// The state file only exists to resume; a finished run must not leave one behind or the next
	// invocation would think it is resuming.
	if err := os.Remove(generator.statePath); err != nil && !os.IsNotExist(err) {
		return args.MakeErr("No se pudo limpiar el estado:", err)
	}

	core.Log("GenerateErpHistory:: terminado", "OCs", generator.state.Stats.PurchaseOrders,
		"ventas", generator.state.Stats.SalesCreated, "unidades recibidas", generator.state.Stats.ReceivedUnits)

	return core.FuncResponse{
		Message: "Historial ERP generado correctamente.",
		Content: map[string]any{
			"days":             len(historicalDates),
			"purchaseOrders":   generator.state.Stats.PurchaseOrders,
			"receivedUnits":    generator.state.Stats.ReceivedUnits,
			"sales":            generator.state.Stats.SalesCreated,
			"salesUnpaid":      generator.state.Stats.SalesUnpaid,
			"salesUndelivered": generator.state.Stats.SalesUndelivered,
			"cashInjections":   generator.state.Stats.CashInjections,
			"cashInjected":     generator.state.Stats.CashInjected,
			"stockRetries":     generator.state.Stats.StockRetries,
			// Sale summaries are rebuilt by cron action 2, which does not run in this process.
			"unixDaysToReprocess": generator.state.AffectedUnixDays,
		},
	}
}

// ─── Resume state ──────────────────────────────────────────────────────────────

// planSignature covers what the generated history looks like, and deliberately leaves out reset
// and dryRun: those pick how this invocation runs, not what it produces. Including them would
// make the resume command — the same line without --reset — look like a different plan.
func (config erpHistoryConfig) planSignature() string {
	return fmt.Sprint(config.days, config.productPoolSize, config.purchaseOrderProducts,
		config.lotProducts, config.serialProducts, config.salesMin, config.salesMax,
		config.saleLinesMin, config.saleLinesMax, config.unpaidMin, config.unpaidMax,
		config.undeliveredMin, config.undeliveredMax, config.warehouseIDs)
}

// loadState restores an interrupted run. A state written under different arguments is refused
// instead of merged: resuming a 15-day plan into a 4-day one would silently produce neither.
func (generator *erpHistoryGenerator) loadState() error {
	generator.statePath = filepath.Join(core.ProjectTmpDir(), erpStateFileName)
	configSignature := generator.config.planSignature()

	if generator.config.reset {
		if err := os.Remove(generator.statePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		generator.state = erpHistoryState{Version: erpStateVersion, ConfigSignature: configSignature}
		return nil
	}

	stateBytes, readError := os.ReadFile(generator.statePath)
	if os.IsNotExist(readError) {
		generator.state = erpHistoryState{Version: erpStateVersion, ConfigSignature: configSignature}
		return nil
	}
	if readError != nil {
		return readError
	}

	storedState := erpHistoryState{}
	if err := json.Unmarshal(stateBytes, &storedState); err != nil {
		return core.Err("el archivo de estado", generator.statePath, "está corrupto; usa --reset:", err)
	}
	if storedState.Version != erpStateVersion {
		return core.Err("el archivo de estado es de otra versión; usa --reset")
	}
	if storedState.ConfigSignature != configSignature {
		return core.Err("el estado guardado corresponde a otros argumentos; usa --reset para empezar de nuevo")
	}

	generator.state = storedState
	generator.productPool = storedState.ProductPool
	core.Log("GenerateErpHistory:: reanudando", "día", storedState.UnixDay,
		"OCs hechas", storedState.CompletedPurchaseOrders, "ventas hechas", storedState.CompletedSales,
		"de", storedState.SalesPlanCount)
	return nil
}

// saveState rewrites the resume point. The write goes to a sibling file and is renamed, because
// a process killed halfway through a plain write would leave unparseable JSON behind — and the
// state file is only useful if it survives exactly the event it exists for.
func (generator *erpHistoryGenerator) saveState() error {
	stateBytes, marshalError := json.Marshal(generator.state)
	if marshalError != nil {
		return marshalError
	}
	temporaryPath := generator.statePath + ".tmp"
	if err := os.WriteFile(temporaryPath, stateBytes, 0o644); err != nil {
		return err
	}
	return os.Rename(temporaryPath, generator.statePath)
}

// startDay resets the per-day counters, unless the run is resuming into that same day.
func (generator *erpHistoryGenerator) startDay(targetUnixDay int16) error {
	if generator.state.UnixDay == targetUnixDay {
		return nil
	}
	generator.state.UnixDay = targetUnixDay
	generator.state.CompletedPurchaseOrders = 0
	generator.state.SalesPlanCount = 0
	generator.state.SalesUnpaidFlags = nil
	generator.state.SalesUndeliveredFlags = nil
	generator.state.CompletedSales = 0
	return generator.saveState()
}

// ─── Arguments ─────────────────────────────────────────────────────────────────

func parseErpHistoryArgs(message string) (erpHistoryConfig, error) {
	config := makeErpHistoryDefaults()

	for _, argument := range strings.Fields(message) {
		name, rawValue, hasValue := strings.Cut(strings.TrimPrefix(argument, "--"), "=")

		if name == "reset" || name == "dry-run" {
			config.reset = config.reset || name == "reset"
			config.dryRun = config.dryRun || name == "dry-run"
			continue
		}
		if !hasValue {
			return config, core.Err("el argumento", argument, "necesita un valor")
		}

		if name == "warehouses" {
			warehouseIDs := []int32{}
			for _, rawWarehouseID := range strings.Split(rawValue, ",") {
				parsedWarehouseID, parseError := strconv.Atoi(strings.TrimSpace(rawWarehouseID))
				if parseError != nil || parsedWarehouseID <= 0 {
					return config, core.Err("almacén inválido:", rawWarehouseID)
				}
				warehouseIDs = append(warehouseIDs, int32(parsedWarehouseID))
			}
			config.warehouseIDs = warehouseIDs
			continue
		}

		parsedValue, parseError := strconv.Atoi(rawValue)
		if parseError != nil {
			return config, core.Err("el argumento", argument, "no es un número")
		}
		switch name {
		case "days":
			config.days = parsedValue
		case "products":
			config.productPoolSize = parsedValue
		case "po-products":
			config.purchaseOrderProducts = parsedValue
		case "lot-products":
			config.lotProducts = parsedValue
		case "serial-products":
			config.serialProducts = parsedValue
		case "sales-min":
			config.salesMin = parsedValue
		case "sales-max":
			config.salesMax = parsedValue
		case "sale-lines-min":
			config.saleLinesMin = parsedValue
		case "sale-lines-max":
			config.saleLinesMax = parsedValue
		case "unpaid-min":
			config.unpaidMin = parsedValue
		case "unpaid-max":
			config.unpaidMax = parsedValue
		case "undelivered-min":
			config.undeliveredMin = parsedValue
		case "undelivered-max":
			config.undeliveredMax = parsedValue
		default:
			return config, core.Err("argumento desconocido:", argument)
		}
	}

	// Fail on inconsistent combinations up front: a bad ratio only shows up hundreds of writes in.
	switch {
	case config.days < 1:
		return config, core.Err("--days debe ser >= 1")
	case len(config.warehouseIDs) == 0:
		return config, core.Err("--warehouses no puede quedar vacío")
	case config.purchaseOrderProducts > config.productPoolSize:
		return config, core.Err("--po-products no puede superar a --products")
	case config.lotProducts+config.serialProducts > config.purchaseOrderProducts:
		return config, core.Err("--lot-products + --serial-products no pueden superar a --po-products")
	case config.salesMin > config.salesMax || config.salesMin < 0:
		return config, core.Err("--sales-min/--sales-max inconsistentes")
	case config.saleLinesMin < 1 || config.saleLinesMin > config.saleLinesMax:
		return config, core.Err("--sale-lines-min/--sale-lines-max inconsistentes")
	case config.unpaidMin > config.unpaidMax || config.undeliveredMin > config.undeliveredMax:
		return config, core.Err("los rangos de impagas/no entregadas son inconsistentes")
	case config.unpaidMax > config.salesMin || config.undeliveredMax > config.salesMin:
		return config, core.Err("impagas/no entregadas no pueden superar al mínimo de ventas del día")
	}
	return config, nil
}

// ─── Context, seeds and catalog ────────────────────────────────────────────────

func (generator *erpHistoryGenerator) validateContext() error {
	users := []coreTypes.User{}
	userQuery := db.Query(&users)
	userQuery.Select(userQuery.ID, userQuery.Status).
		CompanyID.Equals(erpCompanyID).ID.Equals(erpUserID).Limit(1)
	if err := userQuery.Exec(); err != nil {
		return core.Err("error al consultar el usuario:", err)
	}
	if len(users) == 0 {
		return core.Err("no existe el usuario", erpUserID, "en la company", erpCompanyID)
	}

	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status).
		CompanyID.Equals(erpCompanyID).ID.In(generator.config.warehouseIDs...)
	if err := warehouseQuery.Exec(); err != nil {
		return core.Err("error al consultar los almacenes:", err)
	}
	activeWarehouseIDs := core.SliceSet[int32]{}
	for _, warehouse := range warehouses {
		if warehouse.Status == 1 {
			activeWarehouseIDs.Add(warehouse.ID)
		}
	}
	for _, warehouseID := range generator.config.warehouseIDs {
		if !slices.Contains(activeWarehouseIDs.Values, warehouseID) {
			return core.Err("el almacén", warehouseID, "no existe o está inactivo")
		}
	}

	// The cash register is resolved rather than hardcoded so the script runs on any seeded tenant.
	cashBanks := []financeTypes.CashBank{}
	cashBankQuery := db.Query(&cashBanks)
	cashBankQuery.Select(cashBankQuery.ID, cashBankQuery.Status).
		CompanyID.Equals(erpCompanyID).Status.Equals(1)
	if err := cashBankQuery.Exec(); err != nil {
		return core.Err("error al consultar las cajas:", err)
	}
	if len(cashBanks) == 0 {
		return core.Err("no hay ninguna caja activa en la company", erpCompanyID)
	}
	slices.SortFunc(cashBanks, func(leftCashBank, rightCashBank financeTypes.CashBank) int {
		return int(leftCashBank.ID - rightCashBank.ID)
	})
	generator.cashBankID = cashBanks[0].ID

	core.Log("GenerateErpHistory:: contexto validado", "caja", generator.cashBankID,
		"almacenes", generator.config.warehouseIDs)
	return nil
}

// seedProvidersAndClients writes both parties through POST.client-provider, which deduplicates by
// name+registry hash, so re-running the script reuses the same rows instead of piling up copies.
func (generator *erpHistoryGenerator) seedProvidersAndClients() error {
	providerSeeds := []erpPartySeed{}
	if err := json.Unmarshal(erpProviderSeedJSON, &providerSeeds); err != nil {
		return core.Err("error al leer el JSON de proveedores:", err)
	}
	clientSeeds := []erpPartySeed{}
	if err := json.Unmarshal(saleOrderClientsJSON, &clientSeeds); err != nil {
		return core.Err("error al leer el JSON de clientes:", err)
	}

	providerIDs, err := generator.saveAndLoadParties(providerSeeds, businessTypes.ClientProviderTypeProvider)
	if err != nil {
		return err
	}
	generator.providerIDs = providerIDs

	clientIDs, err := generator.saveAndLoadParties(clientSeeds, businessTypes.ClientProviderTypeClient)
	if err != nil {
		return err
	}
	generator.clientIDs = clientIDs

	core.Log("GenerateErpHistory:: terceros listos", "proveedores", len(generator.providerIDs),
		"clientes", len(generator.clientIDs))
	return nil
}

func (generator *erpHistoryGenerator) saveAndLoadParties(seeds []erpPartySeed, partyType int8) ([]int32, error) {
	if len(seeds) == 0 {
		return nil, core.Err("la lista de terceros a sembrar está vacía")
	}

	payload := make([]businessTypes.ClientProvider, 0, len(seeds))
	for seedIndex, seed := range seeds {
		if strings.TrimSpace(seed.Name) == "" {
			return nil, core.Err("el tercero en posición", seedIndex, "no tiene Name")
		}
		payload = append(payload, businessTypes.ClientProvider{
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
	request := generator.makeRequest("POST.client-provider", nil, string(bodyBytes))
	if response := business.PostClientProviders(&request); response.StatusCode != 200 {
		return nil, core.Err(response.Error)
	}

	// Read the IDs back instead of trusting the payload: the handler may have matched existing rows.
	listRequest := generator.makeRequest("GET.client-provider", map[string]string{
		"type": strconv.Itoa(int(partyType)),
		"upv":  "0",
	}, "")
	storedParties := []businessTypes.ClientProvider{}
	if err := decodeResponse(business.GetClientProviders(&listRequest), &storedParties); err != nil {
		return nil, err
	}

	partyIDs := []int32{}
	for _, party := range storedParties {
		if party.ID > 0 && party.Status != 0 && party.Type == partyType {
			partyIDs = append(partyIDs, party.ID)
		}
	}
	if len(partyIDs) == 0 {
		return nil, core.Err("no se pudo resolver ningún tercero de tipo", partyType)
	}
	return partyIDs, nil
}

// loadProductPool fixes the working catalog once: the same products are reused every simulated
// day, which is what makes the generated history look like a real business restocking its shelves.
func (generator *erpHistoryGenerator) loadProductPool() error {
	products := []businessTypes.Product{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.ID, productQuery.Name, productQuery.Price,
		productQuery.FinalPrice, productQuery.Status).
		CompanyID.Equals(erpCompanyID).Status.Equals(1)
	if err := productQuery.Exec(); err != nil {
		return core.Err("error al consultar los productos:", err)
	}

	eligibleProducts := make([]businessTypes.Product, 0, len(products))
	for _, product := range products {
		if product.ID > 0 {
			eligibleProducts = append(eligibleProducts, product)
		}
	}
	if len(eligibleProducts) < generator.config.productPoolSize {
		return core.Err("sólo hay", len(eligibleProducts), "productos activos y se piden",
			generator.config.productPoolSize)
	}

	// A resumed run reuses the pool stored in the state file, so the same products keep repeating
	// across days; only a fresh run draws a new one.
	selectedProducts := eligibleProducts
	if len(generator.state.ProductPool) > 0 {
		pooledProductIDs := core.SliceSet[int32]{}
		for _, productID := range generator.state.ProductPool {
			pooledProductIDs.Add(productID)
		}
		selectedProducts = []businessTypes.Product{}
		for _, product := range eligibleProducts {
			if slices.Contains(pooledProductIDs.Values, product.ID) {
				selectedProducts = append(selectedProducts, product)
			}
		}
		if len(selectedProducts) == 0 {
			return core.Err("ninguno de los productos del estado sigue activo; usa --reset")
		}
	} else {
		generator.random.Shuffle(len(selectedProducts), func(i, j int) {
			selectedProducts[i], selectedProducts[j] = selectedProducts[j], selectedProducts[i]
		})
		selectedProducts = selectedProducts[:generator.config.productPoolSize]
	}

	generator.productPool = make([]int32, 0, len(selectedProducts))
	for _, product := range selectedProducts {
		salePrice := product.FinalPrice
		if salePrice <= 0 {
			salePrice = product.Price
		}
		if salePrice <= 0 {
			salePrice = 100
		}
		// Buying below the sale price is what leaves a margin for the reports to show.
		costPrice := salePrice * int32(60+generator.random.Intn(21)) / 100
		if costPrice <= 0 {
			costPrice = 1
		}

		generator.productPool = append(generator.productPool, product.ID)
		generator.productNameByID[product.ID] = product.Name
		generator.salePriceByProduct[product.ID] = salePrice
		generator.costPriceByProduct[product.ID] = costPrice
	}

	generator.state.ProductPool = generator.productPool
	core.Log("GenerateErpHistory:: pool de productos", len(generator.productPool))
	return generator.saveState()
}

// ─── Stock ledger ──────────────────────────────────────────────────────────────

// reloadWarehouseLedger rebuilds the in-memory view of one warehouse from the same endpoint the
// frontend uses, so the generator reserves against exactly what the API reports.
func (generator *erpHistoryGenerator) reloadWarehouseLedger(warehouseID int32) error {
	request := generator.makeRequest("GET.productos-stock", map[string]string{
		"warehouse-id": strconv.Itoa(int(warehouseID)),
		"updated":      "0",
	}, "")
	stockResult := logistics.GetProductsStockResult{}
	if err := decodeResponse(logistics.GetWarehouseProductStock(&request), &stockResult); err != nil {
		return err
	}

	bucketsByProduct := map[int32][]*stockBucket{}
	presentationByStockID := map[int64]int16{}

	for _, stock := range stockResult.ProductStock {
		presentationByStockID[stock.ID] = stock.PresentationID
		// ProductStock.Quantity is only the free bucket; the lot/serial units live in the details.
		if stock.Status == 0 || stock.Quantity <= 0 {
			continue
		}
		bucketsByProduct[stock.ProductID] = append(bucketsByProduct[stock.ProductID], &stockBucket{
			kind:           stockBucketFree,
			productID:      stock.ProductID,
			presentationID: stock.PresentationID,
			remaining:      stock.Quantity,
		})
	}

	for _, detail := range stockResult.ProductStockDetail {
		if detail.Status == 0 || detail.Quantity <= 0 {
			continue
		}
		bucketKind := stockBucketLot
		if detail.SerialNumber != "" {
			bucketKind = stockBucketSerial
		}
		bucketsByProduct[detail.ProductID] = append(bucketsByProduct[detail.ProductID], &stockBucket{
			kind:           bucketKind,
			productID:      detail.ProductID,
			presentationID: presentationByStockID[detail.ProductStockID],
			lotID:          detail.LotID,
			serialNumber:   detail.SerialNumber,
			remaining:      detail.Quantity,
		})
	}

	generator.bucketsByWarehouse[warehouseID] = bucketsByProduct
	return nil
}

func (generator *erpHistoryGenerator) availableProductIDs(warehouseID int32) []int32 {
	availableProductIDs := []int32{}
	for productID, buckets := range generator.bucketsByWarehouse[warehouseID] {
		for _, bucket := range buckets {
			if bucket.remaining > 0 {
				availableProductIDs = append(availableProductIDs, productID)
				break
			}
		}
	}
	return availableProductIDs
}

func (generator *erpHistoryGenerator) pickBucket(warehouseID int32, productID int32) *stockBucket {
	candidateBuckets := []*stockBucket{}
	for _, bucket := range generator.bucketsByWarehouse[warehouseID][productID] {
		if bucket.remaining > 0 {
			candidateBuckets = append(candidateBuckets, bucket)
		}
	}
	if len(candidateBuckets) == 0 {
		return nil
	}
	return candidateBuckets[generator.random.Intn(len(candidateBuckets))]
}

// ─── Day loop ──────────────────────────────────────────────────────────────────

func (generator *erpHistoryGenerator) runOneDay(targetUnixDay int16) error {
	dayStart := generator.dayStartTime(targetUnixDay)
	core.Log("GenerateErpHistory:: día", targetUnixDay, dayStart.Format("2006-01-02"))

	for purchaseOrderIndex, warehouseID := range generator.config.warehouseIDs {
		// Already written before the interruption; replaying it would duplicate the order.
		if purchaseOrderIndex < generator.state.CompletedPurchaseOrders {
			continue
		}
		if err := generator.runPurchaseOrder(dayStart, purchaseOrderIndex, warehouseID); err != nil {
			return err
		}
		generator.state.CompletedPurchaseOrders = purchaseOrderIndex + 1
		if err := generator.saveState(); err != nil {
			return err
		}
	}

	// Reception changed the warehouses, so the ledger is re-read from the API before selling.
	for _, warehouseID := range generator.config.warehouseIDs {
		if err := generator.reloadWarehouseLedger(warehouseID); err != nil {
			return core.Err("no se pudo recargar el stock del almacén", warehouseID, ":", err)
		}
	}

	if err := generator.runDailySales(dayStart); err != nil {
		return err
	}

	if !slices.Contains(generator.state.AffectedUnixDays, targetUnixDay) {
		generator.state.AffectedUnixDays = append(generator.state.AffectedUnixDays, targetUnixDay)
	}
	return generator.saveState()
}

// runPurchaseOrder walks one order through its full lifecycle: create → confirm → pay → receive.
func (generator *erpHistoryGenerator) runPurchaseOrder(dayStart time.Time, purchaseOrderIndex int, warehouseID int32) error {
	orderSlot := time.Duration(purchaseOrderIndex) * 5 * time.Minute
	selectedProductIDs := generator.pickDistinctFromPool(generator.config.purchaseOrderProducts)
	classByProduct := generator.classifyPurchaseProducts(selectedProductIDs)

	quantityByProduct := map[int32]int32{}
	orderPayload := logisticsTypes.PurchaseOrder{
		ProviderID:  generator.providerIDs[generator.random.Intn(len(generator.providerIDs))],
		WarehouseID: warehouseID,
	}
	for _, productID := range selectedProductIDs {
		// Serialized products move in single units, so they are bought in small numbers; the rest
		// arrive in case-sized quantities that outpace one day of demand.
		quantity := int32(12 + generator.random.Intn(29))
		if classByProduct[productID] == erpProductSerialized {
			quantity = int32(2 + generator.random.Intn(3))
		}
		quantityByProduct[productID] = quantity

		orderPayload.DetailProductIDs = append(orderPayload.DetailProductIDs, productID)
		orderPayload.DetailProductQuantity = append(orderPayload.DetailProductQuantity, quantity)
		orderPayload.DetailProductPrice = append(orderPayload.DetailProductPrice, generator.costPriceByProduct[productID])
		orderPayload.TotalAmount += quantity * generator.costPriceByProduct[productID]
	}

	createdOrder, err := generator.createPurchaseOrder(dayStart.Add(8*time.Hour+orderSlot), orderPayload)
	if err != nil {
		return core.Err("error al crear la orden de compra:", err)
	}

	if err := generator.confirmPurchaseOrder(dayStart.Add(8*time.Hour+orderSlot+time.Minute), createdOrder.ID); err != nil {
		return core.Err("error al confirmar la orden de compra:", err)
	}

	if createdOrder.DebtAmount > 0 {
		if err := generator.payPurchaseOrder(dayStart, orderSlot, createdOrder.ID, createdOrder.DebtAmount); err != nil {
			return core.Err("error al pagar la orden de compra:", err)
		}
	}

	receivedUnits, err := generator.receivePurchaseOrder(
		dayStart.Add(10*time.Hour+orderSlot), createdOrder.ID, warehouseID, selectedProductIDs, classByProduct, quantityByProduct)
	if err != nil {
		return core.Err("error al recibir la orden de compra:", err)
	}

	generator.state.Stats.PurchaseOrders++
	generator.state.Stats.ReceivedUnits += receivedUnits
	core.Log("GenerateErpHistory:: OC recibida", "id", createdOrder.ID, "almacén", warehouseID,
		"total", createdOrder.TotalAmount, "unidades", receivedUnits)
	return nil
}

// classifyPurchaseProducts splits the order's products into free, lot-tracked and serial-tracked.
func (generator *erpHistoryGenerator) classifyPurchaseProducts(productIDs []int32) map[int32]erpProductClass {
	classByProduct := map[int32]erpProductClass{}
	for productIndex, productID := range productIDs {
		switch {
		case productIndex < generator.config.lotProducts:
			classByProduct[productID] = erpProductLotted
		case productIndex < generator.config.lotProducts+generator.config.serialProducts:
			classByProduct[productID] = erpProductSerialized
		default:
			classByProduct[productID] = erpProductFree
		}
	}
	return classByProduct
}

func (generator *erpHistoryGenerator) createPurchaseOrder(at time.Time, payload logisticsTypes.PurchaseOrder) (logisticsTypes.PurchaseOrder, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return logisticsTypes.PurchaseOrder{}, err
	}
	core.SetHistoricalUnix(at.Unix())
	request := generator.makeRequest("POST.purchase-orders", nil, string(bodyBytes))
	createdOrder := logisticsTypes.PurchaseOrder{}
	if err := decodeResponse(logistics.PostPurchaseOrder(&request), &createdOrder); err != nil {
		return logisticsTypes.PurchaseOrder{}, err
	}
	if createdOrder.ID == 0 {
		return logisticsTypes.PurchaseOrder{}, core.Err("la orden de compra no devolvió ID")
	}
	return createdOrder, nil
}

func (generator *erpHistoryGenerator) confirmPurchaseOrder(at time.Time, orderID int32) error {
	core.SetHistoricalUnix(at.Unix())
	request := generator.makeRequest("PUT.purchase-orders", map[string]string{
		"action": strconv.Itoa(logistics.PurchaseOrderActionConfirm),
		"id":     strconv.Itoa(int(orderID)),
	}, "")
	if response := logistics.PutPurchaseOrder(&request); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

// payPurchaseOrder tops the cash register up first when the balance cannot cover the invoice:
// ApplyCashBankMovement does not stop a negative balance, so keeping it solvent is on the caller.
func (generator *erpHistoryGenerator) payPurchaseOrder(dayStart time.Time, orderSlot time.Duration, orderID int32, amountToPay int32) error {
	currentBalance, err := generator.readCashBankBalance()
	if err != nil {
		return err
	}
	if currentBalance < amountToPay {
		injectedAmount := amountToPay - currentBalance + erpCashTopUpPadding
		if err := generator.injectCash(dayStart.Add(7*time.Hour+orderSlot), currentBalance, injectedAmount); err != nil {
			return err
		}
	}

	payloadBytes, err := json.Marshal(map[string]any{
		"CashBankID": generator.cashBankID,
		"Amount":     amountToPay,
	})
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(dayStart.Add(8*time.Hour + orderSlot + 2*time.Minute).Unix())
	request := generator.makeRequest("PUT.purchase-orders", map[string]string{
		"action": strconv.Itoa(logistics.PurchaseOrderActionPay),
		"id":     strconv.Itoa(int(orderID)),
	}, string(payloadBytes))
	if response := logistics.PutPurchaseOrder(&request); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

func (generator *erpHistoryGenerator) readCashBankBalance() (int32, error) {
	request := generator.makeRequest("GET.cash-banks", map[string]string{"upv": "0"}, "")
	// The handler answers with a map keyed "Cajas", so it decodes as a map and not as a struct.
	cashBanksByKey := map[string][]financeTypes.CashBank{}
	if err := decodeResponse(finance.GetCashBanks(&request), &cashBanksByKey); err != nil {
		return 0, err
	}
	for _, cashBank := range cashBanksByKey["Cajas"] {
		if cashBank.ID == generator.cashBankID {
			return cashBank.CurrentAmount, nil
		}
	}
	return 0, core.Err("no se encontró la caja", generator.cashBankID)
}

// injectCash registers a type 7 ("Cobro") movement. The handler rejects the write unless
// FinalAmount - Amount matches the stored balance, so it is sent pre-computed like the frontend does.
func (generator *erpHistoryGenerator) injectCash(at time.Time, currentBalance int32, amountToInject int32) error {
	payloadBytes, err := json.Marshal(financeTypes.CashBankMovement{
		CashBankID:  generator.cashBankID,
		Type:        7,
		Amount:      amountToInject,
		FinalAmount: currentBalance + amountToInject,
	})
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(at.Unix())
	request := generator.makeRequest("POST.cash-banks-movement", nil, string(payloadBytes))
	response := finance.PostCashBankMovement(&request)
	if response.StatusCode != 200 {
		return core.Err(response.Error)
	}

	generator.state.Stats.CashInjections++
	generator.state.Stats.CashInjected += amountToInject
	core.Log("GenerateErpHistory:: efectivo inyectado", "monto", amountToInject, "saldo previo", currentBalance)
	return nil
}

// receivePurchaseOrder expands the order into stock entry items. Serial-tracked products become
// one item of quantity 1 per unit, which is what creates an individually traceable unit.
func (generator *erpHistoryGenerator) receivePurchaseOrder(
	at time.Time, orderID int32, warehouseID int32, productIDs []int32,
	classByProduct map[int32]erpProductClass, quantityByProduct map[int32]int32,
) (int32, error) {
	entryItems := []logistics.PostStockAdjustItem{}
	receivedUnits := int32(0)
	lotDateToken := strconv.Itoa(int(core.TimeToFechaUnix(at)))

	for _, productID := range productIDs {
		quantity := quantityByProduct[productID]
		receivedUnits += quantity

		switch classByProduct[productID] {
		case erpProductLotted:
			// LotCode with no LotID makes the backend resolve or create the lot by
			// Hash(date, supplier, name) — dated on the simulated day thanks to the frozen clock.
			entryItems = append(entryItems, logistics.PostStockAdjustItem{
				WarehouseID: warehouseID,
				ProductID:   productID,
				Quantity:    quantity,
				LotCode:     fmt.Sprint(erpLotCodePrefix, lotDateToken, "-", productID),
			})
		case erpProductSerialized:
			for unitIndex := int32(0); unitIndex < quantity; unitIndex++ {
				entryItems = append(entryItems, logistics.PostStockAdjustItem{
					WarehouseID:  warehouseID,
					ProductID:    productID,
					Quantity:     1,
					SerialNumber: fmt.Sprint(erpSerialPrefix, "-", lotDateToken, "-", productID, "-", unitIndex+1),
				})
			}
		default:
			entryItems = append(entryItems, logistics.PostStockAdjustItem{
				WarehouseID: warehouseID,
				ProductID:   productID,
				Quantity:    quantity,
			})
		}
	}

	payloadBytes, err := json.Marshal(map[string]any{
		"PurchaseOrderID": orderID,
		"WarehouseID":     warehouseID,
		"Items":           entryItems,
	})
	if err != nil {
		return 0, err
	}

	core.SetHistoricalUnix(at.Unix())
	request := generator.makeRequest("POST.purchase-order-entry", nil, string(payloadBytes))
	if response := logistics.PostPurchaseOrderEntry(&request); response.StatusCode != 200 {
		return 0, core.Err(response.Error)
	}
	return receivedUnits, nil
}

// ─── Sales ─────────────────────────────────────────────────────────────────────

// runDailySales draws the day's volume and its payment/delivery plan up front, so the requested
// counts are exact instead of probabilistic.
func (generator *erpHistoryGenerator) runDailySales(dayStart time.Time) error {
	// Draw the plan only once per day: on a resumed day it comes back from the state file, so the
	// unpaid/undelivered counts the day started with still hold.
	if generator.state.SalesPlanCount == 0 {
		salesCount := generator.randomInRange(generator.config.salesMin, generator.config.salesMax)
		// The two flags are drawn independently, so a sale can end up both unpaid and undelivered.
		generator.state.SalesPlanCount = salesCount
		generator.state.SalesUnpaidFlags = generator.makeFlagPlan(salesCount,
			generator.randomInRange(generator.config.unpaidMin, generator.config.unpaidMax))
		generator.state.SalesUndeliveredFlags = generator.makeFlagPlan(salesCount,
			generator.randomInRange(generator.config.undeliveredMin, generator.config.undeliveredMax))
		if err := generator.saveState(); err != nil {
			return err
		}
	}

	salesCount := generator.state.SalesPlanCount
	// Spread the sales across a working day so the hourly reports have a realistic shape.
	saleInterval := (9 * time.Hour) / time.Duration(salesCount)

	for saleIndex := generator.state.CompletedSales; saleIndex < salesCount; saleIndex++ {
		isUnpaid := generator.state.SalesUnpaidFlags[saleIndex]
		isUndelivered := generator.state.SalesUndeliveredFlags[saleIndex]

		saleTime := dayStart.Add(11*time.Hour + time.Duration(saleIndex)*saleInterval)
		if err := generator.createSaleWithRetry(saleTime, isUnpaid, isUndelivered); err != nil {
			return err
		}

		generator.state.Stats.SalesCreated++
		if isUnpaid {
			generator.state.Stats.SalesUnpaid++
		}
		if isUndelivered {
			generator.state.Stats.SalesUndelivered++
		}
		// Checkpoint per sale: the resume point must never point at a sale already written.
		generator.state.CompletedSales = saleIndex + 1
		if err := generator.saveState(); err != nil {
			return err
		}
	}

	core.Log("GenerateErpHistory:: ventas del día", "total", salesCount,
		"impagas", countFlagged(generator.state.SalesUnpaidFlags),
		"no entregadas", countFlagged(generator.state.SalesUndeliveredFlags))
	return nil
}

// createSaleWithRetry rebuilds the sale from freshly reloaded stock when the handler rejects it
// for insufficient stock, which is the only failure the ledger can drift into.
func (generator *erpHistoryGenerator) createSaleWithRetry(saleTime time.Time, isUnpaid bool, isUndelivered bool) error {
	const maxStockRetries = 3

	for attempt := 0; ; attempt++ {
		warehouseID := generator.config.warehouseIDs[generator.random.Intn(len(generator.config.warehouseIDs))]
		salePayload, reservedBuckets, err := generator.makeSalePayload(warehouseID, isUnpaid, isUndelivered)
		if err != nil {
			if attempt >= maxStockRetries {
				return err
			}
			generator.state.Stats.StockRetries++
			if reloadError := generator.reloadWarehouseLedger(warehouseID); reloadError != nil {
				return reloadError
			}
			continue
		}

		createError := generator.postSaleOrder(saleTime, salePayload)
		if createError == nil {
			// Only a delivered sale leaves the warehouse, so undelivered ones must not consume the
			// reservation — the units stay available for the following sales.
			if !isUndelivered {
				for bucketIndex, bucket := range reservedBuckets {
					bucket.remaining -= salePayload.DetailQuantities[bucketIndex]
				}
			}
			return nil
		}
		if !isStockShortageError(createError) || attempt >= maxStockRetries {
			return createError
		}

		generator.state.Stats.StockRetries++
		core.Log("GenerateErpHistory:: stock insuficiente, recargando ledger", createError.Error())
		if reloadError := generator.reloadWarehouseLedger(warehouseID); reloadError != nil {
			return reloadError
		}
	}
}

// makeSalePayload picks distinct products from one warehouse and resolves each to a concrete
// bucket, because a line drawing from a lot or a serial must name it or the stock validation
// looks at the free bucket instead.
func (generator *erpHistoryGenerator) makeSalePayload(warehouseID int32, isUnpaid bool, isUndelivered bool) (salesTypes.SaleOrder, []*stockBucket, error) {
	availableProductIDs := generator.availableProductIDs(warehouseID)
	if len(availableProductIDs) < generator.config.saleLinesMin {
		return salesTypes.SaleOrder{}, nil, core.Err("el almacén", warehouseID, "no tiene productos con stock suficiente")
	}

	lineCount := generator.randomInRange(generator.config.saleLinesMin, generator.config.saleLinesMax)
	if lineCount > len(availableProductIDs) {
		lineCount = len(availableProductIDs)
	}
	generator.random.Shuffle(len(availableProductIDs), func(i, j int) {
		availableProductIDs[i], availableProductIDs[j] = availableProductIDs[j], availableProductIDs[i]
	})

	salePayload := salesTypes.SaleOrder{
		WarehouseID:       warehouseID,
		LastPaymentCajaID: generator.cashBankID,
	}
	reservedBuckets := []*stockBucket{}

	for _, productID := range availableProductIDs[:lineCount] {
		bucket := generator.pickBucket(warehouseID, productID)
		if bucket == nil {
			continue
		}

		// A serialized unit is unique, so its line can only ever be one unit.
		quantity := int32(1)
		if bucket.kind != stockBucketSerial {
			quantity = int32(1 + generator.random.Intn(5))
			if quantity > bucket.remaining {
				quantity = bucket.remaining
			}
		}

		salePrice := generator.salePriceByProduct[productID]
		if salePrice <= 0 {
			salePrice = 100
		}

		salePayload.DetailProductsIDs = append(salePayload.DetailProductsIDs, productID)
		salePayload.DetailQuantities = append(salePayload.DetailQuantities, quantity)
		salePayload.DetailPrices = append(salePayload.DetailPrices, salePrice)
		salePayload.DetailProductPresentations = append(salePayload.DetailProductPresentations, bucket.presentationID)
		salePayload.DetailProductLotIDs = append(salePayload.DetailProductLotIDs, bucket.lotID)
		salePayload.DetailProductSkus = append(salePayload.DetailProductSkus, bucket.serialNumber)
		salePayload.TotalAmount += salePrice * quantity
		reservedBuckets = append(reservedBuckets, bucket)
	}

	if len(reservedBuckets) == 0 {
		return salesTypes.SaleOrder{}, nil, core.Err("no se pudo armar ninguna línea con stock disponible")
	}

	// ActionsIncluded is what drives the sale status: 2 registers the payment in the cash
	// register, 3 moves the goods out of the warehouse.
	if !isUnpaid {
		salePayload.ActionsIncluded = append(salePayload.ActionsIncluded, 2)
	} else {
		salePayload.DebtAmount = salePayload.TotalAmount
	}
	if !isUndelivered {
		salePayload.ActionsIncluded = append(salePayload.ActionsIncluded, 3)
	}

	// Half the sales are attributed to a client and half stay anonymous, as a counter sale would.
	if generator.random.Intn(2) == 0 && len(generator.clientIDs) > 0 {
		salePayload.ClientID = generator.clientIDs[generator.random.Intn(len(generator.clientIDs))]
	}

	return salePayload, reservedBuckets, nil
}

func (generator *erpHistoryGenerator) postSaleOrder(saleTime time.Time, payload salesTypes.SaleOrder) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	core.SetHistoricalUnix(saleTime.Unix())
	request := generator.makeRequest("POST.sale-order", nil, string(bodyBytes))
	if response := sales.PostSaleOrder(&request); response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

// ─── Dates, checkpoint and helpers ─────────────────────────────────────────────

// resolvePendingDates returns the days still to process, oldest first. The day recorded in the
// state is kept in the list rather than skipped: it may be half done, and runOneDay knows how far
// it got.
func (generator *erpHistoryGenerator) resolvePendingDates() ([]int16, error) {
	currentUnixDay := core.TimeToFechaUnix(generator.realNow)
	historicalDates := make([]int16, 0, generator.config.days)
	for dayOffset := generator.config.days - 1; dayOffset >= 0; dayOffset-- {
		historicalDates = append(historicalDates, currentUnixDay-int16(dayOffset))
	}

	resumeUnixDay := generator.state.UnixDay
	if resumeUnixDay == 0 {
		return historicalDates, nil
	}

	pendingDates := []int16{}
	for _, historicalDate := range historicalDates {
		if historicalDate >= resumeUnixDay {
			pendingDates = append(pendingDates, historicalDate)
		}
	}
	if len(pendingDates) == 0 {
		return nil, core.Err("el estado guardado (día", resumeUnixDay, ") deja el rango vacío; usa --reset")
	}
	return pendingDates, nil
}

// dayStartTime is local midnight of a simulated day. It is derived from the real clock captured
// at startup, never from core.Now(), which this generator keeps moving.
func (generator *erpHistoryGenerator) dayStartTime(targetUnixDay int16) time.Time {
	dayOffset := int(targetUnixDay - core.TimeToFechaUnix(generator.realNow))
	dayInTarget := generator.realNow.AddDate(0, 0, dayOffset)
	return time.Date(dayInTarget.Year(), dayInTarget.Month(), dayInTarget.Day(), 0, 0, 0, 0, dayInTarget.Location())
}

func (generator *erpHistoryGenerator) pickDistinctFromPool(count int) []int32 {
	shuffledPool := slices.Clone(generator.productPool)
	generator.random.Shuffle(len(shuffledPool), func(i, j int) {
		shuffledPool[i], shuffledPool[j] = shuffledPool[j], shuffledPool[i]
	})
	if count > len(shuffledPool) {
		count = len(shuffledPool)
	}
	return shuffledPool[:count]
}

// makeFlagPlan marks exactly flaggedCount of totalCount positions, in random order.
func (generator *erpHistoryGenerator) makeFlagPlan(totalCount int, flaggedCount int) []bool {
	if flaggedCount > totalCount {
		flaggedCount = totalCount
	}
	plan := make([]bool, totalCount)
	for planIndex := 0; planIndex < flaggedCount; planIndex++ {
		plan[planIndex] = true
	}
	generator.random.Shuffle(len(plan), func(i, j int) {
		plan[i], plan[j] = plan[j], plan[i]
	})
	return plan
}

func countFlagged(flags []bool) int {
	flaggedCount := 0
	for _, isFlagged := range flags {
		if isFlagged {
			flaggedCount++
		}
	}
	return flaggedCount
}

func (generator *erpHistoryGenerator) randomInRange(minValue int, maxValue int) int {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + generator.random.Intn(maxValue-minValue+1)
}

// makeRequest builds the synthetic request the production handlers expect. There is no timestamp
// here: the date comes from the process clock, which is exactly what makes this backdating global.
func (generator *erpHistoryGenerator) makeRequest(route string, query map[string]string, body string) core.HandlerArgs {
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

// decodeResponse reads a handler body. Responses are encoded with minijson — the compact
// [keys, content] format the frontend decodes — so encoding/json cannot parse them.
func decodeResponse[T any](response core.HandlerResponse, target *T) error {
	if response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	if response.Body == nil {
		return core.Err("la respuesta del handler vino vacía")
	}
	return minijson.Unmarshal(*response.Body, target)
}

// isStockShortageError matches the message validateSaleStock builds when a line outruns its bucket.
func isStockShortageError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Se posee en stock")
}

// previewPayloads shows exactly what would be written for the first simulated day and returns
// without touching the database.
func (generator *erpHistoryGenerator) previewPayloads(args *core.ExecArgs) core.FuncResponse {
	targetUnixDay := core.TimeToFechaUnix(generator.realNow) - int16(generator.config.days-1)
	dayStart := generator.dayStartTime(targetUnixDay)
	warehouseID := generator.config.warehouseIDs[0]

	selectedProductIDs := generator.pickDistinctFromPool(generator.config.purchaseOrderProducts)
	classByProduct := generator.classifyPurchaseProducts(selectedProductIDs)
	orderPayload := logisticsTypes.PurchaseOrder{
		ProviderID:  generator.providerIDs[0],
		WarehouseID: warehouseID,
	}
	lottedCount, serializedCount := 0, 0
	for _, productID := range selectedProductIDs {
		quantity := int32(20)
		switch classByProduct[productID] {
		case erpProductLotted:
			lottedCount++
		case erpProductSerialized:
			serializedCount++
			quantity = 3
		}
		orderPayload.DetailProductIDs = append(orderPayload.DetailProductIDs, productID)
		orderPayload.DetailProductQuantity = append(orderPayload.DetailProductQuantity, quantity)
		orderPayload.DetailProductPrice = append(orderPayload.DetailProductPrice, generator.costPriceByProduct[productID])
		orderPayload.TotalAmount += quantity * generator.costPriceByProduct[productID]
	}

	salePayload, _, saleError := generator.makeSalePayload(warehouseID, false, false)
	if saleError != nil {
		core.Log("GenerateErpHistory:: sin stock para previsualizar una venta:", saleError.Error())
	}

	core.Log("GenerateErpHistory:: PREVIEW primer día", dayStart.Format("2006-01-02"),
		"| OC con", len(selectedProductIDs), "productos:", lottedCount, "con lote,",
		serializedCount, "con serie")
	core.Print(orderPayload)
	core.Print(salePayload)

	return core.FuncResponse{
		Message: "Dry-run: no se escribió nada.",
		Content: map[string]any{
			"firstDay":        dayStart.Format("2006-01-02"),
			"purchaseOrder":   orderPayload,
			"sampleSaleOrder": salePayload,
			"providers":       len(generator.providerIDs),
			"clients":         len(generator.clientIDs),
			"productPool":     len(generator.productPool),
		},
	}
}
