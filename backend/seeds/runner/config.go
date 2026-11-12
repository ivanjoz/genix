package runner

import (
	"app/core"
	"strconv"
	"strings"
)

// Config is the full argument surface of a run; every field has a default so a run needs no flags.
// One surface for every module, so the same command line resumes the same plan.
type Config struct {
	Days int

	// logistics + sales
	ProductPoolSize       int
	PurchaseOrderProducts int
	LotProducts           int
	SerialProducts        int
	SalesMin              int
	SalesMax              int
	SaleLinesMin          int
	SaleLinesMax          int
	UnpaidMin             int
	UnpaidMax             int
	UndeliveredMin        int
	UndeliveredMax        int
	AnnulMin              int
	AnnulMax              int
	WarehouseIDs          []int32

	// base catalog and logistics replenishment settings
	SupplyProducts      int
	ProvidersPerProduct int
	MinStockMin         int
	MinStockMax         int
	SalesPerDayMin      int
	SalesPerDayMax      int
	SupplyMaterials     int
	// CatalogProducts caps the products.xlsx import; 0 imports the whole workbook (the load test caps it).
	CatalogProducts int

	// accounting
	Assets int

	// projects
	ProjectsCount     int
	ProjectSprints    int
	ProjectStories    int
	ProjectSprintDays int

	Seed   int64
	Reset  bool
	DryRun bool
	// SkipConfirm (--yes) skips the pre-flight confirmation, for runs started without a terminal.
	SkipConfirm bool
}

// DefaultConfig is a two-store Lima minimarket over its last 60 days.
func DefaultConfig() Config {
	return Config{
		Days:                  60,
		ProductPoolSize:       400,
		PurchaseOrderProducts: 150, // 200–400 tickets of 3–8 lines drain ~5,000 units a day; two orders restock it
		LotProducts:           15,
		SerialProducts:        10,
		SalesMin:              200,
		SalesMax:              400,
		SaleLinesMin:          3,
		SaleLinesMax:          8,
		UnpaidMin:             5,
		UnpaidMax:             20,
		UndeliveredMin:        5,
		UndeliveredMax:        20,
		AnnulMin:              2,
		AnnulMax:              5,
		WarehouseIDs:          []int32{1, 2},
		SupplyProducts:        2000,
		ProvidersPerProduct:   2,
		MinStockMin:           2,
		MinStockMax:           50,
		SalesPerDayMax:        20,
		SupplyMaterials:       100,
		Assets:                100,
		ProjectsCount:         2,
		ProjectSprints:        4,
		ProjectStories:        20,
		ProjectSprintDays:     14,
		Seed:                  DefaultSeed,
	}
}

