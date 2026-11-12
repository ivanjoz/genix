// Package runner is the machinery every seed module shares: the run's configuration and resume
// state, the frozen-clock day loop, the request plumbing into the production handlers, and the
// cash, stock and document helpers more than one module needs.
//
// The backdating is not simulated at the record level: core.SetHistoricalUnix freezes the whole
// process clock on the simulated instant, so the business handlers, the stock ledger, the cash
// register and the ORM's own created/updated columns all agree on the date. Every write goes
// through the production handler with the exact JSON payload its route expects.
package runner

import (
	"app/core"
	"context"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"os"
	"slices"
	"strings"
	"time"

	production "app/production/types"
)

// CompanyID is the company every module writes. --company=N sets it for the whole process, like
// the historical clock: one process seeds one company. The load test, which drives several, gives
// each run its own UserToken instead.
var CompanyID int32 = 1

const (
	// UserID is the company admin fn-init creates; every request is sent as that user.
	UserID        int32 = 1
	StateFileName       = "genix_seed_state.json"
	stateVersion        = 3
	// cashTopUpPadding is the extra injected on top of a shortfall, so one injection covers
	// several payments instead of one per payment.
	cashTopUpPadding int32 = 100000
	// DefaultSeed is the default random seed: a wiped database refilled with the same arguments
	// gets the same data back. --seed=N draws another one.
	DefaultSeed int64 = 20260101
	// PostBatchSize keeps one request under the handlers' own 1000-record cap.
	PostBatchSize = 500
)

// PartySeed is the seed shape shared by every party JSON — clients, ERP providers, supply providers
// and expense suppliers. CountryID only matters for providers, which SaveClientProviders rejects
// without one.
type PartySeed struct {
	Name           string `json:"name"`
	PersonType     int8   `json:"personType"`
	RegistryNumber string `json:"registryNumber"`
	CountryID      int16  `json:"countryId"`
	CityID         string `json:"cityId"`
}

// Run is one seeding of one company: what the modules' setups resolved and what their day steps
// share. Each module writes the fields of its own domain; the comments say which.
type Run struct {
	Random  *mrand.Rand
	RealNow time.Time
	Config  Config
	// Context cancelled asks the run to stop at its next checkpoint (SaveState). Nil never stops.
	Context  context.Context
	lockFile *os.File

	UserToken core.UsuarioToken
	// SendOverHTTP, when set, sends every request to a running server (the load test) instead of
	// calling the handler in this process. See Send.
	SendOverHTTP func(request core.HandlerArgs) core.HandlerResponse
	daySteps     []DayStep

	// base: one till per warehouse collects the sales; the bank pays suppliers, expenses and payroll.
	TillByWarehouse map[int32]int32
	BankCashBankID  int32

	// base: ProviderIDs are the grocery suppliers the purchase orders buy from, SupplyProviderIDs
	// the static list the replenishment settings and the fixed assets name.
	ProviderIDs       []int32
	SupplyProviderIDs []int32
	// base: clients split by what document a sale to them can carry: a RUC takes a factura, a DNI
	// lets a boleta of S/ 700 or more name its buyer, and the rest buy anonymously.
	ClientIDs         []int32
	BusinessClientIDs []int32
	DNIClientIDs      []int32
	FacturaSeriesID   int8
	BoletaSeriesID    int8

	// base: the working catalog, drawn once and kept in the state.
	ProductPool        []int32
	ProductNameByID    map[int32]string
	SalePriceByProduct map[int32]int32
	CostPriceByProduct map[int32]int32

	// BucketsByWarehouse[warehouseID][productID] holds every place that product has units: loaded
	// by base, filled by the logistics receptions, drawn down by the sales.
	BucketsByWarehouse map[int32]map[int32][]*StockBucket

	// finance
	ExpenseSupplierByRUC map[string]int32
	ExpenseScheduleByKey map[string]int32
	ExpenseSupplies      []production.Product

	StatePath string
	State     State
}

func New(config Config) *Run {
	return &Run{
		// Reproducible for an uninterrupted run; a resumed one restarts the sequence where it resumes.
		Random:  mrand.New(mrand.NewSource(config.Seed)),
		RealNow: time.Now(),
		Config:  config,
		UserToken: core.UsuarioToken{
			CompanyID: CompanyID,
			ID:        UserID,
		},
		TillByWarehouse:    map[int32]int32{},
		ProductNameByID:    map[int32]string{},
		SalePriceByProduct: map[int32]int32{},
		CostPriceByProduct: map[int32]int32{},
		BucketsByWarehouse: map[int32]map[int32][]*StockBucket{},
		State:              NewState(""),
	}
}

