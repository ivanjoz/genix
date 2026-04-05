package sample_records

import (
	"app/comercial"
	comercialTypes "app/comercial/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"app/finanzas"
	"app/logistica"
	logisticaTypes "app/logistica/types"
	negocioTypes "app/negocio/types"
	"encoding/json"
	mrand "math/rand"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	sampleCompanyID       int32 = 1
	sampleUserID          int32 = 1
	sampleWarehouseID     int32 = 1
	sampleCajaID          int32 = 1
	selectedProductsCount       = 100
	selectedSKUProducts         = 15
	historicalDaysCount         = 30
	dailyOrdersMin              = 300
	dailyOrdersMax              = 600
)

type saleOrderStatusTarget int8

const (
	saleOrderStatusGenerated saleOrderStatusTarget = iota + 1
	saleOrderStatusPaidOnly
	saleOrderStatusDeliveredOnly
	saleOrderStatusCompleted
)

type stockLedgerRecord struct {
	stockID        string
	productID      int32
	productName    string
	presentationID int16
	sku            string
	lote           string
	price          int32
	remaining      int32
}

type saleOrderGenerator struct {
	random               *mrand.Rand
	timeHelper           core.TimeHelper
	userToken            core.UsuarioToken
	productNamesByID     map[int32]string
	productPricesByID    map[int32]int32
	selectedProductIDs   []int32
	stockLedgerByProduct map[int32][]*stockLedgerRecord
	totalRemainingUnits  int32
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

	core.Log("GenerateSaleOrders:: validating context", "company", sampleCompanyID, "user", sampleUserID, "warehouse", sampleWarehouseID, "caja", sampleCajaID)
	if err := generator.validateContext(); err != nil {
		return args.MakeErr("No se pudo validar el contexto de sample records:", err)
	}

	core.Log("GenerateSaleOrders:: loading current stock")
	initialStocks, err := generator.loadWarehouseStock()
	if err != nil {
		return args.MakeErr("No se pudo cargar el stock del almacén:", err)
	}

	selectedProductIDs, err := generator.selectProductsFromStock(initialStocks)
	if err != nil {
		return args.MakeErr("No se pudieron seleccionar productos:", err)
	}
	generator.selectedProductIDs = selectedProductIDs
	core.Log("GenerateSaleOrders:: selected products", len(generator.selectedProductIDs))

	if err := generator.loadProductCatalog(); err != nil {
		return args.MakeErr("No se pudo cargar el catálogo de productos:", err)
	}

	core.Log("GenerateSaleOrders:: seeding base stock")
	if err := generator.seedBaseStock(); err != nil {
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
	dailyCounts, statusPlan := generator.makeGenerationPlan(len(historicalDates))
	core.Log("GenerateSaleOrders:: generation plan", "days", len(historicalDates), "orders", len(statusPlan), "remainingUnits", generator.totalRemainingUnits)
	if generator.totalRemainingUnits < int32(len(statusPlan)) {
		return args.MakeErr("El stock total no alcanza para garantizar al menos un item por orden.")
	}

	createdOrders := 0
	for dayIndex, historicalDate := range historicalDates {
		dailyOrdersCount := dailyCounts[dayIndex]
		core.Log("GenerateSaleOrders:: creating day batch", "dayIndex", dayIndex+1, "fecha", historicalDate, "orders", dailyOrdersCount, "remainingUnitsBefore", generator.totalRemainingUnits)

		for orderIndex := 0; orderIndex < dailyOrdersCount; orderIndex++ {
			status := statusPlan[createdOrders]
			historicalUnix := generator.makeHistoricalUnix(historicalDate, orderIndex)
			ordersRemaining := len(statusPlan) - createdOrders

			salePayload, err := generator.makeSalePayload(status, ordersRemaining)
			if err != nil {
				return args.MakeErr("No se pudo construir la orden de venta:", err)
			}

			if _, err := generator.createSaleOrder(historicalUnix, salePayload); err != nil {
				return args.MakeErr("No se pudo registrar la orden de venta:", err)
			}

			createdOrders++
			if createdOrders%250 == 0 {
				core.Log("GenerateSaleOrders:: progress", "createdOrders", createdOrders, "remainingUnits", generator.totalRemainingUnits)
			}
		}
	}

	core.Log("GenerateSaleOrders:: completed", "createdOrders", createdOrders, "remainingUnits", generator.totalRemainingUnits)
	return core.FuncResponse{
		Message: "Órdenes de venta sample generadas correctamente.",
		Content: map[string]any{
			"companyID":      sampleCompanyID,
			"userID":         sampleUserID,
			"warehouseID":    sampleWarehouseID,
			"cajaID":         sampleCajaID,
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

	if _, err := finanzas.GetCaja(sampleCompanyID, sampleCajaID); err != nil {
		return core.Err("error al consultar la caja sample:", err)
	}
	return nil
}

// loadWarehouseStock uses the existing stock handler because the requested flow must start there.
func (generator *saleOrderGenerator) loadWarehouseStock() ([]logisticaTypes.ProductStock, error) {
	query := map[string]string{
		"almacen-id": strconv.Itoa(int(sampleWarehouseID)),
		"updated":    "0",
	}
	request := generator.makeRequest("GET.productos-stock", query, "", 0)
	response := logistica.GetProductosStock(&request)
	if response.StatusCode != 200 {
		return nil, core.Err(response.Error)
	}
	stocks := []logisticaTypes.ProductStock{}
	if err := json.Unmarshal(*response.Body, &stocks); err != nil {
		return nil, err
	}
	return stocks, nil
}

// selectProductsFromStock keeps the selection tied to real warehouse stock instead of arbitrary products.
func (generator *saleOrderGenerator) selectProductsFromStock(stocks []logisticaTypes.ProductStock) ([]int32, error) {
	productIDs := []int32{}
	seenProducts := map[int32]bool{}
	for _, stock := range stocks {
		if stock.ProductID <= 0 || seenProducts[stock.ProductID] {
			continue
		}
		seenProducts[stock.ProductID] = true
		productIDs = append(productIDs, stock.ProductID)
	}
	if len(productIDs) < selectedProductsCount {
		return nil, core.Err("se encontraron menos de 100 productos con stock en el almacén")
	}
	generator.random.Shuffle(len(productIDs), func(i, j int) {
		productIDs[i], productIDs[j] = productIDs[j], productIDs[i]
	})
	return slices.Clone(productIDs[:selectedProductsCount]), nil
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

// seedBaseStock resets the chosen product stock to a controlled sample distribution before generating orders.
func (generator *saleOrderGenerator) seedBaseStock() error {
	stockPayload := []logisticaTypes.ProductStock{}
	productIDsWithSKU := slices.Clone(generator.selectedProductIDs)
	generator.random.Shuffle(len(productIDsWithSKU), func(i, j int) {
		productIDsWithSKU[i], productIDsWithSKU[j] = productIDsWithSKU[j], productIDsWithSKU[i]
	})
	productIDsWithSKU = productIDsWithSKU[:selectedSKUProducts]

	for _, productID := range generator.selectedProductIDs {
		stockPayload = append(stockPayload, logisticaTypes.ProductStock{
			WarehouseID: sampleWarehouseID,
			ProductID:   productID,
			Quantity:    generator.randomInt(100, 500),
		})
	}

	for _, productID := range productIDsWithSKU {
		stockPayload = append(stockPayload, logisticaTypes.ProductStock{
			WarehouseID: sampleWarehouseID,
			ProductID:   productID,
			SKU:         generator.randomAlphaNumeric(10),
			Quantity:    generator.randomInt(1, 10),
		})
	}

	bodyBytes, err := json.Marshal(stockPayload)
	if err != nil {
		return err
	}
	request := generator.makeRequest("POST.productos-stock", nil, string(bodyBytes), 0)
	response := logistica.PostAlmacenStock(&request)
	if response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	return nil
}

// rebuildLedger keeps an in-memory reservation view so total generated demand never exceeds seeded stock.
func (generator *saleOrderGenerator) rebuildLedger(stocks []logisticaTypes.ProductStock) error {
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
			sku:            stock.SKU,
			lote:           stock.Lote,
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

// makeHistoricalDates returns the last 30 local unixday dates including today.
func (generator *saleOrderGenerator) makeHistoricalDates() []int16 {
	currentUnixDay := generator.timeHelper.GetFechaUnix()
	historicalDates := make([]int16, 0, historicalDaysCount)
	for dayOffset := historicalDaysCount - 1; dayOffset >= 0; dayOffset-- {
		historicalDates = append(historicalDates, currentUnixDay-int16(dayOffset))
	}
	return historicalDates
}

// makeGenerationPlan precomputes daily volume and exact status counts before creating any order.
func (generator *saleOrderGenerator) makeGenerationPlan(daysCount int) ([]int, []saleOrderStatusTarget) {
	dailyCounts := make([]int, 0, daysCount)
	totalOrders := 0
	for dayIndex := 0; dayIndex < daysCount; dayIndex++ {
		dailyOrdersCount := int(generator.randomInt(dailyOrdersMin, dailyOrdersMax))
		dailyCounts = append(dailyCounts, dailyOrdersCount)
		totalOrders += dailyOrdersCount
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
	return dailyCounts, statusPlan
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
func (generator *saleOrderGenerator) makeSalePayload(status saleOrderStatusTarget, ordersRemaining int) (comercialTypes.SaleOrder, error) {
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
		LastPaymentCajaID:          sampleCajaID,
		DetailProductsIDs:          make([]int32, 0, len(selectedStocks)),
		DetailPrices:               make([]int32, 0, len(selectedStocks)),
		DetailQuantities:           make([]int32, 0, len(selectedStocks)),
		DetailProductSkus:          make([]string, 0, len(selectedStocks)),
		DetailProductLots:          make([]string, 0, len(selectedStocks)),
		DetailProductPresentations: make([]int16, 0, len(selectedStocks)),
	}

	for stockIndex, selectedStock := range selectedStocks {
		lineQuantity := quantities[stockIndex]
		payload.DetailProductsIDs = append(payload.DetailProductsIDs, selectedStock.productID)
		payload.DetailPrices = append(payload.DetailPrices, selectedStock.price)
		payload.DetailQuantities = append(payload.DetailQuantities, lineQuantity)
		payload.DetailProductSkus = append(payload.DetailProductSkus, selectedStock.sku)
		payload.DetailProductLots = append(payload.DetailProductLots, selectedStock.lote)
		payload.DetailProductPresentations = append(payload.DetailProductPresentations, selectedStock.presentationID)
		totalAmount += selectedStock.price * lineQuantity
	}
	payload.TotalAmount = totalAmount

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

// randomAlphaNumeric generates the requested 10-character SKU codes.
func (generator *saleOrderGenerator) randomAlphaNumeric(length int) string {
	const alphaNumericChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buffer := make([]byte, length)
	for index := range buffer {
		buffer[index] = alphaNumericChars[generator.random.Intn(len(alphaNumericChars))]
	}
	return string(buffer)
}

func minInt32(valueA, valueB int32) int32 {
	if valueA < valueB {
		return valueA
	}
	return valueB
}
