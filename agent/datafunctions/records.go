package datafunctions

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	business "app/business/types"
	crm "app/crm/types"
	"app/db"
	production "app/production/types"
	sales "app/sales/types"
)

// A record name the user wrote ("leche gloria") is looked up in its collection's text search: a
// confident match resolves it, a name nothing matches is an error for the model, and an ambiguous one
// becomes a Choice for the user. The sales tool pauses its Eino turn with it; a data function returns
// it as its Result and the pick comes back as "#<id>", which never resolves again.

const (
	// resolveSearchLimit is how many text-search hits a name's decision reads.
	resolveSearchLimit = 20
	// maxQuestionOptions keeps a choice question to 5 records, plus "Include all".
	maxQuestionOptions = 5
	// IncludeAllChoiceID is the choice that takes every record offered.
	IncludeAllChoiceID = "all"
)

// pickedRecordPattern is a record the user already picked: "#13".
var pickedRecordPattern = regexp.MustCompile(`^#(\d{1,9})$`)

// RecordMatch is one active record a name may stand for.
type RecordMatch struct {
	ID   int32
	Name string
	// Detail tells homonyms apart in the question (a client's RUC/DNI).
	Detail string
}

// RecordCollection finds and names the records of one FrameSQL record column.
type RecordCollection struct {
	// Noun names one record in a question, "English|Spanish".
	Noun string
	// Search returns the active records matching a name, best first.
	Search func(companyID int32, name string) ([]RecordMatch, error)
	// NamesByID names a result's IDs, deleted records included: a past sale still shows its product.
	NamesByID func(companyID int32, ids []int32) (map[int32]string, error)
}

// DatabaseRecordCollections are the sale frames' record collections read from the database; tests and
// genix-local-ai pass collections in memory instead.
func DatabaseRecordCollections() map[string]RecordCollection {
	return map[string]RecordCollection{
		sales.FrameSQLCollectionProducts:   {Noun: "product|producto", Search: searchProducts, NamesByID: productNamesByID},
		sales.FrameSQLCollectionClients:    {Noun: "client|cliente", Search: searchClients, NamesByID: clientNamesByID},
		sales.FrameSQLCollectionCategories: {Noun: "category|categoría", Search: searchCategories, NamesByID: categoryNamesByID},
		sales.FrameSQLCollectionSites:      {Noun: "site|sede", Search: searchSites, NamesByID: siteNamesByID},
	}
}

func searchProducts(companyID int32, name string) ([]RecordMatch, error) {
	products := []production.Product{}
	if _, err := db.SearchText(&products, companyID, name, resolveSearchLimit, 0); err != nil {
		return nil, fmt.Errorf("search products for %q: %w", name, err)
	}
	matches := make([]RecordMatch, len(products))
	for i, product := range products {
		matches[i] = RecordMatch{ID: product.ID, Name: product.Name}
	}
	return matches, nil
}

// searchClients reads the ClientProvider index (D10: Name + RegistryNumber), which holds providers
// too: only clients are sale records.
func searchClients(companyID int32, name string) ([]RecordMatch, error) {
	clientsAndProviders := []crm.ClientProvider{}
	if _, err := db.SearchText(&clientsAndProviders, companyID, name, resolveSearchLimit, 0); err != nil {
		return nil, fmt.Errorf("search clients for %q: %w", name, err)
	}
	matches := []RecordMatch{}
	for _, clientProvider := range clientsAndProviders {
		if clientProvider.Type == crm.ClientProviderTypeClient {
			matches = append(matches, RecordMatch{ID: clientProvider.ID, Name: clientProvider.Name, Detail: clientProvider.RegistryNumber})
		}
	}
	return matches, nil
}

// searchCategories offers every active product category: they have no text search, and a company has
// few, so classifyMatches picks among them all.
func searchCategories(companyID int32, _ string) ([]RecordMatch, error) {
	categories := []business.SharedListRecord{}
	query := db.Query(&categories)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).ListID.Equals(business.SharedListCategoryID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("read the product categories: %w", err)
	}
	matches := make([]RecordMatch, len(categories))
	for i, category := range categories {
		matches[i] = RecordMatch{ID: category.ID, Name: category.Name}
	}
	return matches, nil
}

