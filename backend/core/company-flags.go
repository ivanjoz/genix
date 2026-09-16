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
// It sits in core because the flags gate rules in several modules (sales refuses a sale, logistics
// refuses an express entry) and core is the only package all of them may import.
type companyFlagsCatalog struct {
	flagNameByID map[int16]string
	loadErr      error
}

var embeddedCompanyFlags = &companyFlagsCatalog{}

// LoadEmbeddedCompanyFlags parses the catalog the main package embeds. Called once at startup.
func LoadEmbeddedCompanyFlags(companyFlagsContent []byte) {
	// Sections mix two kinds of key — `label` (a string) and the flag ids (inline tables) — so the
	// value type has to be `any` and the shape is checked below instead of by the unmarshaller.
	parsedSections := map[string]map[string]any{}
	if err := toml.Unmarshal(companyFlagsContent, &parsedSections); err != nil {
		embeddedCompanyFlags.loadErr = Err("company_flags.toml: no se pudo parsear:", err.Error())
		return
	}

	flagNameByID := map[int16]string{}
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

			// Ids are one flat namespace: the company saves them as a single list, so the same id in
			// two sections would make the stored value mean two different rules.
			if _, isDuplicated := flagNameByID[int16(flagID)]; isDuplicated {
				embeddedCompanyFlags.loadErr = Err("company_flags.toml: el id", flagID,
					"está declarado más de una vez; los ids son únicos entre secciones.")
				return
			}

			flagNameByID[int16(flagID)] = flagName
		}
	}

	embeddedCompanyFlags.flagNameByID = flagNameByID
	embeddedCompanyFlags.loadErr = nil
}

// GetCompanyFlagName resolves a flag id to its "English|Español" name.
func GetCompanyFlagName(flagID int16) (string, bool) {
	flagName, flagFound := embeddedCompanyFlags.flagNameByID[flagID]
	return flagName, flagFound
}

// SanitizeCompanyFlags is what a handler must run over the flags a client sends: it drops repeats,
// sorts, and refuses any id the catalog does not declare. An unknown id is not harmless — it would
// be stored and handed back to a browser whose catalog would show the company nothing for it, so
// the company would carry a rule nobody can see or uncheck.
func SanitizeCompanyFlags(flags []int16) ([]int16, error) {
	if embeddedCompanyFlags.loadErr != nil {
		return nil, embeddedCompanyFlags.loadErr
	}
	if embeddedCompanyFlags.flagNameByID == nil {
		return nil, Err("SanitizeCompanyFlags:: LoadEmbeddedCompanyFlags no se ejecutó.")
	}

	sanitizedFlags := []int16{}
	for _, flagID := range flags {
		if _, flagExists := embeddedCompanyFlags.flagNameByID[flagID]; !flagExists {
			return nil, Err("El flag", flagID, "no existe en el catálogo de flags de la empresa.")
		}
		if !slices.Contains(sanitizedFlags, flagID) {
			sanitizedFlags = append(sanitizedFlags, flagID)
		}
	}

	slices.Sort(sanitizedFlags)
	return sanitizedFlags, nil
}

// HasCompanyFlag is how a business rule asks the question: `core.HasCompanyFlag(company.Flags, 2)`.
func HasCompanyFlag(companyFlags []int16, flagID int16) bool {
	return slices.Contains(companyFlags, flagID)
}
