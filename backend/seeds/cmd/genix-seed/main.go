// genix-seed fills a database with dummy data through the production handlers, in this process.
// It is its own binary so the production one carries no seed code or data, and so it can be
// uploaded to the server next to the database and run there. Reads the same config.toml as the
// backend (GENIX_CONFIG_FILE, ../config.toml or ./config.toml).
//
//	genix-seed run <module,...|all> [--flag=value ...]   seed modules (base is always included)
//	genix-seed list                                     the modules and what they require
//	genix-seed status [--company=N]                     the resume point an aborted run left
//	genix-seed wipe [--company=N] [--only=prefix] [--apply]
//	genix-seed loadtest [--flag=value ...]
//
// Ctrl+C (or SIGTERM from systemd) stops a run at its next checkpoint; the same command resumes it.
// A second Ctrl+C exits at once, and the resume point may then repeat the step it cut.
package main

import (
	"app/core"
	"app/db"
	"app/exec"
	invoicing "app/invoicing/types"
	salesTypes "app/sales/types"
	"app/seeds/accounting"
	"app/seeds/base"
	"app/seeds/crm"
	"app/seeds/finance"
	"app/seeds/hr"
	"app/seeds/loadtest"
	"app/seeds/logistics"
	"app/seeds/projects"
	"app/seeds/runner"
	"app/seeds/sales"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	fareward "github.com/ivanjoz/fareward/go"
	"github.com/kr/pretty"
)

// seedModules is the registry, in the order their setups depend on: base first, always.
var seedModules = []runner.Module{
	base.Module, logistics.Module, sales.Module, finance.Module, accounting.Module, hr.Module,
	projects.Module, crm.Module,
}

const usage = `usage:
  genix-seed run <module,...|all> [--flag=value ...]
  genix-seed list
  genix-seed status [--company=N]
  genix-seed wipe [--company=N] [--only=prefix] [--apply]
  genix-seed loadtest [--flag=value ...]`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(2)
	}
	command, arguments := os.Args[1], os.Args[2:]
	if command == "list" {
		printModules()
		return
	}

	bootstrap()
	ctx := cancelOnSignal()

	var err error
	switch command {
	case "run":
		err = runSeed(ctx, arguments)
	case "status":
		err = printStatus(arguments)
	case "wipe":
		err = wipeCompanyData(arguments)
	case "loadtest":
		err = loadtest.RunLoadTest(ctx, arguments)
	default:
		fmt.Println(usage)
		os.Exit(2)
	}
	if err != nil {
		core.Log("genix-seed error::", err)
		os.Exit(1)
	}
}

// bootstrap is what the backend runs before an fn- command, minus the HTTP server.
func bootstrap() {
	core.PopulateVariables()
	core.Env.LOGS_FULL = true
	core.LogHistoricalClockIfActive()
	fareward.SetLogger(core.Log)
	fareward.SetMaxInflightPerConnection(int(core.Env.FAREWARD_MAX_INFLIGHT))
	if err := core.ConfigureFareward(core.Env.FAREWARD_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
		panic("invalid fareward configuration: " + err.Error())
	}
	core.ConfigureTextSearch()
	core.ConfigureFileStore()
	db.Connect(exec.MakeConnParams())
	// The seeded documents are dated in the past: the server's sweep would send them to SUNAT.
	invoicing.EmitSweepDisabled = true
}

// cancelOnSignal cancels the context on the first SIGINT/SIGTERM, so the run stops at its next
// checkpoint, and exits on the second.
func cancelOnSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		core.Log("genix-seed:: deteniendo en el próximo punto de control (otra señal sale de inmediato)")
		cancel()
		<-signals
		core.Log("genix-seed:: salida inmediata")
		os.Exit(130)
	}()
	return ctx
}

func runSeed(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 || strings.HasPrefix(arguments[0], "--") {
		return core.Err("indica los módulos: genix-seed run <module,...|all>")
	}
	config, err := runner.ParseArgs(arguments[1:], runner.DefaultConfig())
	if err != nil {
		return err
	}
	modules, err := runner.ResolveModules(seedModules, strings.Split(arguments[0], ","))
	if err != nil {
		return err
	}
	run := runner.New(config)
	run.Context = ctx

	if err := printPlan(run, modules); err != nil {
		return err
	}
	if config.DryRun {
		return nil
	}
	if err := confirmRun(config.SkipConfirm); err != nil {
		return err
	}

	result, err := runner.Execute(run, modules)
	if err != nil {
		if run.IsAborted() {
			core.Log("genix-seed:: abortado; repite el mismo comando para reanudar")
			os.Exit(130)
		}
		return err
	}
	pretty.Println(result)
	return nil
}

