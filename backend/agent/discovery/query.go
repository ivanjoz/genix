package discovery

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	entityIntroductionPattern = `(?:\b(?:llamad[oa]s?|named|called)\s+|\bque\s+se\s+llame\s+|\bde\s+nombre\s+)?`
	numericValueClausePattern = regexp.MustCompile(`(?i)\s+(?:(?:a|por)\s+|con\s+(?:precio|price|stock|cantidad|quantity)\s+(?:de|of|at)?\s*)(?:s/\.?\s*)?\d+(?:[.,]\d+)?(?:\s+(?:soles?|d[oó]lares?|euros?|unidades?|units?|usd|pen))?`)
)

// normalizeDocumentationSearch keeps RAG focused on Genix capabilities. The
// original request and agent-tool query retain instance values for execution.
func normalizeDocumentationSearch(plan Plan) Plan {
	decision := plan.Searches.DocumentationNavigation
	if !decision.Needed {
		return plan
	}

	query := strings.TrimSpace(decision.Query)
	for _, entity := range plan.Entities {
		entityName := strings.TrimSpace(entity.Name)
		if entityName == "" {
			continue
		}
		pattern := regexp.MustCompile(`(?i)` + entityIntroductionPattern + `["“”']?\s*` + regexp.QuoteMeta(entityName) + `\s*["“”']?`)
		query = pattern.ReplaceAllString(query, " ")
	}
	query = numericValueClausePattern.ReplaceAllString(query, " ")
	query = removeNumericTokens(query)
	query = strings.Join(strings.Fields(strings.Trim(query, " \t\r\n,.;:-–—")), " ")
	if query == "" {
		query = fallbackDocumentationQuery(plan)
	}
	plan.Searches.DocumentationNavigation.Query = query
	return plan
}

func removeNumericTokens(query string) string {
	words := strings.Fields(query)
	genericWords := make([]string, 0, len(words))
	for _, word := range words {
		if strings.IndexFunc(word, unicode.IsDigit) < 0 {
			genericWords = append(genericWords, word)
		}
	}
	return strings.Join(genericWords, " ")
}

func fallbackDocumentationQuery(plan Plan) string {
	query := strings.TrimSpace(string(plan.Operation) + " " + plan.Domain)
	if query == "" {
		query = string(plan.Goal)
	}
	return query
}
