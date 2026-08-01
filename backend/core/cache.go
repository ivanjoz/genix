package core

import (
	"app/db"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
)

type Cache struct {
	db.TableStruct[CacheTable, Cache]
	CompanyID    int32
	ID           int32
	Key          string
	ContentBytes []byte
	Content      string
	Updated      int32
}

type CacheTable struct {
	db.TableStruct[CacheTable, Cache]
	CompanyID    db.Col[CacheTable, int32]
	ID           db.Col[CacheTable, int32]
	Key          db.Col[CacheTable, string]
	ContentBytes db.Col[CacheTable, []byte]
	Content      db.Col[CacheTable, string]
	Updated      db.Col[CacheTable, int32]
}

func (e CacheTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        3,
		Name:      "cache",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
	}
}

func GetCacheByKeys(companyID int32, cacheKeys ...string) ([]Cache, error) {
	if companyID <= 0 {
		return nil, fmt.Errorf("company ID inválido para obtener cache")
	}
	if len(cacheKeys) == 0 {
		return []Cache{}, nil
	}

	cacheIDs := []int32{}
	for _, cacheKey := range cacheKeys {
		if cacheKey == "" {
			continue
		}
		cacheIDs = append(cacheIDs, BasicHashInt(cacheKey))
	}
	if len(cacheIDs) == 0 {
		return []Cache{}, nil
	}

	cacheRows := []Cache{}
	cacheQuery := db.Query(&cacheRows)
	cacheQuery.CompanyID.Equals(companyID)

	if len(cacheIDs) == 1 {
		cacheQuery.ID.Equals(cacheIDs[0])
	} else {
		cacheQuery.ID.In(cacheIDs...)
	}

	if cacheQueryErr := cacheQuery.Exec(); cacheQueryErr != nil {
		return nil, cacheQueryErr
	}
	return cacheRows, nil
}

func ExtractGroupIndexCacheValues(req *HandlerArgs) ([]db.GroupIndexCache, error) {

	groupHashes := parseConcatenatedInts(req.GetQuery("cc-gh"))
	updateCounters := parseConcatenatedInts(req.GetQuery("cc-upc"))

	records := makeGroupIndexCacheValues(groupHashes, updateCounters)

	// Log("records extracted:", len(records))
	return records, nil
}

func makeGroupIndexCacheValues(groupHashes []int64, updateCounters []int64) []db.GroupIndexCache {
	records := make([]db.GroupIndexCache, 0, len(groupHashes))
	for index, encodedGroupHash := range groupHashes {
		if index >= len(updateCounters) {
			continue
		}

		// Frontend sends signed int32 hashes through uint32 packing because the compact encoder is unsigned.
		records = append(records, db.GroupIndexCache{
			GroupHash:     int32(uint32(encodedGroupHash)),
			UpdateCounter: int32(updateCounters[index]),
		})
	}
	return records
}

// ExtractUpdatedVersionValues reads a by-IDs request: "ids" are records the client does not hold
// at all, while "cc-ids"/"cc-ver" carry the ones it does, each with the slot version it was last
// validated against. A version of 0 means "unknown", which never matches and always forces a read.
func (req *HandlerArgs) ExtractUpdatedVersionValues() []db.IDUpdatedVersion {
	idsStr := req.GetQuery("ids")
	cachedIDsStr := req.GetQuery("cc-ids")
	slotVersionsStr := req.GetQuery("cc-ver")
	companyID := Coalesce(req.GetQueryInt("cmp"), req.User.CompanyID)

	if companyID == 0 {
		// Invalid company scope means the cache query cannot be resolved safely.
		Log("error al extraer versiones de cache: no se envio Company-ID")
		return []db.IDUpdatedVersion{}
	}

	ids := parseConcatenatedInts(idsStr)
	cachedIDs := parseConcatenatedInts(cachedIDsStr)
	// Slot versions are a fixed-width u16 array so they stay aligned with cc-ids: the compact
	// encoder buckets by magnitude, which would reorder a mixed-width version list.
	slotVersions := parseConcatenatedUint16s(slotVersionsStr)

	records := []db.IDUpdatedVersion{}

	for _, id := range ids {
		records = append(records, db.IDUpdatedVersion{ID: id, UpdatedVersion: 0, PartitionID: companyID})
	}

	for i, id := range cachedIDs {
		version := uint16(0)
		if i < len(slotVersions) {
			version = slotVersions[i]
		}
		records = append(records, db.IDUpdatedVersion{ID: id, UpdatedVersion: version, PartitionID: companyID})
	}

	Log("records extracted:", len(records))
	return records
}

