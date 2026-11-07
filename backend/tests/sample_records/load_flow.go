// fn-load-test drives a running VPS server the way a growing set of companies does, and reports
// what it sustains.
//
// The flows are the demo generators' own (generate_supply_data.go, generate_erp_history.go,
// generate_expenses.go): the same payloads, sent over HTTP to the server instead of to the handlers
// in this process (see the generators' send). There is no history: fn-init creates empty companies,
// and each one onboards during the run (onboardLoadCompany) — staff, catalog, providers, first stock
// — before its people start working.
//
// Who works, per company:
//   - 4 cashiers (--cashiers): each sells (POST.sale-order, 3–8 lines; ~10% unpaid, ~5% undelivered,
//     ~1% annulled right after), pausing a think time between sales. Every 20 sales its page
//     reloads: the incremental delta syncs a browser that holds its cache sends, and 2% of those
//     page loads come from a cleared cache (a first sync).
//   - 1 back-office user: per 10 company sales an expense paid on the spot, and per 50 sales a
//     purchase order walked through create → confirm → supplier factura → pay → reception.
//   - 1 manager: per 10 company sales one report, in rotation.
//
// The ramp starts --companies-per-stage new companies every --stage-seconds, up to --max-companies —
// the way a server grows, by tenants that each have a few tills — and stops at the first stage
// whose sale p99 or error rate breaks the limits. --live=<file> gets a status line every 5 seconds.
//
// The database benchmarks (backend/tests/loadtest/):
//   - --settle-later=N: each unpaid sale is paid, and each undelivered one delivered, N seconds later
//     by the cashier who sold it: a status change per settlement.
//   - --status-change-percent=P: P% of sales are unpaid and, drawn apart, P% undelivered.
//   - --sales-only: the cashiers' sales and the back office that restocks them, without page syncs or
//     the manager's reports, for tuning the database (tests/loadtest/bench_sales_oceanbase.sh).
package sample_records

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	salesTypes "app/sales/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	mrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ivanjoz/colbin"
	fareward "github.com/ivanjoz/fareward/go"
	"github.com/ivanjoz/genix-orm/utils"
	"github.com/klauspost/compress/zstd"
)

// Who works in each company. The cashier logins are numbered: cajero01, cajero02, ...
const (
	loadTestBackOfficeLogin = "backoffice"
	loadTestManagerLogin    = "gerente"
	loadTestCashierPrefix   = "cajero"
	loadTestPassword        = "genix-load-test"
)

// The workload ratios, per company.
const (
	salesPerPageReload       = 20
	salesPerExpense          = 10
	salesPerPurchaseOrder    = 50
	salesPerReport           = 10
	firstSyncPercent         = 2
	unpaidSalePercent        = 10
	undeliveredSalePercent   = 5
	annulledSalePercent      = 1
	backOfficePollInterval   = 200 * time.Millisecond
	loadTestRequestTimeout   = 30 * time.Second
	loadTestCashierStartSpan = 2 * time.Second
	liveStatusInterval       = 5 * time.Second
)

// makeAllAccessGrants grants every access of the catalog at its top level, with "Todos" on the
// ones that declare sub-accesses: the shape of the bootstrap admin, but as a regular user who goes
// through the access gate.
func makeAllAccessGrants() ([]coreTypes.AccesoGrantRecord, error) {
	accessHelper := core.GetEmbeddedAccessHelper()
	allAccessIDs, err := core.GetAllEmbeddedAccesosIDs()
	if err != nil {
		return nil, core.Err("no se pudo cargar el catálogo de accesos:", err)
	}
	grants := make([]coreTypes.AccesoGrantRecord, 0, len(allAccessIDs))
	for _, accessID := range allAccessIDs {
		grant := coreTypes.AccesoGrantRecord{AccesoID: uint16(accessID), Nivel: 4}
		if accessInfo, found := accessHelper.GetAccesoInfo(accessID); found && accessInfo.SubAccesosMask != 0 {
			grant.SubAccesos = coreTypes.SubAccesoIDs{core.SubAccesoTodosID}
		}
		grants = append(grants, grant)
	}
	return grants, nil
}

// ─── fn-load-test ───────────────────────────────────────────────────────────────

type loadTestConfig struct {
	serverURL string
	// Every stage, companiesPerStage new companies (out of the maxCompanies fn-init created) start
	// onboarding; each one then runs cashiersPerCompany cashiers.
	companiesPerStage  int
	maxCompanies       int
	cashiersPerCompany int
	stageSeconds       int
	// catalogProducts is how many products a new company imports from products.xlsx.
	catalogProducts int
	// liveStatusPath gets one status line every liveStatusInterval, to follow the run with tail -f.
	liveStatusPath string
	thinkTime      time.Duration
	maxSaleP99     time.Duration
	maxErrorPct    float64
	// watchedPIDs are sampled for memory and CPU every second: name=pid.
	watchedPIDs map[string]int
	// settleDelay > 0 pays each unpaid sale and delivers each undelivered one that long after it was
	// sold (--settle-later). 0 leaves them unsettled.
	settleDelay time.Duration
	// The share of sales sold unpaid and undelivered, each drawn on its own (--status-change-percent).
	unpaidPercent      int
	undeliveredPercent int
	// salesOnly runs the cashiers' sales and the back office's restocking (--sales-only): no page
	// syncs and no manager reports.
	// The database-tuning benchmark, with --think-ms=0 to saturate it.
	salesOnly bool
}

