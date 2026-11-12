package runner

import (
	"app/core"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"time"
)

// Module is one domain's seed. Setup runs on every invocation, before the day loop and with the
// clock frozen on the first simulated morning, so what it writes predates the history; it must be
// idempotent, because a resumed run calls it again. It returns the module's day steps — none for a
// module that is not lived day by day — so a step can close over what its setup loaded.
type Module struct {
	Name     string
	Requires []string
	Setup    func(run *Run) ([]DayStep, error)
}

// DayStep is one block of a simulated day. Order places it in the day among the steps of every
// selected module: the day runs them in Order and checkpoints after each one, so a resumed day
// skips the blocks it already finished; a step with many writes keeps its own finer checkpoint
// inside the state.
type DayStep struct {
	Name  string
	Order int
	Run   func(dayStart time.Time) error
}

// The day's order. Every module that moves cash writes into the same day loop, because a cash
// movement stores its register's balance at write time: modules replayed one after another over the
// same days would leave the running balances out of order.
const (
	OrderPurchases     = 10
	OrderExpenses      = 20
	OrderSales         = 30
	OrderAnnulments    = 40
	OrderTillTransfers = 50
	OrderPayroll       = 60
	OrderTillCounts    = 70
	OrderDepreciation  = 80
)

// ResolveModules picks the requested modules and everything they require, in the registry's order
// (which is the order their setups depend on). "all" selects every module.
func ResolveModules(registry []Module, requestedNames []string) ([]Module, error) {
	selected := map[string]bool{}
	var selectWithRequirements func(name string) error
	selectWithRequirements = func(name string) error {
		moduleIndex := slices.IndexFunc(registry, func(module Module) bool { return module.Name == name })
		if moduleIndex < 0 {
			return core.Err("módulo desconocido:", name)
		}
		selected[name] = true
		for _, requiredName := range registry[moduleIndex].Requires {
			if err := selectWithRequirements(requiredName); err != nil {
				return err
			}
		}
		return nil
	}
	for _, requestedName := range requestedNames {
		if requestedName == "all" {
			for _, module := range registry {
				selected[module.Name] = true
			}
			continue
		}
		if err := selectWithRequirements(requestedName); err != nil {
			return nil, err
		}
	}
	// base is always required: it holds the catalog every other module writes against.
	selected[registry[0].Name] = true
	return slices.DeleteFunc(slices.Clone(registry), func(module Module) bool { return !selected[module.Name] }), nil
}

// Execute seeds the run's company with the given modules: every setup, then one day loop with the
// steps of all of them, then the cash accounts' running balances put back in date order. A run
// cancelled through run.Context stops at its next checkpoint and resumes with the same command.
func Execute(run *Run, modules []Module) (map[string]any, error) {
	moduleNames := []string{}
	for _, module := range modules {
		moduleNames = append(moduleNames, module.Name)
	}
	if err := run.lockCompany(); err != nil {
		return nil, err
	}
	if err := run.LoadState(moduleNames); err != nil {
		return nil, core.Err("No se pudo cargar el estado de la corrida:", err)
	}
	// The clock is process-wide: leaving it frozen would date every later write in this process.
	defer core.SetHistoricalUnix(0)

	daySteps := []DayStep{}
	setupTime := run.DayStartTime(run.FirstDay()).Add(6 * time.Hour)
	for _, module := range modules {
		core.SetHistoricalUnix(setupTime.Unix())
		core.Log("Seed:: preparando", module.Name)
		moduleSteps, err := module.Setup(run)
		if err != nil {
			return nil, core.Err("módulo", module.Name, ":", err)
		}
		daySteps = append(daySteps, moduleSteps...)
	}
	core.SetHistoricalUnix(0)

	slices.SortStableFunc(daySteps, func(left, right DayStep) int { return left.Order - right.Order })
	run.daySteps = daySteps
	historicalDates := []int16{}
	if len(daySteps) > 0 {
		var err error
		if historicalDates, err = run.runDayLoop(); err != nil {
			return nil, err
		}
	}
	if err := os.Remove(run.StatePath); err != nil && !os.IsNotExist(err) {
		return nil, core.Err("No se pudo limpiar el estado:", err)
	}

	// A module run on days that already had movements (hr after the ERP history) wrote its payments
	// behind the later ones, so every account the run touched gets its running balance recomputed.
	rebalancedMovements := 0
	cashBankIDs := core.SliceSet[int32]{}
	cashBankIDs.Add(run.BankCashBankID)
	for _, tillID := range run.TillByWarehouse {
		cashBankIDs.Add(tillID)
	}
	for _, cashBankID := range cashBankIDs.Values {
		if cashBankID == 0 {
			continue
		}
		rewritten, err := run.RebalanceCashBank(cashBankID, run.DayStartTime(run.FirstDay()-1))
		if err != nil {
			return nil, core.Err("No se pudieron recalcular los saldos de la caja", cashBankID, ":", err)
		}
		rebalancedMovements += rewritten
	}

	core.Log("Seed:: terminado", "módulos", moduleNames, "días", len(historicalDates), "stats", run.State.Stats)
	return map[string]any{
		"modules":             moduleNames,
		"days":                len(historicalDates),
		"stats":               run.State.Stats,
		"rebalancedMovements": rebalancedMovements,
		"affectedUnixDays":    run.State.AffectedUnixDays,
	}, nil
}