// FirstDay is the first simulated day of the run.
func (run *Run) FirstDay() int16 {
	return core.TimeToFechaUnix(run.RealNow) - int16(run.Config.Days-1)
}

// DayStartTime is local midnight of a simulated day. It is derived from the real clock captured
// at startup, never from core.Now(), which the run keeps moving.
func (run *Run) DayStartTime(targetUnixDay int16) time.Time {
	dayOffset := int(targetUnixDay - core.TimeToFechaUnix(run.RealNow))
	dayInTarget := run.RealNow.AddDate(0, 0, dayOffset)
	return time.Date(dayInTarget.Year(), dayInTarget.Month(), dayInTarget.Day(), 0, 0, 0, 0, dayInTarget.Location())
}

func (run *Run) PickDistinctFromPool(count int) []int32 {
	shuffledPool := slices.Clone(run.ProductPool)
	run.Random.Shuffle(len(shuffledPool), func(i, j int) {
		shuffledPool[i], shuffledPool[j] = shuffledPool[j], shuffledPool[i]
	})
	if count > len(shuffledPool) {
		count = len(shuffledPool)
	}
	return shuffledPool[:count]
}

func (run *Run) RandomInRange(minValue int, maxValue int) int {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + run.Random.Intn(maxValue-minValue+1)
}

// MakeRequest builds the synthetic request the production handlers expect. There is no timestamp
// here: the date comes from the process clock, which is exactly what makes this backdating global.
func (run *Run) MakeRequest(route string, query map[string]string, body string) core.HandlerArgs {
	method := route
	if separatorIndex := strings.IndexByte(route, '.'); separatorIndex > 0 {
		method = route[:separatorIndex]
	}
	return core.HandlerArgs{
		Body:   &body,
		Query:  query,
		Route:  route,
		Method: method,
		User:   &run.UserToken,
	}
}

// Send runs one request through its production handler, in this process. The load test sets
// SendOverHTTP instead, so the same flows reach a running server the way a browser's requests do.
func (run *Run) Send(request core.HandlerArgs, handler func(*core.HandlerArgs) core.HandlerResponse) core.HandlerResponse {
	if run.SendOverHTTP != nil {
		return run.SendOverHTTP(request)
	}
	return handler(&request)
}

// DecodeResponse reads a handler body. The run sends no Accept header, so the body is plain JSON.
func DecodeResponse[T any](response core.HandlerResponse, target *T) error {
	if response.StatusCode != 200 {
		return core.Err(response.Error)
	}
	if response.Body == nil {
		return core.Err("la respuesta del handler vino vacía")
	}
	return json.Unmarshal(*response.Body, target)
}

// CallAs runs a production handler in this process as the given user, with the payload marshalled
// as its JSON body, and decodes the answer into target when there is one. The HR, CRM and legal
// parameter seeds act as specific users rather than as the run's own token.
func CallAs(user core.UsuarioToken, route string, handler func(*core.HandlerArgs) core.HandlerResponse,
	query map[string]string, payload any, target any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body := string(bodyBytes)
	method, _, _ := strings.Cut(route, ".")
	if query == nil {
		query = map[string]string{}
	}
	request := core.HandlerArgs{Body: &body, Query: query, Route: route, Method: method, User: &user}
	response := handler(&request)
	if response.StatusCode != 200 {
		return fmt.Errorf("%s (usuario %d) respondió %d: %s", route, user.ID, response.StatusCode, response.Error)
	}
	if target == nil || response.Body == nil {
		return nil
	}
	return json.Unmarshal(*response.Body, target)
}

// IsMonthDay matches a day of the month, a day past the month's end meaning its last day — the same
// clamp the expense schedules apply.
func IsMonthDay(dayStart time.Time, monthDay int) bool {
	lastDayOfMonth := time.Date(dayStart.Year(), dayStart.Month()+1, 0, 0, 0, 0, 0, dayStart.Location()).Day()
	return dayStart.Day() == min(monthDay, lastDayOfMonth)
}