func parseLoadTestArgs(message string) (loadTestConfig, error) {
	config := loadTestConfig{
		serverURL: "http://127.0.0.1:34010", companiesPerStage: 5, maxCompanies: 100,
		cashiersPerCompany: 4, stageSeconds: 45, catalogProducts: 2000, thinkTime: 2 * time.Second,
		maxSaleP99: time.Second, maxErrorPct: 1, watchedPIDs: map[string]int{},
		unpaidPercent: unpaidSalePercent, undeliveredPercent: undeliveredSalePercent,
	}
	for _, argument := range strings.Fields(message) {
		name, rawValue, _ := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		var err error
		switch name {
		case "server":
			config.serverURL = strings.TrimRight(rawValue, "/")
		case "cashiers":
			config.cashiersPerCompany, err = strconv.Atoi(rawValue)
		case "live":
			config.liveStatusPath = rawValue
		case "companies-per-stage":
			config.companiesPerStage, err = strconv.Atoi(rawValue)
		case "max-companies":
			config.maxCompanies, err = strconv.Atoi(rawValue)
		case "catalog":
			config.catalogProducts, err = strconv.Atoi(rawValue)
		case "stage-seconds":
			config.stageSeconds, err = strconv.Atoi(rawValue)
		case "think-ms":
			var thinkMs int
			thinkMs, err = strconv.Atoi(rawValue)
			config.thinkTime = time.Duration(thinkMs) * time.Millisecond
		case "max-p99-ms":
			var maxMs int
			maxMs, err = strconv.Atoi(rawValue)
			config.maxSaleP99 = time.Duration(maxMs) * time.Millisecond
		case "max-error-pct":
			config.maxErrorPct, err = strconv.ParseFloat(rawValue, 64)
		case "sales-only":
			config.salesOnly = true
		case "settle-later":
			var settleSeconds int
			settleSeconds, err = strconv.Atoi(rawValue)
			config.settleDelay = time.Duration(settleSeconds) * time.Second
		case "status-change-percent":
			config.unpaidPercent, err = strconv.Atoi(rawValue)
			config.undeliveredPercent = config.unpaidPercent
		case "pids":
			for _, namedPID := range strings.Split(rawValue, ",") {
				processName, rawPID, _ := strings.Cut(namedPID, "=")
				pid, parseErr := strconv.Atoi(rawPID)
				if parseErr != nil {
					return config, core.Err("--pids inválido:", namedPID)
				}
				config.watchedPIDs[processName] = pid
			}
		default:
			return config, core.Err("argumento desconocido:", argument)
		}
		if err != nil {
			return config, core.Err("argumento inválido:", argument)
		}
	}
	return config, nil
}

// loadCompany is one company under load: the prepared generator its people are cloned from, and
// the sales counter that paces the back office and the manager.
type loadCompany struct {
	template     *erpHistoryGenerator
	userIDByName map[string]int32
	sales        atomic.Int64
}

