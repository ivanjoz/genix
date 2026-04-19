package sample_records

import (
	"app/comercial"
	comercialTypes "app/comercial/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	finanzasTypes "app/finanzas/types"
	"app/logistica"
	logisticaTypes "app/logistica/types"
	"app/negocio"
	negocioTypes "app/negocio/types"
	_ "embed"
	"encoding/json"
	mrand "math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	sampleCompanyID         int32 = 1
	sampleUserID            int32 = 1
	sampleWarehouseID       int32 = 1
	saleOrdersProgressPath        = "/tmp/genix_generate_sale_orders_unixday.txt"
	selectedProductsCount         = 100
	stockSeedBatchSize            = 90
	historicalDaysCount           = 30
	dailyOrdersMin                = 300
	dailyOrdersMax                = 600
)

//go:embed sale_order_clients.json
var saleOrderClientsJSON []byte

type saleOrderStatusTarget int8

const (
	saleOrderStatusGenerated saleOrderStatusTarget = iota + 1
	saleOrderStatusPaidOnly
	saleOrderStatusDeliveredOnly
	saleOrderStatusCompleted
)

type stockLedgerRecord struct {
	stockID        int64
	productID      int32
	productName    string
	presentationID int16
	price          int32
	remaining      int32
}

type saleOrderGenerator struct {
	random                *mrand.Rand
	timeHelper            core.TimeHelper
	userToken            core.UsuarioToken
	selectedCajaID       int32
	availableClientIDs   []int32
	productNamesByID     map[int32]string
	productPricesByID    map[int32]int32
	selectedProductIDs   []int32
	stockLedgerByProduct map[int32][]*stockLedgerRecord
	totalRemainingUnits  int32
}

type saleOrderClientSeed struct {
	Name           string `json:"name"`
	PersonType     int8   `json:"personType"`
	RegistryNumber string `json:"registryNumber"`
}