// ParseArgs applies --flag=value arguments over config.
func ParseArgs(arguments []string, config Config) (Config, error) {
	intFlags := map[string]*int{
		"days": &config.Days, "products": &config.ProductPoolSize, "po-products": &config.PurchaseOrderProducts,
		"lot-products": &config.LotProducts, "serial-products": &config.SerialProducts,
		"sales-min": &config.SalesMin, "sales-max": &config.SalesMax,
		"sale-lines-min": &config.SaleLinesMin, "sale-lines-max": &config.SaleLinesMax,
		"unpaid-min": &config.UnpaidMin, "unpaid-max": &config.UnpaidMax,
		"undelivered-min": &config.UndeliveredMin, "undelivered-max": &config.UndeliveredMax,
		"annul-min": &config.AnnulMin, "annul-max": &config.AnnulMax,
		"supply-products": &config.SupplyProducts, "providers-per-product": &config.ProvidersPerProduct,
		"min-stock-min": &config.MinStockMin, "min-stock-max": &config.MinStockMax,
		"sales-per-day-min": &config.SalesPerDayMin, "sales-per-day-max": &config.SalesPerDayMax,
		"supply-materials": &config.SupplyMaterials, "assets": &config.Assets,
		"projects": &config.ProjectsCount, "sprints": &config.ProjectSprints,
		"stories": &config.ProjectStories, "sprint-days": &config.ProjectSprintDays,
	}
	for _, argument := range arguments {
		name, rawValue, hasValue := strings.Cut(strings.TrimPrefix(argument, "--"), "=")
		switch name {
		case "reset":
			config.Reset = true
			continue
		case "dry-run":
			config.DryRun = true
			continue
		case "yes":
			config.SkipConfirm = true
			continue
		}
		if !hasValue {
			return config, core.Err("el argumento", argument, "necesita un valor")
		}
		if name == "warehouses" {
			config.WarehouseIDs = []int32{}
			for _, rawWarehouseID := range strings.Split(rawValue, ",") {
				parsedWarehouseID, parseError := strconv.Atoi(strings.TrimSpace(rawWarehouseID))
				if parseError != nil || parsedWarehouseID <= 0 {
					return config, core.Err("almacén inválido:", rawWarehouseID)
				}
				config.WarehouseIDs = append(config.WarehouseIDs, int32(parsedWarehouseID))
			}
			continue
		}

		parsedValue, parseError := strconv.ParseInt(rawValue, 10, 64)
		if parseError != nil {
			return config, core.Err("el argumento", argument, "no es un número")
		}
		switch {
		case name == "company":
			CompanyID = int32(parsedValue)
		case name == "seed":
			config.Seed = parsedValue
		case intFlags[name] != nil:
			*intFlags[name] = int(parsedValue)
		default:
			return config, core.Err("argumento desconocido:", argument)
		}
	}

	// Fail on inconsistent combinations up front: a bad ratio only shows up hundreds of writes in.
	switch {
	case config.Days < 1:
		return config, core.Err("--days debe ser >= 1")
	case len(config.WarehouseIDs) == 0:
		return config, core.Err("--warehouses no puede quedar vacío")
	case config.PurchaseOrderProducts > config.ProductPoolSize:
		return config, core.Err("--po-products no puede superar a --products")
	case config.LotProducts+config.SerialProducts > config.PurchaseOrderProducts:
		return config, core.Err("--lot-products + --serial-products no pueden superar a --po-products")
	case config.SalesMin > config.SalesMax || config.SalesMin < 0:
		return config, core.Err("--sales-min/--sales-max inconsistentes")
	case config.SaleLinesMin < 1 || config.SaleLinesMin > config.SaleLinesMax:
		return config, core.Err("--sale-lines-min/--sale-lines-max inconsistentes")
	case config.UnpaidMin > config.UnpaidMax || config.UndeliveredMin > config.UndeliveredMax ||
		config.AnnulMin > config.AnnulMax || config.AnnulMax > config.SalesMin:
		return config, core.Err("los rangos de impagas/no entregadas/anuladas son inconsistentes")
	case config.UnpaidMax > config.SalesMin || config.UndeliveredMax > config.SalesMin:
		return config, core.Err("impagas/no entregadas no pueden superar al mínimo de ventas del día")
	case config.SupplyProducts < 1 || config.ProvidersPerProduct < 1:
		return config, core.Err("--supply-products y --providers-per-product deben ser >= 1")
	case config.MinStockMin < 0 || config.MinStockMax < config.MinStockMin:
		return config, core.Err("--min-stock-min/--min-stock-max forman un rango inválido")
	case config.SalesPerDayMin < 0 || config.SalesPerDayMax < config.SalesPerDayMin:
		return config, core.Err("--sales-per-day-min/--sales-per-day-max forman un rango inválido")
	case config.SupplyMaterials < 0 || config.Assets < 0:
		return config, core.Err("--supply-materials y --assets no pueden ser negativos")
	case config.ProjectsCount < 1 || config.ProjectsCount > 20:
		return config, core.Err("--projects debe estar entre 1 y 20")
	case config.ProjectSprints < 1 || config.ProjectSprints > 12:
		return config, core.Err("--sprints debe estar entre 1 y 12")
	case config.ProjectStories < 6 || config.ProjectStories > 200:
		return config, core.Err("--stories debe estar entre 6 y 200")
	case config.ProjectSprintDays < 5 || config.ProjectSprintDays > 28:
		return config, core.Err("--sprint-days debe estar entre 5 y 28")
	}
	return config, nil
}
