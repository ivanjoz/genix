package core

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Company flags are the per-company switches declared in company_flags.toml: a company stores the
// ids it has checked and the rules that read them live in the modules they belong to. The catalog
// is embedded at build time, so anything malformed in it is a build-time mistake and this loader
// refuses rather than repairs — same stance as the access catalog next door.
//
// A flag is of one of two kinds and never both. Without a `type` key it is a checkbox: its id is
// either in the company's Flags list or absent. With `type = "i:<decimals>"` it is a valued flag:
// its id never enters Flags, and the number the company typed is stored in FlagValues as an
// integer scaled by the declared decimals — `i:3` means 2.5 is stored as 2500.
//
// It sits in core because the flags gate rules in several modules (sales refuses a sale, logistics
// refuses an express entry) and core is the only package all of them may import.
type companyFlagsCatalog struct {
	flagByID map[int16]companyFlagDefinition
	loadErr  error
}

type companyFlagDefinition struct {
	name string
	// decimals is the scale a valued flag is stored at: the typed value times 10^decimals.
	// Only meaningful when isValued.
	decimals int
	// byteWidth bounds the stored integer — 2 means it must fit an int16, 4 an int32. It is
	// what a save is validated against, so a company can never store a number the rule that
	// reads it, or the column it lands in, could not hold.
	byteWidth int
	isValued  bool
}

var embeddedCompanyFlags = &companyFlagsCatalog{}

// CompanyFlagValue is one valued flag's number as the company record carries it. Value is already
// scaled by the flag's declared decimals, which is why nothing but the catalog may interpret it —
// GetCompanyFlagValue is how a rule reads it back as the number that was typed.
type CompanyFlagValue struct {
	ID    int16 `json:",omitempty"`
	Value int32 `json:",omitempty"`
}

// LoadEmbeddedCompanyFlags parses the catalog the main package embeds. Called once at startup.
func LoadEmbeddedCompanyFlags(companyFlagsContent []byte) {
	// Sections mix two kinds of key — `label` (a string) and the flag ids (inline tables) — so the
	// value type has to be `any` and the shape is checked below instead of by the unmarshaller.
	parsedSections := map[string]map[string]any{}
	if err := toml.Unmarshal(companyFlagsContent, &parsedSections); err != nil {
		embeddedCompanyFlags.loadErr = Err("company_flags.toml: no se pudo parsear:", err.Error())
		return
	}

	flagByID := map[int16]companyFlagDefinition{}
	for sectionName, sectionEntries := range parsedSections {
		for entryKey, entryValue := range sectionEntries {
			if entryKey == "label" {
				continue
			}

			flagID, err := strconv.ParseInt(entryKey, 10, 16)
			if err != nil {
				embeddedCompanyFlags.loadErr = Err("company_flags.toml: la sección", sectionName,
					"declara la clave", entryKey, "que no es un id de flag ni `label`.")
				return
			}

			flagEntry, isTable := entryValue.(map[string]any)
			if !isTable {
				embeddedCompanyFlags.loadErr = Err("company_flags.toml: el flag", flagID,
					"debe declararse como { name = \"...\" }.")
				return
			}

			flagName, _ := flagEntry["name"].(string)
			if strings.TrimSpace(flagName) == "" {
				embeddedCompanyFlags.loadErr = Err("company_flags.toml: el flag", flagID, "no tiene nombre.")
				return
			}

			flagDefinition := companyFlagDefinition{name: flagName}
			if declaredType, hasType := flagEntry["type"]; hasType {
				flagDefinition, err = parseValuedCompanyFlag(flagName, declaredType, flagEntry["bytes"])
				if err != nil {
					embeddedCompanyFlags.loadErr = Err("company_flags.toml: el flag", flagID, "-", err.Error())
					return
				}
			}

			// Ids are one flat namespace: the company saves them as a single list, so the same id in
			// two sections would make the stored value mean two different rules.
			if _, isDuplicated := flagByID[int16(flagID)]; isDuplicated {
				embeddedCompanyFlags.loadErr = Err("company_flags.toml: el id", flagID,
					"está declarado más de una vez; los ids son únicos entre secciones.")
				return
			}

			flagByID[int16(flagID)] = flagDefinition
		}
	}

	embeddedCompanyFlags.flagByID = flagByID
	embeddedCompanyFlags.loadErr = nil
}

// parseValuedCompanyFlag reads the `type` / `bytes` pair of a valued flag. The only type form is
// "i" or "i:<decimals>" — an integer scaled by that many decimals — and `bytes` is mandatory
// alongside it, because the width is what bounds every save and guessing it would let a company
// store a number the column cannot hold.
func parseValuedCompanyFlag(flagName string, declaredType any, declaredBytes any) (companyFlagDefinition, error) {
	typeText, isText := declaredType.(string)
	if !isText {
		return companyFlagDefinition{}, Err("`type` debe ser un texto como \"i:3\".")
	}

	integerPart, decimalsPart, hasDecimals := strings.Cut(strings.TrimSpace(typeText), ":")
	if integerPart != "i" {
		return companyFlagDefinition{}, Err("`type` =", typeText, "no es soportado; el único tipo con valor es \"i[:decimales]\".")
	}

	decimals := 0
	if hasDecimals {
		parsedDecimals, err := strconv.Atoi(decimalsPart)
		if err != nil || parsedDecimals < 0 || parsedDecimals > 9 {
			return companyFlagDefinition{}, Err("`type` =", typeText, "declara decimales inválidos; deben ser 0..9.")
		}
		decimals = parsedDecimals
	}

	byteWidth, isInteger := declaredBytes.(int64)
	if !isInteger || (byteWidth != 2 && byteWidth != 4) {
		return companyFlagDefinition{}, Err("un flag con `type` debe declarar `bytes` = 2 o 4.")
	}

	return companyFlagDefinition{
		name:      flagName,
		decimals:  decimals,
		byteWidth: int(byteWidth),
		isValued:  true,
	}, nil
}