// IsAborted reports whether the run was asked to stop.
func (run *Run) IsAborted() bool {
	return run.Context != nil && run.Context.Err() != nil
}

// lockCompany takes an exclusive lock per company, held until the process exits, so two seeds never
// interleave their writes into the same company.
func (run *Run) lockCompany() error {
	lockDir := filepath.Join(core.VPSFileStoreDir(), "seed-state")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return err
	}
	lockFile, err := os.OpenFile(filepath.Join(lockDir, fmt.Sprintf("c%d.lock", run.UserToken.CompanyID)), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lockFile.Close()
		return core.Err("otra siembra está corriendo sobre la empresa", run.UserToken.CompanyID)
	}
	// Never closed: the lock lives exactly as long as the process.
	run.lockFile = lockFile
	return nil
}

// runDayLoop walks the pending days with the selected modules' steps.
func (run *Run) runDayLoop() ([]int16, error) {
	historicalDates, err := run.resolvePendingDates()
	if err != nil {
		return nil, core.Err("No se pudo resolver el rango de días:", err)
	}
	for _, targetUnixDay := range historicalDates {
		if err := run.startDay(targetUnixDay); err != nil {
			return nil, err
		}
		if err := run.runOneDay(targetUnixDay); err != nil {
			return nil, core.Err("Error en el día", run.DayStartTime(targetUnixDay).Format("2006-01-02"), ":", err)
		}
	}
	return historicalDates, nil
}

func (run *Run) runOneDay(targetUnixDay int16) error {
	dayStart := run.DayStartTime(targetUnixDay)
	core.Log("Seed:: día", targetUnixDay, dayStart.Format("2006-01-02"))

	for stepIndex, dayStep := range run.daySteps {
		// Finished before the interruption; its own fine-grained checkpoint already moved on.
		if stepIndex < run.State.CompletedDaySteps {
			continue
		}
		if err := dayStep.Run(dayStart); err != nil {
			return core.Err("paso", dayStep.Name, ":", err)
		}
		run.State.CompletedDaySteps = stepIndex + 1
		if err := run.SaveState(); err != nil {
			return err
		}
	}

	if !slices.Contains(run.State.AffectedUnixDays, targetUnixDay) {
		run.State.AffectedUnixDays = append(run.State.AffectedUnixDays, targetUnixDay)
	}
	return run.SaveState()
}

// resolvePendingDates returns the days still to process, oldest first. The day recorded in the
// state is kept in the list rather than skipped: it may be half done, and runOneDay knows how far
// it got.
func (run *Run) resolvePendingDates() ([]int16, error) {
	currentUnixDay := core.TimeToFechaUnix(run.RealNow)
	historicalDates := make([]int16, 0, run.Config.Days)
	for dayOffset := run.Config.Days - 1; dayOffset >= 0; dayOffset-- {
		historicalDates = append(historicalDates, currentUnixDay-int16(dayOffset))
	}

	resumeUnixDay := run.State.UnixDay
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