// printPlan is the pre-flight banner: where the run writes, what it writes, and whether it resumes.
func printPlan(run *runner.Run, modules []runner.Module) error {
	hostname, _ := os.Hostname()
	moduleNames := []string{}
	for _, module := range modules {
		moduleNames = append(moduleNames, module.Name)
	}
	companyID := run.UserToken.CompanyID

	existingSales := []salesTypes.SaleOrder{}
	salesQuery := db.Query(&existingSales)
	salesQuery.Select(salesQuery.ID).CompanyID.Equals(companyID).Limit(1)
	if err := salesQuery.Exec(); err != nil {
		return core.Err("no se pudo consultar la base de datos:", err)
	}

	firstDay, lastDay := run.DayStartTime(run.FirstDay()), run.DayStartTime(run.FirstDay()+int16(run.Config.Days-1))
	fmt.Printf("máquina:   %s\nbase:      %s\nempresa:   %d (ya tiene ventas: %v)\nmódulos:   %s\ndías:      %d (%s → %s)\nsemilla:   %d\n",
		hostname, core.Env.DB_NAME, companyID, len(existingSales) > 0, strings.Join(moduleNames, ", "),
		run.Config.Days, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"), run.Config.Seed)

	resumeState, err := readResumeState(companyID)
	switch {
	case err != nil:
		return err
	case resumeState == nil:
		fmt.Println("estado:    corrida nueva")
	case run.Config.Reset:
		fmt.Println("estado:    --reset descarta el punto de reanudación guardado")
	default:
		fmt.Printf("estado:    reanuda en el día %s, paso %d\n", formatUnixDay(resumeState.UnixDay), resumeState.CompletedDaySteps)
	}
	return nil
}

// confirmRun asks before writing. A run without a terminal (systemd-run) must pass --yes.
func confirmRun(skipConfirm bool) error {
	if skipConfirm {
		return nil
	}
	stdinInfo, err := os.Stdin.Stat()
	if err != nil || stdinInfo.Mode()&os.ModeCharDevice == 0 {
		return core.Err("sin terminal para confirmar: agrega --yes")
	}
	fmt.Print("¿Escribir estos datos? [s/N] ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if answer = strings.ToLower(strings.TrimSpace(answer)); answer != "s" && answer != "y" {
		return core.Err("cancelado")
	}
	return nil
}

func printStatus(arguments []string) error {
	if _, err := runner.ParseArgs(arguments, runner.DefaultConfig()); err != nil {
		return err
	}
	resumeState, err := readResumeState(runner.CompanyID)
	if err != nil || resumeState == nil {
		fmt.Println("empresa", runner.CompanyID, ": sin corrida pendiente")
		return err
	}
	fmt.Printf("empresa %d: corrida pendiente\nplan:  %s\ndía:   %s, paso %d\nstats: %+v\n",
		runner.CompanyID, resumeState.ConfigSignature, formatUnixDay(resumeState.UnixDay), resumeState.CompletedDaySteps, resumeState.Stats)
	return nil
}

func formatUnixDay(unixDay int16) string {
	return time.Unix(int64(unixDay)*86400, 0).UTC().Format("2006-01-02")
}

// readResumeState reads the company's resume file, or nil when no run is pending.
func readResumeState(companyID int32) (*runner.State, error) {
	stateBytes, err := os.ReadFile(runner.StatePathFor(companyID))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	resumeState := &runner.State{}
	if err := json.Unmarshal(stateBytes, resumeState); err != nil {
		return nil, core.Err("el archivo de estado está corrupto; usa --reset:", err)
	}
	return resumeState, nil
}

func printModules() {
	for _, module := range seedModules {
		requirement := "base"
		if len(module.Requires) > 0 {
			requirement += ", " + strings.Join(module.Requires, ", ")
		}
		if module.Name == "base" {
			requirement = "-"
		}
		fmt.Printf("%-11s requiere: %s\n", module.Name, requirement)
	}
}