func categoryNamesByID(companyID int32, ids []int32) (map[int32]string, error) {
	categories := []business.SharedListRecord{}
	query := db.Query(&categories)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).ID.In(ids...)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("read category names: %w", err)
	}
	names := make(map[int32]string, len(categories))
	for _, category := range categories {
		names[category.ID] = category.Name
	}
	return names, nil
}

// searchSites offers every active site (sede): they have no text search, and a company has few, so
// classifyMatches picks among them all.
func searchSites(companyID int32, _ string) ([]RecordMatch, error) {
	sites := []business.Site{}
	query := db.Query(&sites)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("read the sites: %w", err)
	}
	matches := make([]RecordMatch, len(sites))
	for i, site := range sites {
		matches[i] = RecordMatch{ID: site.ID, Name: site.Name}
	}
	return matches, nil
}

func siteNamesByID(companyID int32, ids []int32) (map[int32]string, error) {
	sites := []business.Site{}
	query := db.Query(&sites)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).ID.In(ids...)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("read site names: %w", err)
	}
	names := make(map[int32]string, len(sites))
	for _, site := range sites {
		names[site.ID] = site.Name
	}
	return names, nil
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

// Choice is a question for the user: which record a name stands for. The chat shows Options as
// buttons (harness.Choice has the same fields); the browser echoes the picked ID back, and it is
// checked against Options, never trusted as is.
type Choice struct {
	// Argument is the data function's argument holding the name ("products"); empty in the sales tool.
	Argument string
	// Name is the name as written.
	Name     string
	Question string
	Options  []ChoiceOption
}

type ChoiceOption struct {
	// ID is a record ID, or IncludeAllChoiceID.
	ID    string
	Label string
}

// PickedIDs reads the user's answer to a choice: the picked record, or every offered record.
func PickedIDs(options []ChoiceOption, answer string) ([]int32, bool) {
	offeredIDs := []int32{}
	for _, option := range options {
		if id, err := strconv.ParseInt(option.ID, 10, 32); err == nil {
			offeredIDs = append(offeredIDs, int32(id))
			if option.ID == answer {
				return []int32{int32(id)}, true
			}
		}
	}
	if answer == IncludeAllChoiceID && slices.ContainsFunc(options, func(option ChoiceOption) bool { return option.ID == IncludeAllChoiceID }) {
		return offeredIDs, true
	}
	return nil, false
}

// PickedRecords is the answer as a data function's call writes it in place of the name: "#13", or
// every offered record for "Incluir todos".
func (choice Choice) PickedRecords(answer string) ([]string, bool) {
	ids, isOffered := PickedIDs(choice.Options, answer)
	records := make([]string, len(ids))
	for i, id := range ids {
		records[i] = "#" + strconv.Itoa(int(id))
	}
	return records, isOffered
}

// Summary is the tool message a paused turn keeps, so a typed reply ("la de 1 litro") can name the
// record: "Pregunta | ¿Qué producto es «leche»? | #12 Leche Gloria 400 g; #13 Leche Gloria 1 L".
func (choice Choice) Summary() string {
	candidates := []string{}
	for _, option := range choice.Options {
		if option.ID != IncludeAllChoiceID {
			candidates = append(candidates, "#"+option.ID+" "+option.Label)
		}
	}
	return "Pregunta | " + choice.Question + " | " + strings.Join(candidates, "; ")
}

// NeedsChoiceError stops a run whose name lookup must ask the user. Key identifies the name
// (collection:normalized words) for the sales tool's picks.
type NeedsChoiceError struct {
	Key    string
	Choice Choice
}

func (err *NeedsChoiceError) Error() string { return "asking the user: " + err.Choice.Question }

