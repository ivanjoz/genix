package index_builder

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

const (
	taxonomyBinaryMagic   = "GIXTAX01"
	taxonomyBinaryVersion = uint8(1)
)

type taxonomyBinarySection struct {
	id        uint8
	name      string
	data      []byte
	itemCount uint32
}

// ToBytes serializes stage-1 payload and appends stage-2 taxonomy columns.
func (buildResult *ProductosIndexBuild) ToBytes() ([]byte, error) {
	if buildResult == nil {
		return nil, fmt.Errorf("nil productos build result")
	}

	textPayload, marshalTextErr := buildResult.MarshalBinary()
	if marshalTextErr != nil {
		return nil, marshalTextErr
	}
	taxonomyPayload, marshalTaxonomyErr := marshalTaxonomyBinary(buildResult)
	if marshalTaxonomyErr != nil {
		return nil, marshalTaxonomyErr
	}

	combinedPayload := make([]byte, 0, len(textPayload)+len(taxonomyPayload))
	combinedPayload = append(combinedPayload, textPayload...)
	combinedPayload = append(combinedPayload, taxonomyPayload...)
	return combinedPayload, nil
}

func marshalTaxonomyBinary(taxonomyBuildResult *ProductosIndexBuild) ([]byte, error) {
	if validateErr := taxonomyBuildResult.ValidateForBinary(); validateErr != nil {
		return nil, validateErr
	}

	brandIDsSection := marshalUint16SliceLE(taxonomyBuildResult.BrandIDs)
	categoryIDsSection := marshalUint16SliceLE(taxonomyBuildResult.CategoryIDs)
	brandNamesSection, marshalBrandNamesErr := marshalStringColumn(taxonomyBuildResult.BrandNames)
	if marshalBrandNamesErr != nil {
		return nil, fmt.Errorf("marshal brand names: %w", marshalBrandNamesErr)
	}
	categoryNamesSection, marshalCategoryNamesErr := marshalStringColumn(taxonomyBuildResult.CategoryNames)
	if marshalCategoryNamesErr != nil {
		return nil, fmt.Errorf("marshal category names: %w", marshalCategoryNamesErr)
	}

	var brandIndexesSection []byte
	if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint12 {
		brandIndexesSection = taxonomyBuildResult.ProductBrandIndexesUint12Packed
	} else if taxonomyBuildResult.BrandIndexEncodingFlag == BrandIndexEncodingUint16 {
		brandIndexesSection = marshalUint16SliceLE(taxonomyBuildResult.ProductBrandIndexesUint16)
	} else {
		return nil, fmt.Errorf("unsupported brand index encoding flag=%d", taxonomyBuildResult.BrandIndexEncodingFlag)
	}

	sections := []taxonomyBinarySection{
		{id: 1, name: "brand_ids", data: brandIDsSection, itemCount: uint32(len(taxonomyBuildResult.BrandIDs))},
		{id: 2, name: "brand_names", data: brandNamesSection, itemCount: uint32(len(taxonomyBuildResult.BrandNames))},
		{id: 3, name: "category_ids", data: categoryIDsSection, itemCount: uint32(len(taxonomyBuildResult.CategoryIDs))},
		{id: 4, name: "category_names", data: categoryNamesSection, itemCount: uint32(len(taxonomyBuildResult.CategoryNames))},
		{id: 5, name: "brand_indexes", data: brandIndexesSection, itemCount: uint32(len(taxonomyBuildResult.SortedIDs))},
		{id: 6, name: "category_count", data: taxonomyBuildResult.ProductCategoryCount, itemCount: uint32(len(taxonomyBuildResult.SortedIDs))},
		{id: 7, name: "category_indexes", data: taxonomyBuildResult.ProductCategoryIndexes, itemCount: uint32(len(taxonomyBuildResult.ProductCategoryIndexes))},
	}

	const sectionEntrySize = 1 + 4 + 4 + 4 + 4 // section_id + offset + length + item_count + checksum_crc32
	baseHeaderSize := len(taxonomyBinaryMagic) + 1 + 1 + 4 + 1 + 2
	headerSize := baseHeaderSize + len(sections)*sectionEntrySize
	totalSize := headerSize
	for _, taxonomySection := range sections {
		if len(taxonomySection.data) > int(^uint32(0)) {
			return nil, fmt.Errorf("section=%s length overflows uint32", taxonomySection.name)
		}
		totalSize += len(taxonomySection.data)
	}

	payload := make([]byte, 0, totalSize)
	payload = append(payload, []byte(taxonomyBinaryMagic)...)
	payload = append(payload, taxonomyBinaryVersion)
	payload = append(payload, taxonomyBuildResult.BrandIndexEncodingFlag)

	if len(taxonomyBuildResult.SortedIDs) > int(^uint32(0)) {
		return nil, fmt.Errorf("sorted products count overflows uint32")
	}
	var sortedCountBytes [4]byte
	binary.LittleEndian.PutUint32(sortedCountBytes[:], uint32(len(taxonomyBuildResult.SortedIDs)))
	payload = append(payload, sortedCountBytes[:]...)
	payload = append(payload, uint8(len(sections)))
	if headerSize > 65535 {
		return nil, fmt.Errorf("taxonomy header too large bytes=%d", headerSize)
	}
	var headerSizeBytes [2]byte
	binary.LittleEndian.PutUint16(headerSizeBytes[:], uint16(headerSize))
	payload = append(payload, headerSizeBytes[:]...)

	currentOffset := uint32(headerSize)
	for _, taxonomySection := range sections {
		payload = append(payload, taxonomySection.id)
		var offsetBytes [4]byte
		binary.LittleEndian.PutUint32(offsetBytes[:], currentOffset)
		payload = append(payload, offsetBytes[:]...)
		var lengthBytes [4]byte
		binary.LittleEndian.PutUint32(lengthBytes[:], uint32(len(taxonomySection.data)))
		payload = append(payload, lengthBytes[:]...)
		var countBytes [4]byte
		binary.LittleEndian.PutUint32(countBytes[:], taxonomySection.itemCount)
		payload = append(payload, countBytes[:]...)
		var checksumBytes [4]byte
		binary.LittleEndian.PutUint32(checksumBytes[:], crc32.ChecksumIEEE(taxonomySection.data))
		payload = append(payload, checksumBytes[:]...)
		currentOffset += uint32(len(taxonomySection.data))
	}
	for _, taxonomySection := range sections {
		payload = append(payload, taxonomySection.data...)
	}
	return payload, nil
}

func marshalUint16SliceLE(values []uint16) []byte {
	sectionBytes := make([]byte, 0, len(values)*2)
	for _, value := range values {
		sectionBytes = append(sectionBytes, byte(value), byte(value>>8))
	}
	return sectionBytes
}

func marshalStringColumn(values []string) ([]byte, error) {
	totalBytes := 0
	for _, value := range values {
		valueBytes := []byte(value)
		if len(valueBytes) > 255 {
			return nil, fmt.Errorf("string exceeds 255 bytes: %q", value)
		}
		totalBytes += 1 + len(valueBytes)
	}

	sectionBytes := make([]byte, 0, totalBytes)
	for _, value := range values {
		valueBytes := []byte(value)
		sectionBytes = append(sectionBytes, uint8(len(valueBytes)))
		sectionBytes = append(sectionBytes, valueBytes...)
	}
	return sectionBytes, nil
}