// GetCompanyFlagName resolves a flag id to its "English|Español" name.
func GetCompanyFlagName(flagID int16) (string, bool) {
	flagDefinition, flagFound := embeddedCompanyFlags.flagByID[flagID]
	return flagDefinition.name, flagFound
}

// SanitizeCompanyFlags is what a handler must run over the flags a client sends: it drops repeats,
// sorts, and refuses any id the catalog does not declare. An unknown id is not harmless — it would
// be stored and handed back to a browser whose catalog would show the company nothing for it, so
// the company would carry a rule nobody can see or uncheck.
func SanitizeCompanyFlags(flags []int16) ([]int16, error) {
	if err := checkCompanyFlagsCatalogIsLoaded(); err != nil {
		return nil, err
	}

	sanitizedFlags := []int16{}
	for _, flagID := range flags {
		flagDefinition, flagExists := embeddedCompanyFlags.flagByID[flagID]
		if !flagExists {
			return nil, Err("El flag", flagID, "no existe en el catálogo de flags de la empresa.")
		}
		// A valued flag is its value and nothing else: letting its id into this list would give it
		// a second, silent on/off state that no screen shows and no rule reads.
		if flagDefinition.isValued {
			return nil, Err("El flag", flagID, "tiene valor y se guarda en FlagValues, no en Flags.")
		}
		if !slices.Contains(sanitizedFlags, flagID) {
			sanitizedFlags = append(sanitizedFlags, flagID)
		}
	}

	slices.Sort(sanitizedFlags)
	return sanitizedFlags, nil
}

// SanitizeCompanyFlagValues is the same gate for the valued flags: the id must be declared and must
// be a valued one, and the scaled number must fit the width the catalog declares. A zero is dropped
// rather than stored, so "the company left the field empty" and "the company typed 0" are one
// state — absent — and the rule reading it falls back to its own default in both.
func SanitizeCompanyFlagValues(flagValues []CompanyFlagValue) ([]CompanyFlagValue, error) {
	if err := checkCompanyFlagsCatalogIsLoaded(); err != nil {
		return nil, err
	}

	sanitizedValues := []CompanyFlagValue{}
	for _, flagValue := range flagValues {
		flagDefinition, flagExists := embeddedCompanyFlags.flagByID[flagValue.ID]
		if !flagExists {
			return nil, Err("El flag", flagValue.ID, "no existe en el catálogo de flags de la empresa.")
		}
		if !flagDefinition.isValued {
			return nil, Err("El flag", flagValue.ID, "no acepta un valor; es un flag de sí/no.")
		}
		if flagValue.Value == 0 {
			continue
		}
		if flagDefinition.byteWidth == 2 && (flagValue.Value < -32768 || flagValue.Value > 32767) {
			return nil, Err("El valor del flag", flagValue.ID, "está fuera de rango: con",
				flagDefinition.decimals, "decimales el máximo es 32.767 x 10^-", flagDefinition.decimals, ".")
		}
		if slices.ContainsFunc(sanitizedValues, func(stored CompanyFlagValue) bool { return stored.ID == flagValue.ID }) {
			return nil, Err("El flag", flagValue.ID, "trae más de un valor.")
		}
		sanitizedValues = append(sanitizedValues, flagValue)
	}

	slices.SortFunc(sanitizedValues, func(a, b CompanyFlagValue) int { return int(a.ID) - int(b.ID) })
	return sanitizedValues, nil
}

func checkCompanyFlagsCatalogIsLoaded() error {
	if embeddedCompanyFlags.loadErr != nil {
		return embeddedCompanyFlags.loadErr
	}
	if embeddedCompanyFlags.flagByID == nil {
		return Err("Flags de la empresa:: LoadEmbeddedCompanyFlags no se ejecutó.")
	}
	return nil
}

// HasCompanyFlag is how a business rule asks the question: `core.HasCompanyFlag(company.Flags, 2)`.
func HasCompanyFlag(companyFlags []int16, flagID int16) bool {
	return slices.Contains(companyFlags, flagID)
}

// GetCompanyFlagValue is the valued flag's counterpart: it returns the number the company typed,
// already unscaled by the decimals the catalog declares, and whether it set one at all. The raw
// int32 is never handed out — the scale is the catalog's business and a caller dividing by 1000
// itself would be a second copy of it.
func GetCompanyFlagValue(companyFlagValues []CompanyFlagValue, flagID int16) (float64, bool) {
	flagDefinition, flagExists := embeddedCompanyFlags.flagByID[flagID]
	if !flagExists || !flagDefinition.isValued {
		return 0, false
	}

	for _, flagValue := range companyFlagValues {
		if flagValue.ID != flagID {
			continue
		}
		scale := 1.0
		for range flagDefinition.decimals {
			scale *= 10
		}
		return float64(flagValue.Value) / scale, true
	}
	return 0, false
}
