package salestools

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	crm "app/crm/types"
	"app/db"
	production "app/production/types"
	sales "app/sales/types"
	"genix-agent/harness"
)

// A name in quotes in a statement's WHERE ("leche gloria") is looked up in the column's text
// search: a confident match resolves it, an ambiguous one pauses the turn with a choice question
// (harness.ChoiceQuestion), and the resumed call reads the user's pick from the tool's state.

const (
	// resolveSearchLimit is how many text-search hits a name's decision reads.
	resolveSearchLimit = 20
	// maxQuestionOptions keeps a choice question to 5 records, plus "Include all".
	maxQuestionOptions = 5
	// includeAllChoiceID is the choice that takes every record offered.
	includeAllChoiceID = "all"
)

// recordMatch is one active record a name may stand for.
type recordMatch struct {
	ID   int32
	Name string
	// Detail tells homonyms apart in the question (a client's RUC/DNI).
	Detail string
}

// recordCollection finds and names the records of one FrameSQL record column.
type recordCollection struct {
	// noun names one record in a question, "English|Spanish".
	noun string
	// search returns the active records matching a name, best first.
	search func(companyID int32, name string) ([]recordMatch, error)
	// namesByID names a result's IDs, deleted records included: a past sale still shows its product.
	namesByID func(companyID int32, ids []int32) (map[int32]string, error)
}

var recordCollections = map[string]recordCollection{
	sales.FrameSQLCollectionProducts: {noun: "product|producto", search: searchProducts, namesByID: productNamesByID},
	sales.FrameSQLCollectionClients:  {noun: "client|cliente", search: searchClients, namesByID: clientNamesByID},
}

func searchProducts(companyID int32, name string) ([]recordMatch, error) {
	products := []production.Product{}
	if _, err := db.SearchText(&products, companyID, name, resolveSearchLimit, 0); err != nil {
		return nil, fmt.Errorf("search products for %q: %w", name, err)
	}
	matches := make([]recordMatch, len(products))
	for i, product := range products {
		matches[i] = recordMatch{ID: product.ID, Name: product.Name}
	}
	return matches, nil
}

// searchClients reads the ClientProvider index (D10: Name + RegistryNumber), which holds providers
// too: only clients are sale records.
func searchClients(companyID int32, name string) ([]recordMatch, error) {
	clientsAndProviders := []crm.ClientProvider{}
	if _, err := db.SearchText(&clientsAndProviders, companyID, name, resolveSearchLimit, 0); err != nil {
		return nil, fmt.Errorf("search clients for %q: %w", name, err)
	}
	matches := []recordMatch{}
	for _, clientProvider := range clientsAndProviders {
		if clientProvider.Type == crm.ClientProviderTypeClient {
			matches = append(matches, recordMatch{ID: clientProvider.ID, Name: clientProvider.Name, Detail: clientProvider.RegistryNumber})
		}
	}
	return matches, nil
}

func productNamesByID(companyID int32, ids []int32) (map[int32]string, error) {
	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).ID.In(ids...)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("read product names: %w", err)
	}
	names := make(map[int32]string, len(products))
	for _, product := range products {
		names[product.ID] = product.Name
	}
	return names, nil
}

func clientNamesByID(companyID int32, ids []int32) (map[int32]string, error) {
	clients := []crm.ClientProvider{}
	query := db.Query(&clients)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).ID.In(ids...)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("read client names: %w", err)
	}
	names := make(map[int32]string, len(clients))
	for _, client := range clients {
		names[client.ID] = client.Name
	}
	return names, nil
}

// pendingChoice is the tool's interrupt state: the names the user already picked, and the question
// asked now. A resumed call validates the answer against Choices: the browser echoes it back, and
// is never trusted with an ID it wasn't offered.
type pendingChoice struct {
	// Picks are record IDs by name key (collection:normalized words).
	Picks   map[string][]int32
	Key     string
	Choices []harness.Choice
}

// needsChoiceError stops a run whose name lookup must ask the user. The tool turns it into Eino's
// interrupt.
type needsChoiceError struct {
	key      string
	question harness.ChoiceQuestion
}

func (err *needsChoiceError) Error() string { return "asking the user: " + err.question.Question }

// pickedIDs reads the user's answer to pending: the picked record, or every offered record.
func pickedIDs(pending pendingChoice, answer string) ([]int32, bool) {
	offeredIDs := []int32{}
	for _, choice := range pending.Choices {
		if id, err := strconv.ParseInt(choice.ID, 10, 32); err == nil {
			offeredIDs = append(offeredIDs, int32(id))
			if choice.ID == answer {
				return []int32{int32(id)}, true
			}
		}
	}
	if answer == includeAllChoiceID && slices.ContainsFunc(pending.Choices, func(choice harness.Choice) bool { return choice.ID == includeAllChoiceID }) {
		return offeredIDs, true
	}
	return nil, false
}

