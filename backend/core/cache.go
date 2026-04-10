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
	EmpresaID    int32
	ID           int32
	Key          string
	ContentBytes []byte
	Content      string
	Updated      int32
}

type CacheTable struct {
	db.TableStruct[CacheTable, Cache]
	EmpresaID    db.Col[CacheTable, int32]
	ID           db.Col[CacheTable, int32]
	Key          db.Col[CacheTable, string]
	ContentBytes db.Col[CacheTable, []byte]
	Content      db.Col[CacheTable, string]
	Updated      db.Col[CacheTable, int32]
}

func (e CacheTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "cache",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.ID},
	}
}

func GetCacheByKeys(empresaID int32, cacheKeys ...string) ([]Cache, error) {
	if empresaID <= 0 {
		return nil, fmt.Errorf("empresa ID inválido para obtener cache")
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
	cacheQuery.EmpresaID.Equals(empresaID)

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

	Log("records extracted:", len(records))
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

func ExtractCacheVersionValues(req *HandlerArgs) ([]db.IDCacheVersion, error) {
	idsStr := req.GetQuery("ids")
	// New cache delta protocol keys: cc-ids for cached IDs and cc-ver for aligned cache versions.
	cachedIDsStr := req.GetQuery("cc-ids")
	cacheVersionsFromIDsStr := req.GetQuery("cc-ver")
	empresaID := Coalesce(req.GetQueryInt("cmp"), req.Usuario.EmpresaID)

	if empresaID == 0 {
		Log("Error: No se envió: Empresa-ID")
		return nil, Err("No se envió: Empresa-ID")
	}

	ids := parseConcatenatedInts(idsStr)
	cachedIDs := parseConcatenatedInts(cachedIDsStr)
	cacheVersionsFromIDs := parseConcatenatedInts(cacheVersionsFromIDsStr)

	records := []db.IDCacheVersion{}

	for _, id := range ids {
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: 0, PartitionID: empresaID})
	}

	for i, id := range cachedIDs {
		version := uint8(0)
		if i < len(cacheVersionsFromIDs) {
			version = uint8(cacheVersionsFromIDs[i])
		}
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: version, PartitionID: empresaID})
	}

	Log("records extracted:", len(records))
	return records, nil
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