// GenerateSaleOrders creates stock and historical sale orders for local sample data generation.
func GenerateSaleOrders(args *core.ExecArgs) core.FuncResponse {
	generator := saleOrderGenerator{
		random: mrand.New(mrand.NewSource(time.Now().UnixNano())),
		userToken: core.UsuarioToken{
			EmpresaID: sampleCompanyID,
			ID:        sampleUserID,
		},
		productNamesByID:     map[int32]string{},
		productPricesByID:    map[int32]int32{},
		stockLedgerByProduct: map[int32][]*stockLedgerRecord{},
	}

	core.Log("GenerateSaleOrders:: validating context", "company", sampleCompanyID, "user", sampleUserID, "warehouse", sampleWarehouseID)
	if err := generator.validateContext(); err != nil {
		return args.MakeErr("No se pudo validar el contexto de sample records:", err)
	}

	core.Log("GenerateSaleOrders:: loading current stock")
	initialStocks, err := generator.loadWarehouseStock()
	if err != nil {
		return args.MakeErr("No se pudo cargar el stock del almacén:", err)
	}

	core.Log("GenerateSaleOrders:: selecting products from stock and catalog")
	selectedProductIDs, err := generator.selectProducts(initialStocks)
	if err != nil {
		return args.MakeErr("No se pudieron seleccionar productos:", err)
	}
	generator.selectedProductIDs = selectedProductIDs
	core.Log("GenerateSaleOrders:: selected products", len(generator.selectedProductIDs))

	if err := generator.loadProductCatalog(); err != nil {
		return args.MakeErr("No se pudo cargar el catálogo de productos:", err)
	}

	core.Log("GenerateSaleOrders:: saving sample clients from JSON")
	if err := generator.seedClientsFromJSON(); err != nil {
		return args.MakeErr("No se pudieron guardar los clientes sample:", err)
	}

	core.Log("GenerateSaleOrders:: loading active clients")
	if err := generator.loadAvailableClients(); err != nil {
		return args.MakeErr("No se pudieron cargar los clientes activos:", err)
	}

	core.Log("GenerateSaleOrders:: seeding base stock")
	if err := generator.seedBaseStock(initialStocks); err != nil {
		return args.MakeErr("No se pudo sembrar el stock base:", err)
	}

	core.Log("GenerateSaleOrders:: reloading stock after seed")
	seededStocks, err := generator.loadWarehouseStock()
	if err != nil {
		return args.MakeErr("No se pudo recargar el stock sembrado:", err)
	}
	if err := generator.rebuildLedger(seededStocks); err != nil {
		return args.MakeErr("No se pudo preparar el ledger de stock:", err)
	}

	historicalDates := generator.makeHistoricalDates()
	resumeUnixDay, err := generator.loadResumeUnixDay()
	if err != nil {
		return args.MakeErr("No se pudo cargar el progreso de fechas sample:", err)
	}
	historicalDates = generator.filterDatesForResume(historicalDates, resumeUnixDay)
	if len(historicalDates) == 0 {
		return args.MakeErr("No quedaron fechas por procesar para las órdenes sample.")
	}
	dailyCounts, statusPlan, clientPlan := generator.makeGenerationPlan(len(historicalDates))
	core.Log("GenerateSaleOrders:: generation plan", "days", len(historicalDates), "orders", len(statusPlan), "remainingUnits", generator.totalRemainingUnits, "resumeUnixDay", resumeUnixDay)
	if generator.totalRemainingUnits < int32(len(statusPlan)) {
		return args.MakeErr("El stock total no alcanza para garantizar al menos un item por orden.")
	}

	createdOrders := 0
	for dayIndex, historicalDate := range historicalDates {
		dailyOrdersCount := dailyCounts[dayIndex]
		if err := generator.saveResumeUnixDay(historicalDate); err != nil {
			return args.MakeErr("No se pudo guardar el progreso de fecha sample:", err)
		}
		core.Log("GenerateSaleOrders:: creating day batch", "dayIndex", dayIndex+1, "fecha", historicalDate, "orders", dailyOrdersCount, "remainingUnitsBefore", generator.totalRemainingUnits)

		for orderIndex := 0; orderIndex < dailyOrdersCount; orderIndex++ {
			status := statusPlan[createdOrders]
			historicalUnix := generator.makeHistoricalUnix(historicalDate, orderIndex)
			ordersRemaining := len(statusPlan) - createdOrders

			createdSaleOrder, err := generator.createOrderWithRetry(status, ordersRemaining, clientPlan[createdOrders], historicalUnix)
			if err != nil {
				return args.MakeErr("No se pudo registrar la orden de venta:", err)
			}
			_ = createdSaleOrder

			createdOrders++
			if createdOrders%250 == 0 {
				core.Log("GenerateSaleOrders:: progress", "createdOrders", createdOrders, "remainingUnits", generator.totalRemainingUnits)
			}
		}
	}

	if err := generator.clearResumeUnixDay(); err != nil {
		return args.MakeErr("No se pudo limpiar el progreso de fecha sample:", err)
	}
	core.Log("GenerateSaleOrders:: completed", "createdOrders", createdOrders, "remainingUnits", generator.totalRemainingUnits)
	return core.FuncResponse{
		Message: "Órdenes de venta sample generadas correctamente.",
		Content: map[string]any{
			"companyID":      sampleCompanyID,
			"userID":         sampleUserID,
			"warehouseID":    sampleWarehouseID,
			"cajaID":         generator.selectedCajaID,
			"ordersCreated":  createdOrders,
			"remainingUnits": generator.totalRemainingUnits,
		},
	}
}

// validateContext ensures the fixed sample references already exist in DB before generating data.
func (generator *saleOrderGenerator) validateContext() error {
	users := []coreTypes.Usuario{}
	userQuery := db.Query(&users)
	userQuery.Select(userQuery.ID, userQuery.Status).
		EmpresaID.Equals(sampleCompanyID).
		ID.Equals(sampleUserID).
		Limit(1)
	if err := userQuery.Exec(); err != nil {
		return core.Err("error al consultar el usuario:", err)
	}
	if len(users) == 0 {
		return core.Err("no se encontró el usuario sample")
	}

	warehouses := []negocioTypes.Almacen{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status).
		EmpresaID.Equals(sampleCompanyID).
		ID.Equals(sampleWarehouseID).
		Limit(1)
	if err := warehouseQuery.Exec(); err != nil {
		return core.Err("error al consultar el almacén:", err)
	}
	if len(warehouses) == 0 {
		return core.Err("no se encontró el almacén sample")
	}

	resolvedCajaID, err := generator.resolveActiveCajaID()
	if err != nil {
		return core.Err("error al consultar cajas activas:", err)
	}
	generator.selectedCajaID = resolvedCajaID
	core.Log("GenerateSaleOrders:: selected caja", "cajaID", generator.selectedCajaID)
	return nil
}