// resolveNames turns the names of one record column into IDs: a pick first, then a confident
// search, otherwise one question (the first ambiguous name; the resumed call asks the next). A name
// nothing matches fails back to the model before anything is asked.
func resolveNames(companyID int32, language string, collectionName string, names []string, picks map[string][]int32) ([]int32, error) {
	collection, isKnown := recordCollections[collectionName]
	if !isKnown {
		return nil, fmt.Errorf("the column's records (%s) can't be looked up by name: filter it by ID", collectionName)
	}
	englishNoun, spanishNoun, _ := strings.Cut(collection.noun, "|")
	resolvedIDs := []int32{}
	unmatchedNames := []string{}
	var firstQuestion *needsChoiceError
	for _, name := range names {
		queryWords := normalizedWords(name)
		if len(queryWords) == 0 {
			return nil, fmt.Errorf("the %s name %q has no searchable word", englishNoun, name)
		}
		key := collectionName + ":" + strings.Join(queryWords, " ")
		if pickedIDs, isPicked := picks[key]; isPicked {
			resolvedIDs = append(resolvedIDs, pickedIDs...)
			continue
		}
		matches, err := collection.search(companyID, name)
		if err != nil {
			return nil, err
		}
		decision := classifyMatches(name, matches)
		switch {
		case decision.kind == matchNotFound:
			unmatchedNames = append(unmatchedNames, name)
		case decision.kind == matchConfident:
			resolvedIDs = append(resolvedIDs, decision.confidentIDs...)
		case firstQuestion == nil:
			firstQuestion = &needsChoiceError{key: key, question: choiceQuestion(language, name, spanishNoun, englishNoun, decision)}
		}
	}
	if len(unmatchedNames) > 0 {
		return nil, fmt.Errorf("no %s matches %q: the search already ignores case, accents and a typo, so tell the user and ask them to check the name",
			englishNoun, strings.Join(unmatchedNames, `", "`))
	}
	if firstQuestion != nil {
		return nil, firstQuestion
	}
	return resolvedIDs, nil
}

func choiceQuestion(language, name, spanishNoun, englishNoun string, decision matchDecision) harness.ChoiceQuestion {
	question := harness.ChoiceQuestion{Question: localized(language, "¿Qué "+spanishNoun+" es «"+name+"»?", "Which "+englishNoun+" is “"+name+"”?")}
	if decision.kind == matchSuggested {
		question.Question = localized(language, "No encontré «"+name+"». ¿Quisiste decir «"+decision.options[0].Name+"»?",
			"Nothing matches “"+name+"”. Did you mean “"+decision.options[0].Name+"”?")
	}
	for _, option := range decision.options {
		label := option.Name
		if option.Detail != "" {
			label += " · " + option.Detail
		}
		question.Choices = append(question.Choices, harness.Choice{ID: strconv.Itoa(int(option.ID)), Label: label})
	}
	if len(decision.options) > 1 {
		question.Choices = append(question.Choices, harness.Choice{ID: includeAllChoiceID, Label: localized(language, "Incluir todos", "Include all")})
	}
	return question
}

type matchKind int8

const (
	matchNotFound matchKind = iota + 1
	matchConfident
	matchAmbiguous
	matchSuggested
)

type matchDecision struct {
	kind         matchKind
	confidentIDs []int32
	options      []recordMatch
}