// ResolveNames turns the names of one record column into IDs: "#<id>" is a record already picked, a
// pick of the sales tool's state comes next, then a confident search, otherwise one question (the
// first ambiguous name; the resumed call asks the next). A name nothing matches fails back to the
// model before anything is asked.
func ResolveNames(collections map[string]RecordCollection, companyID int32, language string, collectionName string, names []string,
	picks map[string][]int32) ([]int32, error) {

	collection, isKnown := collections[collectionName]
	if !isKnown {
		return nil, fmt.Errorf("the column's records (%s) can't be looked up by name: filter it by ID", collectionName)
	}
	englishNoun, spanishNoun, _ := strings.Cut(collection.Noun, "|")
	resolvedIDs := []int32{}
	unmatchedNames := []string{}
	pickedIDs := []int32{}
	var firstQuestion *NeedsChoiceError
	for _, name := range names {
		if match := pickedRecordPattern.FindStringSubmatch(strings.TrimSpace(name)); match != nil {
			id, _ := strconv.ParseInt(match[1], 10, 32)
			pickedIDs = append(pickedIDs, int32(id))
			resolvedIDs = append(resolvedIDs, int32(id))
			continue
		}
		queryWords := normalizedWords(name)
		if len(queryWords) == 0 {
			return nil, fmt.Errorf(Localized(language, "el nombre de %s %q no tiene ninguna palabra que buscar", "the %s name %q has no searchable word"),
				Localized(language, spanishNoun, englishNoun), name)
		}
		key := collectionName + ":" + strings.Join(queryWords, " ")
		if statePicks, isPicked := picks[key]; isPicked {
			resolvedIDs = append(resolvedIDs, statePicks...)
			continue
		}
		matches, err := collection.Search(companyID, name)
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
			firstQuestion = &NeedsChoiceError{Key: key, Choice: choiceQuestion(language, name, spanishNoun, englishNoun, decision)}
		}
	}
	// A picked ID must still name a record of the company: a model can write any number.
	if len(pickedIDs) > 0 {
		pickedNames, err := collection.NamesByID(companyID, pickedIDs)
		if err != nil {
			return nil, err
		}
		for _, id := range pickedIDs {
			if pickedNames[id] == "" {
				unmatchedNames = append(unmatchedNames, "#"+strconv.Itoa(int(id)))
			}
		}
	}
	if len(unmatchedNames) > 0 {
		return nil, fmt.Errorf(Localized(language,
			"no hay %s con el nombre «%s»: la búsqueda ya ignora mayúsculas, tildes y un error de tipeo, así que díselo al usuario y pídele revisar el nombre",
			"no %s matches %q: the search already ignores case, accents and a typo, so tell the user and ask them to check the name"),
			Localized(language, spanishNoun, englishNoun), strings.Join(unmatchedNames, Localized(language, "», «", `", "`)))
	}
	if firstQuestion != nil {
		return nil, firstQuestion
	}
	return resolvedIDs, nil
}

func choiceQuestion(language, name, spanishNoun, englishNoun string, decision matchDecision) Choice {
	choice := Choice{Name: name, Question: Localized(language, "¿Qué "+spanishNoun+" es «"+name+"»?", "Which "+englishNoun+" is “"+name+"”?")}
	if decision.kind == matchSuggested {
		choice.Question = Localized(language, "No encontré «"+name+"». ¿Quisiste decir «"+decision.options[0].Name+"»?",
			"Nothing matches “"+name+"”. Did you mean “"+decision.options[0].Name+"”?")
	}
	for _, option := range decision.options {
		label := option.Name
		if option.Detail != "" {
			label += " · " + option.Detail
		}
		choice.Options = append(choice.Options, ChoiceOption{ID: strconv.Itoa(int(option.ID)), Label: label})
	}
	if len(decision.options) > 1 {
		choice.Options = append(choice.Options, ChoiceOption{ID: IncludeAllChoiceID, Label: Localized(language, "Incluir todos", "Include all")})
	}
	return choice
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
	options      []RecordMatch
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
func classifyMatches(name string, matches []RecordMatch) matchDecision {
	queryWords := normalizedWords(name)
	bestMatchedCount := 0
	matchedCountByID := map[int32]int{}
	exactNameIDs := []int32{}
	fullMatches := []RecordMatch{}
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
