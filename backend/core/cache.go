package core

import (
	"app/db"
	"encoding/binary"
	"fmt"
	"github.com/ivanjoz/genix-orm/utils"
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
	CompanyID    db.Col[*CacheTable, int32]
	ID           db.Col[*CacheTable, int32]
	Key          db.Col[*CacheTable, string]
	ContentBytes db.Col[*CacheTable, []byte]
	Content      db.Col[*CacheTable, string]
	Updated      db.Col[*CacheTable, int32]
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

// GetCachedIDsRequest decodes the by-IDs request in ?cc=. The partition is the tenant, so it is never
// the client's: genix clients send every ID under partition 0, and companyID (the token's, or the
// storefront's on a public route) takes its place. Any other partition is refused.
func (req *HandlerArgs) GetCachedIDsRequest(companyID int32) (db.CachedIDsRequest, error) {
	if companyID <= 0 {
		return db.CachedIDsRequest{}, fmt.Errorf("the by-IDs request has no company")
	}
	request, err := db.DecodeCachedIDsRequest(req.GetQuery("cc"))
	if err != nil {
		return db.CachedIDsRequest{}, fmt.Errorf("invalid by-IDs request (cc): %w", err)
	}
	if len(request.IDs) == 0 {
		return db.CachedIDsRequest{}, fmt.Errorf("the by-IDs request names no IDs")
	}
	for i := range request.IDs {
		if request.IDs[i].Partition != 0 {
			return db.CachedIDsRequest{}, fmt.Errorf("the by-IDs request must send its IDs under partition 0, got %d", request.IDs[i].Partition)
		}
		request.IDs[i].Partition = int64(companyID)
	}
	return request, nil
}

// HideCachedPartitions puts back the partition 0 the client asked under: the company is the server's.
func HideCachedPartitions[E any](partitions []db.CachedPartition[E]) []db.CachedPartition[E] {
	for i := range partitions {
		partitions[i].Partition = 0
	}
	return partitions
}

// deltaReplaceRecord is the one Metadata record a route not backed by a delta log appends.
type deltaReplaceRecord struct {
	Metadata string `json:"mdt"`
}

// WithDeltaReplace is the response of a cached route that is not backed by a delta log: its whole
// list plus a replace Metadata record, so the client drops every held record the list doesn't bring
// (a deleted one, or one that left the list).
func WithDeltaReplace[E any](records []E) []any {
	response := make([]any, 0, len(records)+1)
	for _, record := range records {
		response = append(response, record)
	}
	return append(response, deltaReplaceRecord{Metadata: db.DeltaReplaceMetadata()})
}

// GetStaticRequestIDs reads the IDs of a by-IDs request (?cc=) sent to a static lookup route: one
// whose record never changes for an ID, so the ticks the client holds don't matter and it answers
// every ID. genix clients send them under partition 0.
func (req *HandlerArgs) GetStaticRequestIDs() ([]int64, error) {
	request, err := db.DecodeCachedIDsRequest(req.GetQuery("cc"))
	if err != nil {
		return nil, fmt.Errorf("invalid by-IDs request (cc): %w", err)
	}
	ids := make([]int64, 0, len(request.IDs))
	for _, cachedID := range request.IDs {
		if cachedID.Partition != 0 {
			return nil, fmt.Errorf("the by-IDs request must send its IDs under partition 0, got %d", cachedID.Partition)
		}
		ids = append(ids, cachedID.ID)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("the by-IDs request names no IDs")
	}
	return ids, nil
}

// StaticCachedPartition is the by-IDs response of a static lookup route: every record, in partition 0.
func StaticCachedPartition[E any](records []E) []db.CachedPartition[E] {
	return []db.CachedPartition[E]{{Records: records}}
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
		data, err := utils.Base64.DecodeString(part)
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
	GroupID db.Col[*GlobalCacheTable, int16]
	ID      db.Col[*GlobalCacheTable, int32]
	Content db.Col[*GlobalCacheTable, []byte]
	Updated db.Col[*GlobalCacheTable, int32]
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