// resolveActiveCajaID picks the lowest active caja ID so the sample generator can run in seeded environments without assuming ID=1.
func (generator *saleOrderGenerator) resolveActiveCajaID() (int32, error) {
	activeCajas := []finanzasTypes.Caja{}
	cajaQuery := db.Query(&activeCajas)
	cajaQuery.Select(cajaQuery.ID, cajaQuery.Status).
		EmpresaID.Equals(sampleCompanyID).
		Status.Equals(1)
	if err := cajaQuery.Exec(); err != nil {
		return 0, err
	}
	if len(activeCajas) == 0 {
		return 0, core.Err("no se encontró ninguna caja activa")
	}

	slices.SortFunc(activeCajas, func(leftCaja, rightCaja finanzasTypes.Caja) int {
		switch {
		case leftCaja.ID < rightCaja.ID:
			return -1
		case leftCaja.ID > rightCaja.ID:
			return 1
		default:
			return 0
		}
	})
	return activeCajas[0].ID, nil
}

// seedClientsFromJSON keeps the client seed list in JSON so the sample catalog stays editable outside Go code.
func (generator *saleOrderGenerator) seedClientsFromJSON() error {
	clientSeeds := []saleOrderClientSeed{}
	if err := json.Unmarshal(saleOrderClientsJSON, &clientSeeds); err != nil {
		return core.Err("error al deserializar el JSON de clientes sample:", err)
	}
	if len(clientSeeds) != 50 {
		return core.Err("el JSON de clientes sample debe contener exactamente 50 registros")
	}

	clientPayload := make([]negocioTypes.ClientProvider, 0, len(clientSeeds))
	for seedIndex, clientSeed := range clientSeeds {
		if strings.TrimSpace(clientSeed.Name) == "" {
			return core.Err("el cliente sample en posición", seedIndex, "no tiene Name")
		}
		if clientSeed.PersonType != negocioTypes.PersonTypeNatural && clientSeed.PersonType != negocioTypes.PersonTypeCompany {
			return core.Err("el cliente sample en posición", seedIndex, "tiene PersonType inválido")
		}

		// Keep the JSON as the source of truth so sample seeds can mix clients with and without registry numbers.
		clientPayload = append(clientPayload, negocioTypes.ClientProvider{
			Type:           negocioTypes.ClientProviderTypeClient,
			Name:           clientSeed.Name,
			PersonType:     clientSeed.PersonType,
			RegistryNumber: strings.TrimSpace(clientSeed.RegistryNumber),
		})
	}

	bodyBytes, err := json.Marshal(clientPayload)
	if err != nil {
		return err
	}

	request := generator.makeRequest("POST.client-provider", nil, string(bodyBytes), 0)
	response := negocio.PostClientProviders(&request)
	if response.StatusCode != 200 {
		return core.Err(response.Error)
	}

	core.Log("GenerateSaleOrders:: sample clients saved", "payloadCount", len(clientPayload))
	return nil
}

// loadAvailableClients fetches active clients after POST so generated sales use persisted IDs instead of payload-local assumptions.
func (generator *saleOrderGenerator) loadAvailableClients() error {
	query := map[string]string{
		"type":    strconv.Itoa(int(negocioTypes.ClientProviderTypeClient)),
		"updated": "0",
	}
	request := generator.makeRequest("GET.client-provider", query, "", 0)
	response := negocio.GetClientProviders(&request)
	if response.StatusCode != 200 {
		return core.Err(response.Error)
	}

	clientProviders := []negocioTypes.ClientProvider{}
	if err := json.Unmarshal(*response.Body, &clientProviders); err != nil {
		return err
	}

	generator.availableClientIDs = generator.availableClientIDs[:0]
	for _, clientProvider := range clientProviders {
		if clientProvider.Type != negocioTypes.ClientProviderTypeClient || clientProvider.Status == 0 || clientProvider.ID <= 0 {
			continue
		}
		generator.availableClientIDs = append(generator.availableClientIDs, clientProvider.ID)
	}

	if len(generator.availableClientIDs) == 0 {
		return core.Err("no se encontraron clientes activos para asignar a las ventas")
	}

	core.Log("GenerateSaleOrders:: active clients loaded", "count", len(generator.availableClientIDs))
	return nil
}

// loadWarehouseStock uses the existing stock handler because the requested flow must start there after seeding.
// The generator only exercises the free bucket (no lot / no serial), so the detail and lot sections of the
// response are discarded here; ProductStockV2.Quantity is the authoritative ledger input.
func (generator *saleOrderGenerator) loadWarehouseStock() ([]logisticaTypes.ProductStockV2, error) {
	query := map[string]string{
		"almacen-id": strconv.Itoa(int(sampleWarehouseID)),
		"updated":    "0",
	}
	request := generator.makeRequest("GET.productos-stock", query, "", 0)
	response := logistica.GetProductosStock(&request)
	if response.StatusCode != 200 {
		return nil, core.Err(response.Error)
	}
	result := logistica.GetProductosStockResult{}
	if err := json.Unmarshal(*response.Body, &result); err != nil {
		return nil, err
	}
	return result.ProductStock, nil
}