// RunLoadTest is fn-load-test.
func RunLoadTest(args *core.ExecArgs) core.FuncResponse {
	config, err := parseLoadTestArgs(args.Message)
	if err != nil {
		return args.MakeErr(err)
	}
	recorder := newLoadRecorder()
	client := &loadTestClient{
		serverURL: config.serverURL,
		recorder:  recorder,
		http: &http.Client{Timeout: loadTestRequestTimeout, Transport: &http.Transport{
			MaxIdleConns: 4096, MaxIdleConnsPerHost: 4096, IdleConnTimeout: 90 * time.Second,
		}},
	}

	// Measured from second 0: the first companies' onboarding is the first load the server sees.
	recorder.reset()
	recorder.startStage(0)
	stopEverything := make(chan struct{})
	waitGroup := sync.WaitGroup{}
	activeCompanies, onboardingCompanies := atomic.Int64{}, atomic.Int64{}
	stopSampler := startProcessSampler(config.watchedPIDs, recorder)
	stopLiveStatus := startLiveStatus(config.liveStatusPath, recorder, &activeCompanies, &onboardingCompanies, config.cashiersPerCompany)

	report := []loadStageResult{}
	for stageIndex := 0; stageIndex*config.companiesPerStage < config.maxCompanies; stageIndex++ {
		recorder.startStage(stageIndex)
		// The stage's new companies onboard while the earlier ones keep working; each one starts
		// its people as soon as its own onboarding ends.
		firstCompanyID := stageIndex*config.companiesPerStage + 1
		lastCompanyID := min(firstCompanyID+config.companiesPerStage-1, config.maxCompanies)
		for companyID := int32(firstCompanyID); companyID <= int32(lastCompanyID); companyID++ {
			waitGroup.Go(func() {
				onboardingCompanies.Add(1)
				onboardingStart := time.Now()
				company, err := onboardLoadCompany(companyID, client, config)
				onboardingCompanies.Add(-1)
				if err != nil {
					core.Log("*LoadTest:: la empresa", companyID, "no pudo empezar:", err)
					appendLiveLine(config.liveStatusPath, fmt.Sprintf("company %d FAILED onboarding: %s", companyID, truncateForLog(err.Error(), 200)))
					return
				}
				appendLiveLine(config.liveStatusPath, fmt.Sprintf("company %d onboarded in %v, starts selling", companyID, time.Since(onboardingStart).Round(time.Second)))
				activeCompanies.Add(1)
				startCompanyPeople(company, client, config, &waitGroup, stopEverything)
			})
		}
		if !sleepOrStop(time.Duration(config.stageSeconds)*time.Second, stopEverything) {
			break
		}

		stageResult := recorder.summarizeStage(stageIndex, config.cashiersPerCompany, int(activeCompanies.Load()), config.stageSeconds)
		report = append(report, stageResult)
		core.Log("*" + stageResult.String())
		appendLiveLine(config.liveStatusPath, fmt.Sprintf("=== stage %d done: %d companies working, %.1f sales/s, sale p99 %v, errors %.2f%%",
			stageIndex+1, stageResult.companies, stageResult.salesPerSecond, stageResult.saleP99.Round(time.Millisecond), stageResult.errorPct))
		if stageResult.saleP99 > config.maxSaleP99 || stageResult.errorPct > config.maxErrorPct {
			core.Log("*LoadTest:: límite superado en la etapa", stageIndex+1, "; se detiene la rampa")
			appendLiveLine(config.liveStatusPath, "=== limit exceeded: the ramp stops")
			break
		}
	}
	close(stopEverything)
	stopSampler()
	stopLiveStatus()
	waitGroup.Wait()

	core.Log("*" + formatLoadReport(report))
	return core.FuncResponse{Message: "Prueba de carga terminada."}
}

// onboardLoadCompany takes an empty company (fn-init created only the company and its admin) to
// its first sale, through the server like a new customer would: the admin creates the staff, the
// back office imports the catalog, registers providers, supplies and the 100 supply materials
// (equipment and the consumables its expenses buy), opens the
// second store's warehouse and the tills, and buys the first stock for each warehouse.
func onboardLoadCompany(companyID int32, client *loadTestClient, config loadTestConfig) (*loadCompany, error) {
	if err := setLoadCompanyBudget(companyID); err != nil {
		return nil, err
	}

	company, err := createLoadCompanyStaff(companyID, client, config.cashiersPerCompany)
	if err != nil {
		return nil, err
	}
	backOfficeToken := core.UsuarioToken{CompanyID: companyID, ID: company.userIDByName[loadTestBackOfficeLogin]}

	supplyConfig := makeSupplyDataDefaults()
	supplyConfig.catalogProducts = config.catalogProducts
	supplyConfig.productsCount = config.catalogProducts
	supplyConfig.assetsCount = 0
	supplier := &supplyDataGenerator{
		random: mrand.New(mrand.NewSource(int64(companyID))), config: supplyConfig,
		userToken: backOfficeToken, sendOverHTTP: client.senderFor(backOfficeToken),
	}
	if _, err := supplier.importProductsWorkbook(); err != nil {
		return nil, core.Err("importando el catálogo:", err)
	}
	if err := supplier.seedProviders(); err != nil {
		return nil, core.Err("registrando proveedores:", err)
	}
	if err := supplier.loadProducts(); err != nil {
		return nil, err
	}
	if err := supplier.postSupplyBatches(supplier.makeSupplyRecords()); err != nil {
		return nil, core.Err("guardando el abastecimiento:", err)
	}
	if err := supplier.seedSupplyMaterials(); err != nil {
		return nil, core.Err("registrando insumos:", err)
	}

	historyConfig := makeDemoDataDefaults()
	// Serial numbers are minted from the day and the product: several orders a day would collide.
	historyConfig.serialProducts = 0
	historyConfig.reset = true
	template := newHistoryGenerator(historyConfig, "genix_load_test_state.json")
	template.userToken = backOfficeToken
	template.sendOverHTTP = client.senderFor(backOfficeToken)
	if err := template.loadState(); err != nil {
		return nil, err
	}
	for _, onboardingStep := range []func() error{
		template.ensureDemoWarehouses, template.validateContext, template.ensureDemoCashAccounts,
		template.prepareErpCatalog, template.prepareExpenses,
	} {
		if err := onboardingStep(); err != nil {
			return nil, err
		}
	}
	// The first stock: one purchase order per warehouse, walked through to its reception.
	if err := template.runPurchaseOrdersStep(template.dayStartTime(core.TimeToFechaUnix(time.Now()))); err != nil {
		return nil, core.Err("comprando el primer stock:", err)
	}
	core.SetHistoricalUnix(0)
	company.template = template
	core.Log("*LoadTest:: empresa", companyID, "lista:", len(template.productPool), "productos,",
		len(template.clientIDs), "clientes")
	return company, nil
}

