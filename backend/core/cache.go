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

func ExtractCacheVersionValues(req *HandlerArgs) []db.IDCacheVersion {
	idsStr := req.GetQuery("ids")
	cachedIDsStr := req.GetQuery("cids")
	cacheVersionsFromIDsStr := req.GetQuery("ccv")

	ids := parseConcatenatedInts(idsStr)
	cachedIDs := parseConcatenatedInts(cachedIDsStr)
	cacheVersionsFromIDs := parseConcatenatedInts(cacheVersionsFromIDsStr)

	records := []db.IDCacheVersion{}

	for _, id := range ids {
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: 0, PartitionID: req.Usuario.EmpresaID})
	}

	for i, id := range cachedIDs {
		version := uint8(0)
		if i < len(cacheVersionsFromIDs) {
			version = uint8(cacheVersionsFromIDs[i])
		}
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: version, PartitionID: req.Usuario.EmpresaID})
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