// selectProducts prioritizes products that already have stock and only complements from the catalog when needed.
func (generator *saleOrderGenerator) selectProducts(stocks []logisticaTypes.ProductStockV2) ([]int32, error) {
	selectedProductIDs := []int32{}
	selectedProductSet := map[int32]bool{}

	for _, stock := range stocks {
		if stock.ProductID <= 0 || stock.Status == 0 || stock.Quantity <= 0 || selectedProductSet[stock.ProductID] {
			continue
		}
		selectedProductSet[stock.ProductID] = true
		selectedProductIDs = append(selectedProductIDs, stock.ProductID)
	}

	if len(selectedProductIDs) >= selectedProductsCount {
		generator.random.Shuffle(len(selectedProductIDs), func(i, j int) {
			selectedProductIDs[i], selectedProductIDs[j] = selectedProductIDs[j], selectedProductIDs[i]
		})
		return slices.Clone(selectedProductIDs[:selectedProductsCount]), nil
	}

	products := []negocioTypes.Producto{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.ID, productQuery.Status).
		EmpresaID.Equals(sampleCompanyID).
		Status.Equals(1)
	if err := productQuery.Exec(); err != nil {
		return nil, err
	}

	complementProductIDs := make([]int32, 0, len(products))
	for _, product := range products {
		if product.ID > 0 && !selectedProductSet[product.ID] {
			complementProductIDs = append(complementProductIDs, product.ID)
		}
	}
	missingProductsCount := selectedProductsCount - len(selectedProductIDs)
	if len(complementProductIDs) < missingProductsCount {
		return nil, core.Err("no hay suficientes productos activos para completar los 100 productos objetivo")
	}
	generator.random.Shuffle(len(complementProductIDs), func(i, j int) {
		complementProductIDs[i], complementProductIDs[j] = complementProductIDs[j], complementProductIDs[i]
	})
	selectedProductIDs = append(selectedProductIDs, complementProductIDs[:missingProductsCount]...)
	generator.random.Shuffle(len(selectedProductIDs), func(i, j int) {
		selectedProductIDs[i], selectedProductIDs[j] = selectedProductIDs[j], selectedProductIDs[i]
	})
	return selectedProductIDs, nil
}

// loadProductCatalog resolves names and prices once so every generated line uses the persisted product price.
func (generator *saleOrderGenerator) loadProductCatalog() error {
	products := []negocioTypes.Producto{}
	query := db.Query(&products)
	query.Select(query.ID, query.Nombre, query.Precio, query.PrecioFinal, query.Status).
		EmpresaID.Equals(sampleCompanyID).
		ID.In(generator.selectedProductIDs...)
	if err := query.Exec(); err != nil {
		return err
	}
	if len(products) != len(generator.selectedProductIDs) {
		return core.Err("no se pudieron resolver todos los productos seleccionados")
	}
	for _, product := range products {
		resolvedPrice := product.PrecioFinal
		if resolvedPrice <= 0 {
			resolvedPrice = product.Precio
		}
		if resolvedPrice <= 0 {
			resolvedPrice = 1
		}
		generator.productNamesByID[product.ID] = product.Nombre
		generator.productPricesByID[product.ID] = resolvedPrice
	}
	return nil
}

// seedBaseStock resets the chosen product stock on the free bucket (no lot / no serial).
// Each selected product receives a random quantity in the V2 row; detail tracking is out of scope
// for this generator now that lot/serial seeding requires separate handler shape.
func (generator *saleOrderGenerator) seedBaseStock(_ []logisticaTypes.ProductStockV2) error {
	stockPayload := make([]logistica.PostStockAdjustItem, 0, len(generator.selectedProductIDs))

	for _, productID := range generator.selectedProductIDs {
		stockPayload = append(stockPayload, logistica.PostStockAdjustItem{
			WarehouseID: sampleWarehouseID,
			ProductID:   productID,
			Quantity:    generator.randomInt(100, 500),
		})
	}

	for batchStart := 0; batchStart < len(stockPayload); batchStart += stockSeedBatchSize {
		batchEnd := minInt(batchStart+stockSeedBatchSize, len(stockPayload))
		currentBatch := stockPayload[batchStart:batchEnd]

		// Keep each stock write below Scylla clustering-key IN limits triggered by ApplyMovimientos.
		bodyBytes, err := json.Marshal(currentBatch)
		if err != nil {
			return err
		}
		request := generator.makeRequest("POST.productos-stock", nil, string(bodyBytes), 0)
		response := logistica.PostAlmacenStock(&request)
		if response.StatusCode != 200 {
			return core.Err(response.Error)
		}

		core.Log("GenerateSaleOrders:: seeded stock batch", "batchStart", batchStart, "batchEnd", batchEnd, "batchSize", len(currentBatch))
	}
	return nil
}

