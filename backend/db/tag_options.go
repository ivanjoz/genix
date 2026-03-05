package db

import (
	"fmt"
	"strings"
)

type dbTagConfig struct {
	columnName string
	options    map[string]struct{}
}

func parseDBTagConfig(dbTagRaw string) dbTagConfig {
	tagConfig := dbTagConfig{options: map[string]struct{}{}}
	if dbTagRaw == "" {
		return tagConfig
	}

	tagParts := strings.Split(dbTagRaw, ",")
	tagConfig.columnName = strings.TrimSpace(tagParts[0])

	// Normalize option flags so callers can check options in a case-insensitive way.
	for optionIndex := 1; optionIndex < len(tagParts); optionIndex++ {
		tagOption := strings.ToLower(strings.TrimSpace(tagParts[optionIndex]))
		if tagOption == "" {
			continue
		}
		tagConfig.options[tagOption] = struct{}{}
	}

	return tagConfig
}

func (tagConfig dbTagConfig) hasOption(optionName string) bool {
	_, hasOption := tagConfig.options[strings.ToLower(optionName)]
	return hasOption
}

func (tagConfig dbTagConfig) hasCollectionOptions() bool {
	return tagConfig.hasOption("set") || tagConfig.hasOption("frozen")
}

func applyCollectionTagOptions(recordTypeName string, recordFieldName string, inferredColType colType, tagConfig dbTagConfig) colType {
	if !tagConfig.hasCollectionOptions() {
		return inferredColType
	}

	if !inferredColType.IsSlice {
		panic(fmt.Sprintf(`Record "%v": field "%v" uses db collection options on non-slice type "%v".`,
			recordTypeName, recordFieldName, inferredColType.FieldType))
	}

	collectionColType := normalizeCollectionColType(inferredColType.ColType, tagConfig.hasOption("set"), tagConfig.hasOption("frozen"))
	inferredColType.ColType = collectionColType
	return inferredColType
}

func normalizeCollectionColType(baseColType string, useSet bool, useFrozen bool) string {
	innerCollectionType := unwrapFrozenCollectionType(baseColType)
	if useSet {
		innerCollectionType = swapCollectionKind(innerCollectionType, "set")
	} else {
		innerCollectionType = swapCollectionKind(innerCollectionType, "list")
	}

	if useFrozen {
		return "frozen<" + innerCollectionType + ">"
	}
	return innerCollectionType
}

func applyFrozenCollectionDefault(baseColType string, shouldBeFrozen bool) string {
	innerCollectionType := unwrapFrozenCollectionType(baseColType)
	if shouldBeFrozen {
		return "frozen<" + innerCollectionType + ">"
	}
	return innerCollectionType
}

func unwrapFrozenCollectionType(collectionColType string) string {
	const frozenPrefix = "frozen<"
	if strings.HasPrefix(collectionColType, frozenPrefix) && strings.HasSuffix(collectionColType, ">") {
		return collectionColType[len(frozenPrefix) : len(collectionColType)-1]
	}
	return collectionColType
}

func swapCollectionKind(collectionColType string, targetKind string) string {
	if targetKind != "list" && targetKind != "set" {
		return collectionColType
	}

	openBracketIndex := strings.IndexByte(collectionColType, '<')
	closeBracketIndex := strings.LastIndexByte(collectionColType, '>')
	if openBracketIndex < 0 || closeBracketIndex < 0 || closeBracketIndex <= openBracketIndex {
		return collectionColType
	}

	elementType := collectionColType[openBracketIndex+1 : closeBracketIndex]
	return targetKind + "<" + elementType + ">"
}