// createLoadCompanyStaff is the company admin (user 1, from fn-init) creating the back-office user,
// the manager and the cashiers, each with every access.
func createLoadCompanyStaff(companyID int32, client *loadTestClient, cashierCount int) (*loadCompany, error) {
	allAccessGrants, err := makeAllAccessGrants()
	if err != nil {
		return nil, err
	}
	logins := []string{loadTestBackOfficeLogin, loadTestManagerLogin}
	for cashierIndex := 1; cashierIndex <= cashierCount; cashierIndex++ {
		logins = append(logins, fmt.Sprintf("%s%02d", loadTestCashierPrefix, cashierIndex))
	}
	sendAsAdmin := client.senderFor(core.UsuarioToken{CompanyID: companyID, ID: 1})
	for _, login := range logins {
		bodyBytes, _ := json.Marshal(coreTypes.User{
			User: login, FirstName: login, LastName: "Load Test", Password: loadTestPassword,
			AccesosGrants: allAccessGrants,
		})
		body := string(bodyBytes)
		if response := sendAsAdmin(core.HandlerArgs{Body: &body, Route: "POST.users", Method: "POST"}); response.StatusCode != 200 {
			return nil, core.Err("creando el usuario", login, ":", response.Error)
		}
	}

	users := []coreTypes.User{}
	query := db.Query(&users)
	if err := query.Select(query.ID, query.User).CompanyID.Equals(companyID).Exec(); err != nil {
		return nil, err
	}
	company := &loadCompany{userIDByName: map[string]int32{}}
	for _, user := range users {
		company.userIDByName[user.User] = user.ID
	}
	return company, nil
}

// startCompanyPeople starts the company's back office, manager and cashiers.
func startCompanyPeople(company *loadCompany, client *loadTestClient, config loadTestConfig, waitGroup *sync.WaitGroup, stop chan struct{}) {
	// The back office stays under --sales-only: its purchase orders are what restocks the shelves.
	waitGroup.Go(func() { runBackOffice(company, client, stop) })
	if !config.salesOnly {
		waitGroup.Go(func() { runManager(company, client, stop) })
	}
	for cashierIndex := 1; cashierIndex <= config.cashiersPerCompany; cashierIndex++ {
		cashierID := company.userIDByName[fmt.Sprintf("%s%02d", loadTestCashierPrefix, cashierIndex)]
		startDelay := time.Duration(mrand.Int63n(int64(loadTestCashierStartSpan)))
		waitGroup.Go(func() { runCashier(company, cashierID, client, config, startDelay, stop) })
	}
}

// cloneForPerson is the company's generator for one of its people: its own token, random sequence
// and copy of the stock ledger. The catalog maps are shared, read-only from here on.
func (template *erpHistoryGenerator) cloneForPerson(userID int32, client *loadTestClient) *erpHistoryGenerator {
	person := *template
	person.random = mrand.New(mrand.NewSource(int64(template.userToken.CompanyID)*1_000_003 + int64(userID)))
	person.userToken.ID = userID
	person.sendOverHTTP = client.senderFor(person.userToken)
	person.state = newHistoryState("")
	person.statePath = ""
	person.bucketsByWarehouse = map[int32]map[int32][]*stockBucket{}
	for warehouseID, bucketsByProduct := range template.bucketsByWarehouse {
		copiedByProduct := make(map[int32][]*stockBucket, len(bucketsByProduct))
		for productID, buckets := range bucketsByProduct {
			for _, bucket := range buckets {
				copiedBucket := *bucket
				copiedByProduct[productID] = append(copiedByProduct[productID], &copiedBucket)
			}
		}
		person.bucketsByWarehouse[warehouseID] = copiedByProduct
	}
	return &person
}

