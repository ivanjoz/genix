package runner

import (
	"app/core"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Exported fields because the whole struct is persisted in the resume file.
type Stats struct {
	PurchaseOrders   int
	ReceivedUnits    int32
	SalesCreated     int
	SalesInvoiced    int
	SalesUnpaid      int
	SalesUndelivered int
	SalesAnnulled    int
	CashInjections   int
	CashInjected     int32
	StockRetries     int
	Expenses         int
	ExpensePayments  int
	TillTransfers    int
	Reconciliations  int
}

// State is the resume point, rewritten after every purchase order and after every single sale. A
// closed terminal or a killed process therefore loses at most one record, and the rerun continues
// from the exact day and step instead of replaying — which would duplicate the history, since the
// day steps are not idempotent.
//
// The product pool travels in the state on purpose: it is drawn at random once, and the whole
// point of the run is that the same products repeat across days. Redrawing it on resume would split
// one run into two unrelated catalogs.
type State struct {
	Version         int
	ConfigSignature string
	ProductPool     []int32

	// Day in progress and how far into it the run got.
	UnixDay           int16
	CompletedDaySteps int

	// logistics
	CompletedPurchaseOrders int

	// sales: the day's plan is drawn once and stored, so a resumed day keeps the exact
	// unpaid/undelivered/annulled counts that were promised when it started.
	SalesPlanCount        int
	SalesUnpaidFlags      []bool
	SalesUndeliveredFlags []bool
	SalesAnnulFlags       []bool
	CompletedSales        int
	// Sales flagged for annulment, voided by the evening step once the day's selling is done.
	PendingAnnulments []PendingAnnulment

	// finance: the day's expense plan, drawn once like the sales plan, and the payments booked for
	// a later day.
	ExpensePlan            []PlannedExpense
	ExpensePlanDrawn       bool
	CompletedExpenses      int
	PendingExpensePayments []PendingExpensePayment

	// Next comprobante number per supplier document key (purchase orders and expenses), so a
	// supplier's numbers keep increasing.
	NextDocNumberByKey map[string]int64
	// What the month's taxes are computed from: sales add to it, purchases and expenses credit it.
	TaxTotalsByPeriod map[int16]*PeriodTaxTotals

	Stats            Stats
	AffectedUnixDays []int16
}

type PendingAnnulment struct {
	SaleID     int64
	CashBankID int32
}

// PlannedExpense is one expense of the day, drawn when the day starts and stored in the state, so
// a resumed day books exactly what it planned.
type PlannedExpense struct {
	TemplateKey string
	Amount      int32
	// What gets paid and when: 0 leaves the expense pending forever.
	PayAmount int32
	PayDay    int16
	// Inventory purchases only.
	ProductID   int32
	WarehouseID int32
	Quantity    int32
	UnitPrice   int32
}

// PendingExpensePayment is a payment promised for a later day — a bill paid at its due date.
type PendingExpensePayment struct {
	ExpenseID int32
	PayDay    int16
	Amount    int32
}

// planSignature covers what the generated history looks like — the modules and the arguments — and
// deliberately leaves out reset, dry-run and yes: those pick how this invocation runs, not what it
// produces. Including them would make the resume command look like a different plan.
func planSignature(config Config, moduleNames []string) string {
	config.Reset, config.DryRun, config.SkipConfirm = false, false, false
	return fmt.Sprintf("%s %+v", strings.Join(moduleNames, ","), config)
}

func NewState(configSignature string) State {
	return State{
		Version:            stateVersion,
		ConfigSignature:    configSignature,
		NextDocNumberByKey: map[string]int64{},
		TaxTotalsByPeriod:  map[int16]*PeriodTaxTotals{},
	}
}

// StatePathFor is a company's resume file. One per company, kept beside the database's file store:
// a state belongs to one keyspace, and a reseed that wipes the store wipes it too.
func StatePathFor(companyID int32) string {
	return filepath.Join(core.VPSFileStoreDir(), "seed-state", fmt.Sprintf("c%d_%s", companyID, StateFileName))
}

// LoadState restores an interrupted run. A state written under different modules or arguments is
// refused instead of merged: resuming a 15-day plan into a 4-day one would silently produce neither.
func (run *Run) LoadState(moduleNames []string) error {
	run.StatePath = StatePathFor(run.UserToken.CompanyID)
	if err := os.MkdirAll(filepath.Dir(run.StatePath), 0o755); err != nil {
		return err
	}
	configSignature := planSignature(run.Config, moduleNames)

	if run.Config.Reset {
		if err := os.Remove(run.StatePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		run.State = NewState(configSignature)
		return nil
	}

	stateBytes, readError := os.ReadFile(run.StatePath)
	if os.IsNotExist(readError) {
		run.State = NewState(configSignature)
		return nil
	}
	if readError != nil {
		return readError
	}

	storedState := State{}
	if err := json.Unmarshal(stateBytes, &storedState); err != nil {
		return core.Err("el archivo de estado", run.StatePath, "está corrupto; usa --reset:", err)
	}
	if storedState.Version != stateVersion {
		return core.Err("el archivo de estado es de otra versión; usa --reset")
	}
	if storedState.ConfigSignature != configSignature {
		return core.Err("el estado guardado corresponde a otros módulos o argumentos; usa --reset para empezar de nuevo")
	}

	run.State = storedState
	run.ProductPool = storedState.ProductPool
	core.Log("Seed:: reanudando", "día", storedState.UnixDay, "pasos hechos", storedState.CompletedDaySteps,
		"OCs hechas", storedState.CompletedPurchaseOrders, "ventas hechas", storedState.CompletedSales,
		"de", storedState.SalesPlanCount)
	return nil
}

// SaveState rewrites the resume point. The write goes to a sibling file and is renamed, because a
// process killed halfway through a plain write would leave unparseable JSON behind — and the state
// file is only useful if it survives exactly the event it exists for. A run without a state file
// (the load test's) keeps its state in memory only.
//
// Every checkpoint is also where an abort lands: right after the resume point is on disk, a
// cancelled run returns errAborted, which every step propagates. Stopping anywhere else could leave
// a purchase order half walked, which a resume would then create a second time.
func (run *Run) SaveState() error {
	if run.StatePath != "" {
		stateBytes, marshalError := json.Marshal(run.State)
		if marshalError != nil {
			return marshalError
		}
		temporaryPath := run.StatePath + ".tmp"
		if err := os.WriteFile(temporaryPath, stateBytes, 0o644); err != nil {
			return err
		}
		if err := os.Rename(temporaryPath, run.StatePath); err != nil {
			return err
		}
	}
	if run.IsAborted() {
		return errAborted
	}
	return nil
}

var errAborted = errors.New("corrida abortada; repite el mismo comando para reanudar")

// startDay resets the per-day counters, unless the run is resuming into that same day.
func (run *Run) startDay(targetUnixDay int16) error {
	if run.State.UnixDay == targetUnixDay {
		return nil
	}
	run.State.UnixDay = targetUnixDay
	run.State.CompletedDaySteps = 0
	run.State.CompletedPurchaseOrders = 0
	run.State.SalesPlanCount = 0
	run.State.SalesUnpaidFlags = nil
	run.State.SalesUndeliveredFlags = nil
	run.State.SalesAnnulFlags = nil
	run.State.CompletedSales = 0
	run.State.ExpensePlan = nil
	run.State.ExpensePlanDrawn = false
	run.State.CompletedExpenses = 0
	return run.SaveState()
}