// rebuildLedger keeps an in-memory reservation view so total generated demand never exceeds seeded stock.
func (generator *saleOrderGenerator) rebuildLedger(stocks []logisticaTypes.ProductStockV2) error {
	generator.stockLedgerByProduct = map[int32][]*stockLedgerRecord{}
	generator.totalRemainingUnits = 0

	selectedProductsMap := map[int32]bool{}
	for _, productID := range generator.selectedProductIDs {
		selectedProductsMap[productID] = true
	}

	for _, stock := range stocks {
		if !selectedProductsMap[stock.ProductID] || stock.Status == 0 || stock.Quantity <= 0 {
			continue
		}
		record := &stockLedgerRecord{
			stockID:        stock.ID,
			productID:      stock.ProductID,
			productName:    generator.productNamesByID[stock.ProductID],
			presentationID: stock.PresentationID,
			price:          generator.productPricesByID[stock.ProductID],
			remaining:      stock.Quantity,
		}
		generator.stockLedgerByProduct[stock.ProductID] = append(generator.stockLedgerByProduct[stock.ProductID], record)
		generator.totalRemainingUnits += stock.Quantity
	}

	if len(generator.stockLedgerByProduct) == 0 {
		return core.Err("no se encontraron registros en el ledger de stock")
	}
	return nil
}

// reloadStockLedger refreshes the in-memory counters from persisted warehouse stock after a stock mismatch.
func (generator *saleOrderGenerator) reloadStockLedger() error {
	currentStocks, err := generator.loadWarehouseStock()
	if err != nil {
		return err
	}
	return generator.rebuildLedger(currentStocks)
}

// makeHistoricalDates returns the last 30 local unixday dates including today.
func (generator *saleOrderGenerator) makeHistoricalDates() []int16 {
	currentUnixDay := generator.timeHelper.GetFechaUnix()
	historicalDates := make([]int16, 0, historicalDaysCount)
	for dayOffset := historicalDaysCount - 1; dayOffset >= 0; dayOffset-- {
		historicalDates = append(historicalDates, currentUnixDay-int16(dayOffset))
	}
	return historicalDates
}