// parseConcatenatedUint16s decodes one base64url-encoded little-endian u16 array. Unlike
// parseConcatenatedInts it has no magnitude buckets, so element order is exactly what the client
// sent — which is what keeps cc-ver aligned with cc-ids.
func parseConcatenatedUint16s(encodedValues string) []uint16 {
	if encodedValues == "" {
		return nil
	}
	data, err := base64.RawURLEncoding.DecodeString(encodedValues)
	if err != nil {
		Log("error al decodificar cc-ver:", err)
		return nil
	}

	values := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		values = append(values, binary.LittleEndian.Uint16(data[i:]))
	}
	return values
}

func parseConcatenatedInts(s string) []int64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ".")
	var result []int64

	for i, part := range parts {
		if part == "" {
			continue
		}
		data, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			continue
		}

		switch i {
		case 0: // u8
			for _, b := range data {
				result = append(result, int64(b))
			}
		case 1: // u16
			for j := 0; j+1 < len(data); j += 2 {
				val := binary.LittleEndian.Uint16(data[j:])
				result = append(result, int64(val))
			}
		case 2: // u32
			for j := 0; j+3 < len(data); j += 4 {
				val := binary.LittleEndian.Uint32(data[j:])
				result = append(result, int64(val))
			}
		}
	}
	return result
}

// GlobalCache is a tenant-agnostic key/value cache partitioned by GroupID (the logical cache kind)
// and clustered by ID (here used as the CompanyID). It lets a single partition scan return every
// company registered under a group — e.g. all companies whose products changed since the last build.
type GlobalCache struct {
	db.TableStruct[GlobalCacheTable, GlobalCache]
	GroupID int16
	ID      int32
	Content []byte
	Updated int32
}

type GlobalCacheTable struct {
	db.TableStruct[GlobalCacheTable, GlobalCache]
	GroupID db.Col[GlobalCacheTable, int16]
	ID      db.Col[GlobalCacheTable, int32]
	Content db.Col[GlobalCacheTable, []byte]
	Updated db.Col[GlobalCacheTable, int32]
}

func (e GlobalCacheTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        4,
		Name:      "cache_global",
		Partition: e.GroupID,
		Keys:      db.Cols(e.ID),
	}
}

// SaveCacheGlobal upserts one global-cache row. Content is reserved for future use; the meaningful
// signal for most callers is Updated, the watermark observed when the row was written.
func SaveCacheGlobal(groupID int16, companyID int32, content []byte, updated int32) error {
	if groupID <= 0 || companyID <= 0 {
		return fmt.Errorf("groupID y companyID son requeridos para SaveCacheGlobal")
	}
	row := GlobalCache{GroupID: groupID, ID: companyID, Content: content, Updated: updated}
	if err := db.Insert(&[]GlobalCache{row}); err != nil {
		return fmt.Errorf("error al guardar cache global (group=%d company=%d): %w", groupID, companyID, err)
	}
	return nil
}

// GetCacheGlobal reads rows for a group. With no companyIDs it scans the whole group partition
// (all registered companies); otherwise it filters by the given company IDs.
func GetCacheGlobal(groupID int16, companyIDs ...int32) ([]GlobalCache, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("groupID es requerido para GetCacheGlobal")
	}
	rows := []GlobalCache{}
	query := db.Query(&rows).GroupID.Equals(groupID)
	if len(companyIDs) == 1 {
		query.ID.Equals(companyIDs[0])
	} else if len(companyIDs) > 1 {
		query.ID.In(companyIDs...)
	}
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer cache global (group=%d): %w", groupID, err)
	}
	return rows, nil
}
