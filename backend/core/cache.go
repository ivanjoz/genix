package core

import (
	"app/db"
	"encoding/base64"
	"encoding/binary"
	"strings"
)

func ExtractCacheVersionValues(req *HandlerArgs) []db.IDCacheVersion {
	idsStr := req.GetQuery("ids")
	cidsStr := req.GetQuery("cids")
	ccvStr := req.GetQuery("ccv")

	ids := parseConcatenatedInts(idsStr)
	cachedIDs := parseConcatenatedInts(cidsStr)
	cacheVersionsFromIDs := parseConcatenatedInts(ccvStr)

	records := []db.IDCacheVersion{}

	for _, id := range ids {
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: 0})
	}

	for i, id := range cachedIDs {
		version := uint8(0)
		if i < len(cacheVersionsFromIDs) {
			version = uint8(cacheVersionsFromIDs[i])
		}
		records = append(records, db.IDCacheVersion{ID: id, CacheVersion: version})
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