// runCashier sells until the run stops: the settlements that came due, a sale, sometimes its
// annulment, the page reload every salesPerPageReload sales, and a think time between sales.
func runCashier(company *loadCompany, cashierID int32, client *loadTestClient, config loadTestConfig, startDelay time.Duration, stop chan struct{}) {
	cashier := company.template.cloneForPerson(cashierID, client)
	authorization := client.authorizationFor(cashier.userToken)
	// The cashier's browser already holds its lists from earlier in the day.
	syncCursors := newDeltaSyncCursors(time.Now().Add(-time.Hour))
	// Oldest first: every settlement waits the same delay, so they come due in the order queued.
	pendingSettlements := []loadTestSettlement{}
	if !sleepOrStop(startDelay, stop) {
		return
	}

	for salesSinceReload := 0; ; salesSinceReload++ {
		if !config.salesOnly && salesSinceReload%salesPerPageReload == 0 {
			isClearedCache := cashier.random.Intn(100) < firstSyncPercent
			client.syncCashierPage(authorization, syncCursors, isClearedCache)
		}

		dueCount := 0
		for dueCount < len(pendingSettlements) && !time.Now().Before(pendingSettlements[dueCount].dueAt) {
			cashier.settleLoadTestSale(client, authorization, pendingSettlements[dueCount])
			dueCount++
		}
		pendingSettlements = pendingSettlements[dueCount:]

		isUnpaid := cashier.random.Intn(100) < config.unpaidPercent
		isUndelivered := cashier.random.Intn(100) < config.undeliveredPercent
		createdSale, err := cashier.createSaleWithRetry(time.Now(), isUnpaid, isUndelivered)
		if err == nil {
			company.sales.Add(1)
			isAnnulled := cashier.random.Intn(100) < annulledSalePercent && cashier.annulLoadTestSale(createdSale)
			// An annulled sale is never settled; payment goes first when both are pending.
			if config.settleDelay > 0 && !isAnnulled {
				dueAt := time.Now().Add(config.settleDelay)
				if isUnpaid {
					pendingSettlements = append(pendingSettlements, loadTestSettlement{createdSale, loadTestSettlePayment, dueAt})
				}
				if isUndelivered {
					pendingSettlements = append(pendingSettlements, loadTestSettlement{createdSale, loadTestSettleDelivery, dueAt})
				}
			}
		}

		thinkJitter := time.Duration(cashier.random.Int63n(int64(config.thinkTime) + 1))
		if !sleepOrStop(config.thinkTime/2+thinkJitter, stop) {
			return
		}
	}
}

// The ActionsIncluded of a settlement: 2 pays the sale, 3 delivers it.
const (
	loadTestSettlePayment  = int8(2)
	loadTestSettleDelivery = int8(3)
)

// loadTestSettlement is a payment or a delivery the cashier owes a sale, due at dueAt.
type loadTestSettlement struct {
	sale   salesTypes.SaleOrder
	action int8
	dueAt  time.Time
}

// settleLoadTestSale pays the whole debt into the warehouse's till, or delivers from the sale's
// warehouse, through the sale route's update. Recorded as settle:payment / settle:delivery, apart
// from the sales.
func (generator *erpHistoryGenerator) settleLoadTestSale(client *loadTestClient, authorization string, settlement loadTestSettlement) {
	settlementRequest := map[string]any{
		"ID": settlement.sale.ID, "ActionsIncluded": []int8{settlement.action},
		"WarehouseID": settlement.sale.WarehouseID,
	}
	operation := "settle:delivery"
	if settlement.action == loadTestSettlePayment {
		operation = "settle:payment"
		settlementRequest["DebtAmount"] = 0
		settlementRequest["LastPaymentCajaID"] = generator.tillByWarehouse[settlement.sale.WarehouseID]
	}
	bodyBytes, _ := json.Marshal(settlementRequest)
	client.post(authorization, operation, "sale-order", bodyBytes)
}

// annulLoadTestSale voids a sale the cashier just made, refunding it to the till it was paid into.
// Reports whether it sent the annulment.
func (generator *erpHistoryGenerator) annulLoadTestSale(sale salesTypes.SaleOrder) bool {
	// An unpaid sale has nothing to refund, and the annulment needs the till the payment went to.
	if sale.LastPaymentCajaID == 0 {
		return false
	}
	bodyBytes, _ := json.Marshal(map[string]any{
		"ID": sale.ID, "RefundCashBankID": sale.LastPaymentCajaID, "Reason": demoAnnulReasons[0],
	})
	request := generator.makeRequest("POST.sale-order-annul", nil, string(bodyBytes))
	generator.sendOverHTTP(request)
	return true
}

// runBackOffice books an expense (paid on the spot) every salesPerExpense company sales and a full
// purchase order every salesPerPurchaseOrder, alternating the warehouses.
func runBackOffice(company *loadCompany, client *loadTestClient, stop chan struct{}) {
	backOffice := company.template.cloneForPerson(company.template.userToken.ID, client)
	nextExpenseAt, nextPurchaseAt := int64(salesPerExpense), int64(salesPerPurchaseOrder)
	purchaseIndex := 0
	for sleepOrStop(backOfficePollInterval, stop) {
		companySales := company.sales.Load()
		if companySales >= nextExpenseAt {
			nextExpenseAt += salesPerExpense
			if err := backOffice.bookLoadTestExpense(time.Now()); err != nil {
				core.Log("LoadTest:: gasto de la empresa", backOffice.userToken.CompanyID, ":", err)
			}
		}
		if companySales >= nextPurchaseAt {
			nextPurchaseAt += salesPerPurchaseOrder
			warehouseIDs := backOffice.config.warehouseIDs
			todayStart := backOffice.dayStartTime(core.TimeToFechaUnix(time.Now()))
			if err := backOffice.runPurchaseOrder(todayStart, purchaseIndex, warehouseIDs[purchaseIndex%len(warehouseIDs)]); err != nil {
				core.Log("LoadTest:: compra de la empresa", backOffice.userToken.CompanyID, ":", err)
			}
			purchaseIndex++
		}
	}
}

