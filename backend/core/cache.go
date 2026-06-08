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
		Name:      "cache",
		Partition: e.CompanyID,
		Keys:      []db.Coln{e.ID},
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

func (req *HandlerArgs) ExtractCacheVersionValues() []db.IDCacheVersion {
	idsStr := req.GetQuery("ids")
	// New cache delta protocol keys: cc-ids for cached IDs and cc-ver for aligned cache versions.
	cachedIDsStr := req.GetQuery("cc-ids")
	cacheVersionsFromIDsStr := req.GetQuery("cc-ver")
	companyID := Coalesce(req.GetQueryInt("cmp"), req.User.CompanyID)

	if companyID == 0 {
		// Invalid company scope means the cache query cannot be resolved safely.
		Log("error al extraer versiones de cache: no se envio Company-ID")
		return []db.IDCacheVersion{}
	}

	ids := parseConcatenatedInts(idsStr)
	cachedIDs := parseConcatenatedInts(cachedIDsStr)
	cacheVersionsFromIDs := parseConcatenatedInts(cacheVersionsFromIDsStr)

	records := []db.IDCacheVersion{}

	for _, id := range ids {
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: 0, PartitionID: companyID})
	}

	for i, id := range cachedIDs {
		version := uint8(0)
		if i < len(cacheVersionsFromIDs) {
			version = uint8(cacheVersionsFromIDs[i])
		}
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: version, PartitionID: companyID})
	}

	Log("records extracted:", len(records))
	return records
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
		Name:      "cache_global",
		Partition: e.GroupID,
		Keys:      []db.Coln{e.ID},
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
