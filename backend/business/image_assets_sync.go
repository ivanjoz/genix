package business

import (
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	textsearch "app/libs/text-search"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const imageAssetsRawBaseURL = "https://raw.githubusercontent.com/ivanjoz/genix-assets/main/docs/images"
const imageAssetCategoryGroupID int8 = 1

var imageAssetCategoryPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type imageAssetCategorySummary struct {
	Name  string
	MaxID int32
}

type ImageAssetSyncResult struct {
	CategoriesInserted int
	CategoriesFetched  int
	CategoriesSkipped  int
	RecordsInserted    int
}

type imageAssetTextFetcher func(url string) (string, error)

// SyncImageAssets imports only categories whose repository watermark is ahead of ScyllaDB.
func SyncImageAssets() (ImageAssetSyncResult, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	return syncImageAssets(fetchImageAssetText(httpClient))
}

func fetchImageAssetText(httpClient *http.Client) imageAssetTextFetcher {
	return func(url string) (string, error) {
		response, err := httpClient.Get(url)
		if err != nil {
			return "", fmt.Errorf("fetch %s: %w", url, err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, response.Body)
			return "", fmt.Errorf("fetch %s: HTTP %d", url, response.StatusCode)
		}
		content, err := io.ReadAll(response.Body)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", url, err)
		}
		return string(content), nil
	}
}

func syncImageAssets(fetchText imageAssetTextFetcher) (ImageAssetSyncResult, error) {
	result := ImageAssetSyncResult{}
	summaryContent, err := fetchText(imageAssetsRawBaseURL + "/SUMMARY.md")
	if err != nil {
		return result, err
	}
	categorySummaries, err := parseImageAssetSummary(summaryContent)
	if err != nil {
		return result, err
	}
	categoryByName, categoriesInserted, err := syncImageAssetCategories(categorySummaries)
	if err != nil {
		return result, err
	}
	result.CategoriesInserted = categoriesInserted

	// Parse every changed list before writing image rows; category rows are intentionally created first.
	recordsToInsert := []businessTypes.ImageAsset{}
	categoriesToUpdate := []businessTypes.ImageAssetCategory{}
	categoryByImageID := map[int32]string{}
	updated := core.SUnixTime()
	for _, categorySummary := range categorySummaries {
		category := categoryByName[categorySummary.Name]
		if category.ID <= 0 {
			return result, fmt.Errorf("category %q has no assigned database ID", categorySummary.Name)
		}
		storedMaxID := category.MaxID
		if categorySummary.MaxID <= storedMaxID {
			result.CategoriesSkipped++
			continue
		}

		listURL := fmt.Sprintf("%s/%s/IMAGES_LIST.ES.md", imageAssetsRawBaseURL, categorySummary.Name)
		listContent, fetchErr := fetchText(listURL)
		if fetchErr != nil {
			return result, fetchErr
		}
		categoryRecords, parseErr := parseImageAssetList(categorySummary, category.ID, storedMaxID, updated, listContent)
		if parseErr != nil {
			return result, parseErr
		}
		for _, record := range categoryRecords {
			if previousCategory, duplicated := categoryByImageID[record.ID]; duplicated {
				return result, fmt.Errorf("image ID %d is duplicated in categories %q and %q", record.ID, previousCategory, categorySummary.Name)
			}
			categoryByImageID[record.ID] = categorySummary.Name
			recordsToInsert = append(recordsToInsert, record)
		}
		category.MaxID = categorySummary.MaxID
		category.Updated = updated
		categoriesToUpdate = append(categoriesToUpdate, category)
		result.CategoriesFetched++
		core.Log("SyncImageAssets parsed category:", categorySummary.Name, "| stored max:", storedMaxID, "| remote max:", categorySummary.MaxID, "| new records:", len(categoryRecords))
	}

	if len(recordsToInsert) == 0 {
		core.Log("SyncImageAssets no new records:", "categories skipped", result.CategoriesSkipped)
		return result, nil
	}
	if err := db.Insert(&recordsToInsert); err != nil {
		return result, fmt.Errorf("insert image assets: %w", err)
	}
	categoryTable := db.Table[businessTypes.ImageAssetCategory]()
	if err := db.Update(&categoriesToUpdate, categoryTable.MaxID, categoryTable.Updated); err != nil {
		return result, fmt.Errorf("update image asset category watermarks: %w", err)
	}
	result.RecordsInserted = len(recordsToInsert)
	core.Log("SyncImageAssets completed:", "categories inserted", result.CategoriesInserted, "| categories fetched", result.CategoriesFetched, "| categories skipped:", result.CategoriesSkipped, "| records inserted:", result.RecordsInserted)
	return result, nil
}