// bookLoadTestExpense books one expense of a template a business pays any day (not the monthly
// schedules nor the taxes, which belong to a calendar), paying it at once from the bank.
func (generator *erpHistoryGenerator) bookLoadTestExpense(now time.Time) error {
	candidateTemplates := []expenseTemplate{}
	for _, template := range expenseTemplates {
		if template.cadence != cadenceScheduledMonthly && template.cadence != cadenceMonthlyTax {
			candidateTemplates = append(candidateTemplates, template)
		}
	}
	template := candidateTemplates[generator.random.Intn(len(candidateTemplates))]
	planned := plannedExpense{TemplateKey: template.key, PayDay: core.TimeToFechaUnix(now)}
	if template.isInventory {
		supply := generator.expenseSupplies[generator.random.Intn(len(generator.expenseSupplies))]
		planned.ProductID = supply.ID
		planned.WarehouseID = generator.config.warehouseIDs[generator.random.Intn(len(generator.config.warehouseIDs))]
		planned.Quantity = int32(5 + generator.random.Intn(26))
		planned.UnitPrice = supply.Price
		planned.Amount = planned.UnitPrice * planned.Quantity
	} else {
		planned.Amount = int32(generator.randomInRange(int(template.minAmount), int(template.maxAmount)))
	}
	planned.PayAmount = planned.Amount
	return generator.bookPlannedExpense(planned, now)
}

// loadTestReports are what the manager opens, in rotation. They go without the browser's cached
// groups and frame files, so each one reads its whole range: the costliest form of the report.
func loadTestReports(now time.Time) []string {
	today := core.TimeToFechaUnix(now)
	period := now.Format("200601")
	return []string{
		fmt.Sprintf("sale-summary?f=sale-summary&from=%d&to=%d", today-30, today),
		fmt.Sprintf("sales-book?period=%s", period),
		fmt.Sprintf("accounting-operations?period=%s", period),
		fmt.Sprintf("accounting-cash-balances?period=%s", period),
		fmt.Sprintf("sale-order-query?date-start=%d&date-end=%d", today-7, today),
	}
}

// runManager opens one report every salesPerReport company sales.
func runManager(company *loadCompany, client *loadTestClient, stop chan struct{}) {
	managerToken := core.UsuarioToken{CompanyID: company.template.userToken.CompanyID, ID: company.userIDByName[loadTestManagerLogin]}
	authorization := client.authorizationFor(managerToken)
	nextReportAt := int64(salesPerReport)
	for reportIndex := 0; sleepOrStop(backOfficePollInterval, stop); {
		if company.sales.Load() < nextReportAt {
			continue
		}
		nextReportAt += salesPerReport
		reports := loadTestReports(time.Now())
		reportPath := reports[reportIndex%len(reports)]
		reportName, _, _ := strings.Cut(reportPath, "?")
		client.get(authorization, "report:"+reportName, reportPath)
		reportIndex++
	}
}

func sleepOrStop(duration time.Duration, stop chan struct{}) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-stop:
		return false
	case <-timer.C:
		return true
	}
}

// ─── Delta syncs ────────────────────────────────────────────────────────────────

// cashierPageSyncs are the delta lists the sales page loads, with the param each one's cursor
// travels in.
var cashierPageSyncs = []struct{ name, path, cursorParam string }{
	{"products", "products", "records"},
	{"products-stock", "products-stock", core.DeltaCursorParamOfDefaultKey},
	{"sale-orders", "sale-orders?pending-status=2", "records"},
	{"client-provider", "client-provider?type=1", core.DeltaCursorParamOfDefaultKey},
}

// deltaSyncCursors is what one browser holds per list: the server time its last sync started.
// The cursor sent is "C.V" with C = V = that time, so the server answers what changed since and
// hashes the unsettled window; the load test never checks the hash, so it never retries.
type deltaSyncCursors struct {
	lastSyncByList map[string]time.Time
}

func newDeltaSyncCursors(heldSince time.Time) *deltaSyncCursors {
	cursors := &deltaSyncCursors{lastSyncByList: map[string]time.Time{}}
	for _, list := range cashierPageSyncs {
		cursors.lastSyncByList[list.name] = heldSince
	}
	return cursors
}

// syncCashierPage sends the page's delta syncs. A cleared cache sends no cursor: a first sync.
func (client *loadTestClient) syncCashierPage(authorization string, cursors *deltaSyncCursors, isClearedCache bool) {
	for _, list := range cashierPageSyncs {
		syncStart := time.Now()
		requestPath, operation := list.path, "sync:"+list.name
		if isClearedCache {
			operation = "first-sync:" + list.name
		} else {
			heldTime := int32((cursors.lastSyncByList[list.name].Unix() - 1e9) / 2)
			separator := core.If(strings.Contains(requestPath, "?"), "&", "?")
			requestPath += fmt.Sprintf("%s%s=%d.%d", separator, list.cursorParam, heldTime, heldTime)
		}
		if client.get(authorization, operation, requestPath) {
			cursors.lastSyncByList[list.name] = syncStart
		}
	}
}

// ─── HTTP ───────────────────────────────────────────────────────────────────────