// classifyMatches decides whether a name names one record. The search ranks the hits, but its score
// can't say "this is the one": the decision reads the hits' names against the words the user typed.
// A name word holds a query word when it equals it or starts with it (exact), or is one edit away
// (fuzzy, a typo the search tolerated).
//
//   - Not found: no hit, or no name holds even half of the query words, fuzzy included.
//   - Confident: exactly one name has the query's words and no other; else exactly one name holds
//     every query word exactly; else a single candidate holding every word, fuzzy included.
//   - Ambiguous: the names holding every word exactly, or else the first hits, at most 5.
//   - Suggested: a single candidate that misses a query word: ask "did you mean" instead of
//     answering about another record.
func classifyMatches(name string, matches []recordMatch) matchDecision {
	queryWords := normalizedWords(name)
	bestMatchedCount := 0
	matchedCountByID := map[int32]int{}
	exactNameIDs := []int32{}
	fullMatches := []recordMatch{}
	for _, match := range matches {
		nameWords := normalizedWords(match.Name)
		exactCount, matchedCount := 0, 0
		for _, queryWord := range queryWords {
			switch bestWordMatch(queryWord, nameWords) {
			case wordMatchExact:
				exactCount++
				matchedCount++
			case wordMatchFuzzy:
				matchedCount++
			}
		}
		bestMatchedCount = max(bestMatchedCount, matchedCount)
		matchedCountByID[match.ID] = matchedCount
		if exactCount == len(queryWords) {
			fullMatches = append(fullMatches, match)
			if sameWordSet(nameWords, queryWords) {
				exactNameIDs = append(exactNameIDs, match.ID)
			}
		}
	}

	switch {
	case len(matches) == 0 || bestMatchedCount*2 < len(queryWords):
		return matchDecision{kind: matchNotFound}
	case len(exactNameIDs) == 1:
		return matchDecision{kind: matchConfident, confidentIDs: exactNameIDs}
	case len(fullMatches) == 1:
		return matchDecision{kind: matchConfident, confidentIDs: []int32{fullMatches[0].ID}}
	}
	optionMatches := fullMatches
	if len(optionMatches) == 0 {
		// The hits holding the most query words, in the search's rank.
		for _, match := range matches {
			if matchedCountByID[match.ID] == bestMatchedCount {
				optionMatches = append(optionMatches, match)
			}
		}
	}
	if len(optionMatches) == 1 {
		if matchedCountByID[optionMatches[0].ID] == len(queryWords) {
			return matchDecision{kind: matchConfident, confidentIDs: []int32{optionMatches[0].ID}}
		}
		return matchDecision{kind: matchSuggested, options: optionMatches}
	}
	return matchDecision{kind: matchAmbiguous, options: optionMatches[:min(len(optionMatches), maxQuestionOptions)]}
}

type wordMatch int8

const (
	wordMatchNone wordMatch = iota
	wordMatchFuzzy
	wordMatchExact
)

// bestWordMatch compares one query word with a name's words. Numbers match only when equal: a
// prefix would take "1" for "10". A prefix needs 3 letters, so "l" (litro) doesn't hold "lima".
func bestWordMatch(queryWord string, nameWords []string) wordMatch {
	best := wordMatchNone
	isNumber := strings.IndexFunc(queryWord, func(character rune) bool { return !unicode.IsDigit(character) }) < 0
	for _, nameWord := range nameWords {
		switch {
		case nameWord == queryWord:
			return wordMatchExact
		case !isNumber && len(queryWord) >= 3 && strings.HasPrefix(nameWord, queryWord):
			return wordMatchExact
		case !isNumber && len(queryWord) >= 4 && withinOneEdit(queryWord, nameWord):
			best = wordMatchFuzzy
		}
	}
	return best
}

// withinOneEdit reports whether one insertion, deletion or substitution turns a into b.
func withinOneEdit(a, b string) bool {
	left, right := []rune(a), []rune(b)
	if len(left) > len(right) {
		left, right = right, left
	}
	if len(right)-len(left) > 1 {
		return false
	}
	position := 0
	for position < len(left) && left[position] == right[position] {
		position++
	}
	if len(left) == len(right) {
		return slices.Equal(left[min(position+1, len(left)):], right[min(position+1, len(right)):])
	}
	return slices.Equal(left[position:], right[position+1:])
}

func sameWordSet(leftWords, rightWords []string) bool {
	leftSet, rightSet := slices.Clone(leftWords), slices.Clone(rightWords)
	slices.Sort(leftSet)
	slices.Sort(rightSet)
	return slices.Equal(slices.Compact(leftSet), slices.Compact(rightSet))
}

// resolveStopWords never decide a match: "vinagre de manzana" is the words vinagre and manzana.
var resolveStopWords = map[string]bool{
	"a": true, "al": true, "con": true, "de": true, "del": true, "el": true, "en": true, "la": true,
	"las": true, "lo": true, "los": true, "para": true, "por": true, "sin": true, "un": true, "una": true,
	"y": true, "e": true, "o": true, "and": true, "for": true, "of": true, "the": true, "with": true,
}

// normalizedWords lowercases, folds accents (ñ stays: it is a letter of its own in Spanish names),
// splits on anything that is not a letter or digit, and drops stopwords and lone letters.
func normalizedWords(text string) []string {
	decomposed := norm.NFD.String(strings.ToLower(text))
	folded := strings.Builder{}
	for _, character := range decomposed {
		// NFD splits ñ into n + U+0303: keep that one mark and drop the others.
		if unicode.Is(unicode.Mn, character) && character != '̃' {
			continue
		}
		folded.WriteRune(character)
	}
	words := []string{}
	for _, field := range strings.FieldsFunc(norm.NFC.String(folded.String()), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsDigit(character)
	}) {
		isLoneLetter := len([]rune(field)) == 1 && !unicode.IsDigit([]rune(field)[0])
		if !resolveStopWords[field] && !isLoneLetter {
			words = append(words, field)
		}
	}
	return words
}

func localized(language, spanish, english string) string {
	if language == "en" {
		return english
	}
	return spanish
}