func syncImageAssetCategories(categorySummaries []imageAssetCategorySummary) (map[string]businessTypes.ImageAssetCategory, int, error) {
	storedCategories := []businessTypes.ImageAssetCategory{}
	query := db.Query(&storedCategories)
	if err := query.GroupID.Equals(imageAssetCategoryGroupID).Exec(); err != nil {
		return nil, 0, fmt.Errorf("query image asset categories: %w", err)
	}

	categoryByName := make(map[string]businessTypes.ImageAssetCategory, len(categorySummaries))
	for _, category := range storedCategories {
		if category.ID <= 0 || !imageAssetCategoryPattern.MatchString(category.Name) {
			return nil, 0, fmt.Errorf("invalid stored image asset category: ID=%d name=%q", category.ID, category.Name)
		}
		if _, duplicated := categoryByName[category.Name]; duplicated {
			return nil, 0, fmt.Errorf("duplicated stored image asset category name %q", category.Name)
		}
		categoryByName[category.Name] = category
	}

	updated := core.SUnixTime()
	categoriesToInsert := []businessTypes.ImageAssetCategory{}
	for _, categorySummary := range categorySummaries {
		if categoryByName[categorySummary.Name].ID > 0 {
			continue
		}
		categoriesToInsert = append(categoriesToInsert, businessTypes.ImageAssetCategory{
			GroupID: imageAssetCategoryGroupID,
			Name:    categorySummary.Name,
			Updated: updated,
		})
	}
	if len(categoriesToInsert) == 0 {
		core.Log("SyncImageAssets categories already synchronized:", len(categoryByName))
		return categoryByName, 0, nil
	}

	core.Log("SyncImageAssets inserting categories:", len(categoriesToInsert))
	if err := db.Insert(&categoriesToInsert); err != nil {
		return nil, 0, fmt.Errorf("insert image asset categories: %w", err)
	}
	for _, category := range categoriesToInsert {
		// A non-positive int16 indicates sequence exhaustion or failed assignment.
		if category.ID <= 0 {
			return nil, 0, fmt.Errorf("category %q received invalid autoincrement ID %d", category.Name, category.ID)
		}
		categoryByName[category.Name] = category
		core.Log("SyncImageAssets category inserted:", category.Name, "| ID:", category.ID)
	}
	return categoryByName, len(categoriesToInsert), nil
}

func parseImageAssetSummary(content string) ([]imageAssetCategorySummary, error) {
	categorySummaries := []imageAssetCategorySummary{}
	categoryNames := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		categoryName, maxIDText, found := strings.Cut(line, ":")
		categoryName = strings.TrimSpace(categoryName)
		if !found || !imageAssetCategoryPattern.MatchString(categoryName) {
			return nil, fmt.Errorf("SUMMARY.md line %d has an invalid category: %q", lineNumber, line)
		}
		maxIDValue, err := strconv.ParseInt(strings.TrimSpace(maxIDText), 10, 32)
		if err != nil || maxIDValue < 0 {
			return nil, fmt.Errorf("SUMMARY.md line %d has an invalid maximum ID: %q", lineNumber, maxIDText)
		}
		if categoryNames[categoryName] {
			return nil, fmt.Errorf("SUMMARY.md line %d duplicates category %q", lineNumber, categoryName)
		}
		categoryNames[categoryName] = true
		categorySummaries = append(categorySummaries, imageAssetCategorySummary{Name: categoryName, MaxID: int32(maxIDValue)})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan SUMMARY.md: %w", err)
	}
	if len(categorySummaries) == 0 {
		return nil, fmt.Errorf("SUMMARY.md contains no categories")
	}
	return categorySummaries, nil
}