type loadTestClient struct {
	serverURL string
	http      *http.Client
	recorder  *loadRecorder
}

// authorizationFor builds the session token login.go hands the browser, without the login round
// trip: the server validates its keyed hash, not that a login happened.
func (client *loadTestClient) authorizationFor(userToken core.UsuarioToken) string {
	userToken.Created = int32((time.Now().Unix() - 1e9) / 2)
	userToken.User = "load-test"
	userToken.Hash = core.ComputeUsuarioTokenHash(userToken)
	tokenBytes, err := colbin.Marshal(userToken)
	if err != nil {
		panic("load test: encoding the session token: " + err.Error())
	}
	return "Bearer " + utils.Base64.EncodeToString(tokenBytes)
}

// senderFor is the generator's transport to the server for one user: the request becomes the HTTP
// call the browser makes, JSON in and out, and the answer comes back as the handler's would.
func (client *loadTestClient) senderFor(userToken core.UsuarioToken) func(core.HandlerArgs) core.HandlerResponse {
	authorization := client.authorizationFor(userToken)
	return func(request core.HandlerArgs) core.HandlerResponse {
		method, routePath, _ := strings.Cut(request.Route, ".")
		query := url.Values{}
		for key, value := range request.Query {
			query.Set(key, value)
		}
		requestURL := client.serverURL + "/api/" + routePath
		if len(query) > 0 {
			requestURL += "?" + query.Encode()
		}
		var body io.Reader
		if request.Body != nil && method != http.MethodGet {
			body = strings.NewReader(*request.Body)
		}
		httpRequest, _ := http.NewRequest(method, requestURL, body)
		httpRequest.Header.Set("Authorization", authorization)

		statusCode, responseBody, err := client.do(httpRequest, request.Route)
		if err != nil {
			return core.HandlerResponse{StatusCode: 599, Error: err.Error()}
		}
		if statusCode != http.StatusOK {
			return core.HandlerResponse{StatusCode: statusCode, Error: string(responseBody)}
		}
		return core.HandlerResponse{StatusCode: 200, Body: &responseBody}
	}
}

// get sends a GET the way the browser does (colbin, zstd) and keeps only its size. Reports
// whether it succeeded.
func (client *loadTestClient) get(authorization, operation, requestPath string) bool {
	httpRequest, _ := http.NewRequest(http.MethodGet, client.serverURL+"/api/"+requestPath, nil)
	httpRequest.Header.Set("Authorization", authorization)
	httpRequest.Header.Set("Accept", core.ColbinMediaType)
	httpRequest.Header.Set("Accept-Encoding", "zstd")
	statusCode, _, err := client.do(httpRequest, operation)
	return err == nil && statusCode == http.StatusOK
}

// post sends a JSON POST recorded under operation, which need not be its route. Reports whether it
// succeeded.
func (client *loadTestClient) post(authorization, operation, routePath string, body []byte) bool {
	httpRequest, _ := http.NewRequest(http.MethodPost, client.serverURL+"/api/"+routePath, bytes.NewReader(body))
	httpRequest.Header.Set("Authorization", authorization)
	statusCode, _, err := client.do(httpRequest, operation)
	return err == nil && statusCode == http.StatusOK
}

var zstdDecoder, _ = zstd.NewReader(nil)

// do runs one request and records it under operation. A stock shortage is the business refusing
// an oversold sale — what a real till sees under concurrent sales — so it counts as refused, not
// as an error.
func (client *loadTestClient) do(httpRequest *http.Request, operation string) (int, []byte, error) {
	requestStart := time.Now()
	response, err := client.http.Do(httpRequest)
	if err != nil {
		client.recorder.record(operation, time.Since(requestStart), loadOutcomeError, 0)
		return 0, nil, err
	}
	responseBody, err := io.ReadAll(response.Body)
	response.Body.Close()
	elapsed := time.Since(requestStart)
	if err == nil && response.Header.Get("Content-Encoding") == "zstd" && httpRequest.Header.Get("Accept") != core.ColbinMediaType {
		responseBody, err = zstdDecoder.DecodeAll(responseBody, nil)
	}
	outcome := loadOutcomeOK
	switch {
	case err != nil:
		outcome = loadOutcomeError
	case response.StatusCode != http.StatusOK && bytes.Contains(responseBody, []byte("Se posee en stock")):
		outcome = loadOutcomeRefused
	case response.StatusCode != http.StatusOK:
		outcome = loadOutcomeError
		core.Log("LoadTest::", operation, response.StatusCode, truncateForLog(string(responseBody), 300))
	}
	client.recorder.record(operation, elapsed, outcome, len(responseBody))
	return response.StatusCode, responseBody, err
}

func truncateForLog(text string, maxLength int) string {
	if len(text) > maxLength {
		return text[:maxLength] + "…"
	}
	return text
}

// ─── Live status ────────────────────────────────────────────────────────────────