// loadResumeUnixDay reads the persisted unixday checkpoint so interrupted runs can resume from the same date.
func (generator *saleOrderGenerator) loadResumeUnixDay() (int16, error) {
	progressBytes, err := os.ReadFile(saleOrdersProgressPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	progressValue := strings.TrimSpace(string(progressBytes))
	if len(progressValue) == 0 {
		return 0, nil
	}

	resumeUnixDay, err := strconv.Atoi(progressValue)
	if err != nil {
		return 0, err
	}
	return int16(resumeUnixDay), nil
}

// saveResumeUnixDay persists the current batch date before processing it so retries restart from that unixday.
func (generator *saleOrderGenerator) saveResumeUnixDay(unixDay int16) error {
	return os.WriteFile(saleOrdersProgressPath, []byte(strconv.Itoa(int(unixDay))), 0o644)
}

// clearResumeUnixDay removes the checkpoint once the generation completes successfully.
func (generator *saleOrderGenerator) clearResumeUnixDay() error {
	err := os.Remove(saleOrdersProgressPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// filterDatesForResume keeps only dates at or after the checkpoint so reruns continue from the interrupted day.
func (generator *saleOrderGenerator) filterDatesForResume(historicalDates []int16, resumeUnixDay int16) []int16 {
	if resumeUnixDay == 0 {
		return historicalDates
	}

	pendingDates := []int16{}
	for _, historicalDate := range historicalDates {
		if historicalDate >= resumeUnixDay {
			pendingDates = append(pendingDates, historicalDate)
		}
	}
	return pendingDates
}

// makeGenerationPlan precomputes daily volume, statuses and exact client assignment before creating any order.
func (generator *saleOrderGenerator) makeGenerationPlan(daysCount int) ([]int, []saleOrderStatusTarget, []bool) {
	dailyCounts := make([]int, 0, daysCount)
	totalOrders := 0
	for dayIndex := 0; dayIndex < daysCount; dayIndex++ {
		dailyOrdersCount := int(generator.randomInt(dailyOrdersMin, dailyOrdersMax))
		dailyCounts = append(dailyCounts, dailyOrdersCount)
		totalOrders += dailyOrdersCount
	}
	if totalOrders%2 != 0 && len(dailyCounts) > 0 {
		// Force an even total so the client assignment rule can stay exactly 50/50.
		dailyCounts[len(dailyCounts)-1]--
		totalOrders--
	}

	generatedCount := totalOrders / 10
	paidOnlyCount := totalOrders / 10
	deliveredOnlyCount := totalOrders / 10
	completedCount := totalOrders - generatedCount - paidOnlyCount - deliveredOnlyCount

	statusPlan := make([]saleOrderStatusTarget, 0, totalOrders)
	for index := 0; index < generatedCount; index++ {
		statusPlan = append(statusPlan, saleOrderStatusGenerated)
	}
	for index := 0; index < paidOnlyCount; index++ {
		statusPlan = append(statusPlan, saleOrderStatusPaidOnly)
	}
	for index := 0; index < deliveredOnlyCount; index++ {
		statusPlan = append(statusPlan, saleOrderStatusDeliveredOnly)
	}
	for index := 0; index < completedCount; index++ {
		statusPlan = append(statusPlan, saleOrderStatusCompleted)
	}

	generator.random.Shuffle(len(statusPlan), func(i, j int) {
		statusPlan[i], statusPlan[j] = statusPlan[j], statusPlan[i]
	})

	// Keep the requirement exact instead of probabilistic so half the generated sales have a client and half remain anonymous.
	clientPlan := make([]bool, 0, totalOrders)
	ordersWithClient := totalOrders / 2
	for index := 0; index < ordersWithClient; index++ {
		clientPlan = append(clientPlan, true)
	}
	for index := ordersWithClient; index < totalOrders; index++ {
		clientPlan = append(clientPlan, false)
	}
	generator.random.Shuffle(len(clientPlan), func(i, j int) {
		clientPlan[i], clientPlan[j] = clientPlan[j], clientPlan[i]
	})

	return dailyCounts, statusPlan, clientPlan
}

// makeHistoricalUnix assigns each order a unique local second inside the target day to keep IDs spread out.
func (generator *saleOrderGenerator) makeHistoricalUnix(targetDate int16, orderIndex int) int64 {
	currentUnixDay := generator.timeHelper.GetFechaUnix()
	dayOffset := int(targetDate - currentUnixDay)
	localBaseTime := time.Now().AddDate(0, 0, dayOffset)
	historicalTime := time.Date(localBaseTime.Year(), localBaseTime.Month(), localBaseTime.Day(), 12, 0, 0, 0, localBaseTime.Location())
	return historicalTime.Add(time.Duration(orderIndex) * time.Second).Unix()
}

// makeSalePayload reserves stock in-memory first and then maps the selected lines to the real sale payload.
func (generator *saleOrderGenerator) makeSalePayload(status saleOrderStatusTarget, ordersRemaining int, shouldAssignClient bool) (comercialTypes.SaleOrder, error) {
	if generator.totalRemainingUnits < int32(ordersRemaining) {
		return comercialTypes.SaleOrder{}, core.Err("no queda stock suficiente para completar el plan de órdenes")
	}

	availableProductIDs := generator.availableProductIDs()
	if len(availableProductIDs) == 0 {
		return comercialTypes.SaleOrder{}, core.Err("no quedan productos con stock disponible")
	}

	lineCount := generator.chooseLineCount(len(availableProductIDs))
	maxUnitsForOrder := generator.totalRemainingUnits - int32(ordersRemaining-1)
	if maxUnitsForOrder < int32(lineCount) {
		lineCount = int(maxUnitsForOrder)
	}
	if lineCount < 1 {
		return comercialTypes.SaleOrder{}, core.Err("no se pudo asignar al menos una línea a la orden")
	}

	targetUnits := generator.chooseTargetUnits(lineCount, maxUnitsForOrder)
	selectedProductIDs := generator.pickDistinctProducts(availableProductIDs, lineCount)
	selectedStocks := make([]*stockLedgerRecord, 0, len(selectedProductIDs))
	quantities := make([]int32, 0, len(selectedProductIDs))
	remainingUnitsToAssign := targetUnits

	for lineIndex, productID := range selectedProductIDs {
		availableStocks := generator.availableStocksForProduct(productID)
		if len(availableStocks) == 0 {
			return comercialTypes.SaleOrder{}, core.Err("el producto quedó sin stock disponible durante la asignación")
		}

		selectedStock := availableStocks[generator.random.Intn(len(availableStocks))]
		linesRemainingAfterCurrent := len(selectedProductIDs) - lineIndex - 1
		maxQuantity := minInt32(5, selectedStock.remaining)
		maxQuantity = minInt32(maxQuantity, remainingUnitsToAssign-int32(linesRemainingAfterCurrent))
		if maxQuantity < 1 {
			return comercialTypes.SaleOrder{}, core.Err("no se pudo asignar cantidad válida a la línea")
		}

		quantity := generator.chooseLineQuantity(maxQuantity)
		if lineIndex == len(selectedProductIDs)-1 {
			quantity = remainingUnitsToAssign
		}
		selectedStock.remaining -= quantity
		generator.totalRemainingUnits -= quantity
		remainingUnitsToAssign -= quantity

		selectedStocks = append(selectedStocks, selectedStock)
		quantities = append(quantities, quantity)
	}

	totalAmount := int32(0)
	payload := comercialTypes.SaleOrder{
		WarehouseID:                sampleWarehouseID,
		LastPaymentCajaID:          generator.selectedCajaID,
		DetailProductsIDs:          make([]int32, 0, len(selectedStocks)),
		DetailPrices:               make([]int32, 0, len(selectedStocks)),
		DetailQuantities:           make([]int32, 0, len(selectedStocks)),
		DetailProductPresentations: make([]int16, 0, len(selectedStocks)),
	}

	for stockIndex, selectedStock := range selectedStocks {
		lineQuantity := quantities[stockIndex]
		payload.DetailProductsIDs = append(payload.DetailProductsIDs, selectedStock.productID)
		payload.DetailPrices = append(payload.DetailPrices, selectedStock.price)
		payload.DetailQuantities = append(payload.DetailQuantities, lineQuantity)
		payload.DetailProductPresentations = append(payload.DetailProductPresentations, selectedStock.presentationID)
		totalAmount += selectedStock.price * lineQuantity
	}
	payload.TotalAmount = totalAmount
	if shouldAssignClient {
		if len(generator.availableClientIDs) == 0 {
			return comercialTypes.SaleOrder{}, core.Err("no hay clientes disponibles para asignar ClientID")
		}
		payload.ClientID = generator.availableClientIDs[generator.random.Intn(len(generator.availableClientIDs))]
	}

	switch status {
	case saleOrderStatusGenerated:
		payload.DebtAmount = totalAmount
	case saleOrderStatusPaidOnly:
		payload.ActionsIncluded = []int8{2}
	case saleOrderStatusDeliveredOnly:
		payload.ActionsIncluded = []int8{3}
		payload.DebtAmount = totalAmount
	case saleOrderStatusCompleted:
		payload.ActionsIncluded = []int8{2, 3}
	default:
		return comercialTypes.SaleOrder{}, core.Err("status de orden no reconocido")
	}

	return payload, nil
}

// createSaleOrder calls the production sale handler directly so all business side effects stay intact.
func (generator *saleOrderGenerator) createSaleOrder(historicalUnix int64, payload comercialTypes.SaleOrder) (*comercialTypes.SaleOrder, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request := generator.makeRequest("POST.sale-order", nil, string(bodyBytes), historicalUnix)
	response := comercial.PostSaleOrder(&request)
	if response.StatusCode != 200 {
		return nil, core.Err(response.Error)
	}

	saleOrder := comercialTypes.SaleOrder{}
	if err := json.Unmarshal(*response.Body, &saleOrder); err != nil {
		return nil, err
	}
	return &saleOrder, nil
}

// createOrderWithRetry regenerates the order once after reloading stock if the handler reports a stock mismatch.
func (generator *saleOrderGenerator) createOrderWithRetry(status saleOrderStatusTarget, ordersRemaining int, shouldAssignClient bool, historicalUnix int64) (*comercialTypes.SaleOrder, error) {
	salePayload, err := generator.makeSalePayload(status, ordersRemaining, shouldAssignClient)
	if err != nil {
		return nil, err
	}

	createdSaleOrder, err := generator.createSaleOrder(historicalUnix, salePayload)
	if err == nil {
		return createdSaleOrder, nil
	}
	if !generator.isStockShortageError(err) {
		return nil, err
	}

	core.Log("GenerateSaleOrders:: stock mismatch detected, reloading ledger", "error", err.Error())
	if reloadErr := generator.reloadStockLedger(); reloadErr != nil {
		return nil, core.Err("no se pudo recargar el stock tras error de validación:", reloadErr)
	}

	salePayload, err = generator.makeSalePayload(status, ordersRemaining, shouldAssignClient)
	if err != nil {
		return nil, err
	}
	return generator.createSaleOrder(historicalUnix, salePayload)
}

// makeRequest centralizes the synthetic handler request so sample flows stay consistent.
func (generator *saleOrderGenerator) makeRequest(route string, query map[string]string, body string, historicalUnix int64) core.HandlerArgs {
	method := route
	if separatorIndex := strings.IndexByte(route, '.'); separatorIndex > 0 {
		method = route[:separatorIndex]
	}
	return core.HandlerArgs{
		Body:           &body,
		Query:          query,
		Route:          route,
		Method:         method,
		Usuario:        &generator.userToken,
		HistoricalUnix: historicalUnix,
	}
}

// availableProductIDs returns only products that still have at least one stock row with remaining quantity.
func (generator *saleOrderGenerator) availableProductIDs() []int32 {
	availableProductIDs := []int32{}
	for productID, stockRecords := range generator.stockLedgerByProduct {
		for _, stockRecord := range stockRecords {
			if stockRecord.remaining > 0 {
				availableProductIDs = append(availableProductIDs, productID)
				break
			}
		}
	}
	return availableProductIDs
}

// availableStocksForProduct keeps SKU and non-SKU rows eligible while preventing duplicate products per order.
func (generator *saleOrderGenerator) availableStocksForProduct(productID int32) []*stockLedgerRecord {
	availableStocks := []*stockLedgerRecord{}
	for _, stockRecord := range generator.stockLedgerByProduct[productID] {
		if stockRecord.remaining > 0 {
			availableStocks = append(availableStocks, stockRecord)
		}
	}
	return availableStocks
}

func (generator *saleOrderGenerator) isStockShortageError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Se posee en stock")
}

// pickDistinctProducts randomly chooses the products that will appear in one order.
func (generator *saleOrderGenerator) pickDistinctProducts(productIDs []int32, lineCount int) []int32 {
	shuffledProductIDs := slices.Clone(productIDs)
	generator.random.Shuffle(len(shuffledProductIDs), func(i, j int) {
		shuffledProductIDs[i], shuffledProductIDs[j] = shuffledProductIDs[j], shuffledProductIDs[i]
	})
	return shuffledProductIDs[:lineCount]
}

// chooseLineCount keeps most orders compact so the seeded stock lasts across the full 30-day plan.
func (generator *saleOrderGenerator) chooseLineCount(maxLines int) int {
	weightedLineCounts := []int{1, 1, 1, 1, 1, 2, 2, 2, 3, 3, 4, 5, 6, 7, 8}
	eligibleLineCounts := []int{}
	for _, lineCount := range weightedLineCounts {
		if lineCount <= maxLines {
			eligibleLineCounts = append(eligibleLineCounts, lineCount)
		}
	}
	if len(eligibleLineCounts) == 0 {
		return 1
	}
	return eligibleLineCounts[generator.random.Intn(len(eligibleLineCounts))]
}

// chooseTargetUnits keeps quantities low enough to cover the full sample horizon while still varying orders.
func (generator *saleOrderGenerator) chooseTargetUnits(lineCount int, maxUnitsForOrder int32) int32 {
	targetUnits := int32(lineCount)
	maxExtraUnits := int(maxUnitsForOrder) - lineCount
	if maxExtraUnits <= 0 {
		return targetUnits
	}

	weightedExtraUnits := []int{0, 0, 0, 0, 1, 1, 1, 2, 2, 3, 4}
	eligibleExtraUnits := []int{}
	for _, extraUnits := range weightedExtraUnits {
		if extraUnits <= maxExtraUnits {
			eligibleExtraUnits = append(eligibleExtraUnits, extraUnits)
		}
	}
	if len(eligibleExtraUnits) == 0 {
		return targetUnits
	}
	return targetUnits + int32(eligibleExtraUnits[generator.random.Intn(len(eligibleExtraUnits))])
}

// chooseLineQuantity keeps line quantities inside the required 1..5 range with a strong bias toward 1.
func (generator *saleOrderGenerator) chooseLineQuantity(maxQuantity int32) int32 {
	weightedQuantities := []int32{1, 1, 1, 1, 2, 2, 3, 4, 5}
	eligibleQuantities := []int32{}
	for _, quantity := range weightedQuantities {
		if quantity <= maxQuantity {
			eligibleQuantities = append(eligibleQuantities, quantity)
		}
	}
	if len(eligibleQuantities) == 0 {
		return 1
	}
	return eligibleQuantities[generator.random.Intn(len(eligibleQuantities))]
}

// randomInt wraps inclusive random ranges used by the sample generator.
func (generator *saleOrderGenerator) randomInt(minValue, maxValue int) int32 {
	return int32(minValue + generator.random.Intn(maxValue-minValue+1))
}

func minInt32(valueA, valueB int32) int32 {
	if valueA < valueB {
		return valueA
	}
	return valueB
}

func minInt(valueA, valueB int) int {
	if valueA < valueB {
		return valueA
	}
	return valueB
}