func parseImageAssetList(
	categorySummary imageAssetCategorySummary,
	categoryID int16,
	storedMaxID int32,
	updated int32,
	content string,
) ([]businessTypes.ImageAsset, error) {
	records := []businessTypes.ImageAsset{}
	seenIDs := map[int32]bool{}
	parsedMaxID := int32(0)
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells, err := parseMarkdownTableRow(line)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %w", categorySummary.Name, lineNumber, err)
		}
		if len(cells) == 0 || cells[0] == "Nombre" || isMarkdownSeparatorRow(cells) {
			continue
		}
		if len(cells) != 7 {
			return nil, fmt.Errorf("%s line %d: expected 7 columns, found %d", categorySummary.Name, lineNumber, len(cells))
		}
		imageIDValue, err := strconv.ParseInt(cells[0], 10, 32)
		if err != nil || imageIDValue <= 0 {
			return nil, fmt.Errorf("%s line %d: invalid image ID %q", categorySummary.Name, lineNumber, cells[0])
		}
		imageID := int32(imageIDValue)
		if seenIDs[imageID] {
			return nil, fmt.Errorf("%s line %d: duplicated image ID %d", categorySummary.Name, lineNumber, imageID)
		}
		seenIDs[imageID] = true
		parsedMaxID = max(parsedMaxID, imageID)
		if imageID <= storedMaxID {
			continue
		}

		description := strings.TrimSpace(cells[1])
		objects := splitImageAssetObjects(cells[2])
		if description == "" || len(objects) == 0 {
			return nil, fmt.Errorf("%s line %d: description and objects are required", categorySummary.Name, lineNumber)
		}
		// Search signatures represent the concrete objects listed in Elementos.
		searchText := strings.Join(objects, " ")
		records = append(records, businessTypes.ImageAsset{
			GroupID:     imageAssetCategoryGroupID,
			ID:          imageID,
			CategoryID:  categoryID,
			Description: description,
			Objects:     objects,
			Bigrams:     imageAssetBigramsToInt8(textsearch.EncodeTextBigrams(searchText)),
			Updated:     updated,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s image list: %w", categorySummary.Name, err)
	}
	if parsedMaxID != categorySummary.MaxID {
		return nil, fmt.Errorf("%s maximum ID mismatch: SUMMARY.md=%d list=%d", categorySummary.Name, categorySummary.MaxID, parsedMaxID)
	}
	slices.SortFunc(records, func(left, right businessTypes.ImageAsset) int {
		return int(left.ID - right.ID)
	})
	return records, nil
}

func parseMarkdownTableRow(line string) ([]string, error) {
	trimmedLine := strings.TrimSpace(line)
	if len(trimmedLine) < 2 || trimmedLine[0] != '|' || trimmedLine[len(trimmedLine)-1] != '|' {
		return nil, fmt.Errorf("invalid Markdown table row")
	}

	cells := []string{}
	var cell strings.Builder
	escaped := false
	for _, character := range trimmedLine[1 : len(trimmedLine)-1] {
		switch {
		case escaped:
			cell.WriteRune(character)
			escaped = false
		case character == '\\':
			escaped = true
		case character == '|':
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteRune(character)
		}
	}
	if escaped {
		cell.WriteByte('\\')
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells, nil
}

func isMarkdownSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		separator := strings.Trim(strings.TrimSpace(cell), ":")
		if len(separator) < 3 || strings.Trim(separator, "-") != "" {
			return false
		}
	}
	return true
}

func splitImageAssetObjects(objectsText string) []string {
	objects := []string{}
	for _, object := range strings.Split(objectsText, ",") {
		trimmedObject := strings.TrimSpace(object)
		if trimmedObject != "" {
			objects = append(objects, trimmedObject)
		}
	}
	return objects
}

func imageAssetBigramsToInt8(encodedBigrams []uint8) []int8 {
	if len(encodedBigrams) == 0 {
		return nil
	}
	storedBigrams := make([]int8, len(encodedBigrams))
	for index, encodedBigram := range encodedBigrams {
		storedBigrams[index] = int8(encodedBigram)
	}
	return storedBigrams
}