// startLiveStatus appends one line every liveStatusInterval to path: what the last interval did.
// An empty path writes nothing.
func startLiveStatus(path string, recorder *loadRecorder, activeCompanies, onboardingCompanies *atomic.Int64, cashiersPerCompany int) func() {
	if path == "" {
		return func() {}
	}
	liveFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		core.Log("LoadTest:: no se pudo abrir el estado en vivo", path, ":", err)
		return func() {}
	}
	stopLive := make(chan struct{})
	liveDone := make(chan struct{})
	go func() {
		defer close(liveDone)
		defer liveFile.Close()
		recorder.takeLiveWindow()
		previousTicks := map[string]int64{}
		intervalSeconds := liveStatusInterval.Seconds()
		for sleepOrStop(liveStatusInterval, stopLive) {
			window := recorder.takeLiveWindow()
			slices.Sort(window.saleDurations)
			line := fmt.Sprintf("%s companies %3d (+%d onboarding) cashiers %4d | sales/s %6.1f req/s %6.1f | sale p50 %6v p99 %6v | refused %4d errors %3d",
				time.Now().Format("15:04:05"), activeCompanies.Load(), onboardingCompanies.Load(), activeCompanies.Load()*int64(cashiersPerCompany),
				float64(len(window.saleDurations))/intervalSeconds, float64(window.requests)/intervalSeconds,
				percentileOf(window.saleDurations, 0.50).Round(time.Millisecond),
				percentileOf(window.saleDurations, 0.99).Round(time.Millisecond), window.refused, window.errors)
			processNames := slices.Sorted(maps.Keys(window.cpuTicksByProcess))
			for _, processName := range processNames {
				cpuTicks := window.cpuTicksByProcess[processName]
				cpuPercent := 0.0
				if previous, hasPrevious := previousTicks[processName]; hasPrevious {
					cpuPercent = 100 * float64(cpuTicks-previous) / clockTicksPerSecond / intervalSeconds
				}
				previousTicks[processName] = cpuTicks
				line += fmt.Sprintf(" | %s %3.0f%% %4dMB", processName, cpuPercent, window.rssBytesByProcess[processName]>>20)
			}
			fmt.Fprintln(liveFile, line)
		}
	}()
	return func() { close(stopLive); <-liveDone }
}

// appendLiveLine adds one event line to the live log. O_APPEND keeps it whole beside the status
// lines startLiveStatus writes.
func appendLiveLine(path, text string) {
	if path == "" {
		return
	}
	liveFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer liveFile.Close()
	fmt.Fprintln(liveFile, time.Now().Format("15:04:05")+" "+text)
}

// ─── Process sampling ───────────────────────────────────────────────────────────

// startProcessSampler records each watched process's memory (VmRSS) and CPU (utime+stime) every
// second, into the stage running at the time.
func startProcessSampler(watchedPIDs map[string]int, recorder *loadRecorder) func() {
	if len(watchedPIDs) == 0 {
		return func() {}
	}
	stopSampling := make(chan struct{})
	samplerDone := make(chan struct{})
	go func() {
		defer close(samplerDone)
		for sleepOrStop(time.Second, stopSampling) {
			for processName, pid := range watchedPIDs {
				rssBytes, cpuTicks, err := readProcessUsage(pid)
				if err == nil {
					recorder.sampleProcess(processName, rssBytes, cpuTicks)
				}
			}
		}
	}()
	return func() { close(stopSampling); <-samplerDone }
}

// readProcessUsage reads VmRSS from /proc/<pid>/status and utime+stime (clock ticks) from stat.
func readProcessUsage(pid int) (int64, int64, error) {
	statusBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, 0, err
	}
	rssBytes := int64(0)
	for _, line := range strings.Split(string(statusBytes), "\n") {
		if rssText, isRSS := strings.CutPrefix(line, "VmRSS:"); isRSS {
			rssKB, _ := strconv.ParseInt(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rssText), "kB")), 10, 64)
			rssBytes = rssKB << 10
		}
	}
	statBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0, err
	}
	// The command name may hold spaces; the fields after it are space separated.
	afterCommand := string(statBytes[bytes.LastIndexByte(statBytes, ')')+2:])
	statFields := strings.Fields(afterCommand)
	userTicks, _ := strconv.ParseInt(statFields[11], 10, 64)
	systemTicks, _ := strconv.ParseInt(statFields[12], 10, 64)
	return rssBytes, userTicks + systemTicks, nil
}

// setLoadCompanyBudget gives the company a credit budget far past what a run spends, so a 429 never
// measures the limiter's policy: without a budget row its daily allowance is zero. Under
// [fareward].bypass the server charges nothing, so there is no budget to set.
func setLoadCompanyBudget(companyID int32) error {
	if core.Env.FAREWARD_BYPASS {
		return nil
	}
	budgetContext, cancelBudget := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelBudget()
	for _, budgetOperation := range []fareward.BudgetOperation{fareward.BudgetSetDaily, fareward.BudgetSetCurrent} {
		if err := fareward.MutateCompanyCreditBudget(budgetContext, companyID, budgetOperation, 1<<40, 1<<40); err != nil {
			return core.Err("no se pudo fijar el presupuesto:", err)
		}
	}
	return nil
}
