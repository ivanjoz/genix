package db

import (
	"fmt"
	"strings"
)

type dbTagConfig struct {
	columnName string
	isFrozen   bool
	isTypeList bool
	isTypeSet  bool
}

func parseDBTagConfig(dbTagRaw string) dbTagConfig {
	tagConfig := dbTagConfig{}
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

		// Keep collection tags as strongly typed booleans to avoid magic-string checks.
		switch tagOption {
		case "frozen":
			tagConfig.isFrozen = true
		case "list":
			tagConfig.isTypeList = true
		case "set":
			tagConfig.isTypeSet = true
		}
	}

	return tagConfig
}

func (tagConfig dbTagConfig) hasCollectionOptions() bool {
	return tagConfig.isTypeList || tagConfig.isTypeSet || tagConfig.isFrozen
}

func applyCollectionTagOptions(recordTypeName string, recordFieldName string, inferredColType colType, tagConfig dbTagConfig) colType {
	if !tagConfig.hasCollectionOptions() {
		return inferredColType
	}

	if !inferredColType.IsSlice {
		panic(fmt.Sprintf(`Record "%v": field "%v" uses db collection options on non-slice type "%v".`,
			recordTypeName, recordFieldName, inferredColType.FieldType))
	}

	// Reject ambiguous collection kind options so tag behavior is explicit and deterministic.
	if tagConfig.isTypeList && tagConfig.isTypeSet {
		panic(fmt.Sprintf(`Record "%v": field "%v" cannot declare both "list" and "set" db options.`,
			recordTypeName, recordFieldName))
	}

	innerCollectionType := unwrapFrozenCollectionType(inferredColType.ColType)
	if tagConfig.isTypeSet {
		innerCollectionType = swapCollectionKind(innerCollectionType, "set")
	} else {
		innerCollectionType = swapCollectionKind(innerCollectionType, "list")
	}

	if tagConfig.isFrozen {
		inferredColType.ColType = "frozen<" + innerCollectionType + ">"
		return inferredColType
	}

	collectionColType := innerCollectionType
	inferredColType.ColType = collectionColType
	return inferredColType
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
