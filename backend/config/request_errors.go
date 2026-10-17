package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"cmp"
	"slices"
)

const requestErrorIDsLimit = 500

type requestErrorEntry struct {
	CodeLine string
	Text     string
	Updated  int32 `json:"upd,omitempty"`
}

type requestErrorByID struct {
	ID      int32 `json:"ID"`
	Entries []requestErrorEntry
}

// GetRequestErrorsByIDs preserves every code-line row under its hash so rare collisions cannot
// overwrite one another in the frontend's static by-ID cache.
func GetRequestErrorsByIDs(req *core.HandlerArgs) core.HandlerResponse {
	staticIDs, err := req.GetStaticRequestIDs()
	if err != nil {
		return req.MakeErr(err)
	}
	if len(staticIDs) > requestErrorIDsLimit {
		return req.MakeErr("Se enviaron demasiados IDs de errores.")
	}

	requestedIDs := make([]int32, 0, len(staticIDs))
	for _, id := range staticIDs {
		if id > 0 && id <= int64(^uint32(0)>>1) {
			requestedIDs = append(requestedIDs, int32(id))
		}
	}
	slices.Sort(requestedIDs)
	requestedIDs = slices.Compact(requestedIDs)
	if len(requestedIDs) == 0 {
		return req.MakeErr("Los IDs de errores enviados no son válidos.")
	}

	rows := []coreTypes.RequestError{}
	query := db.Query(&rows)
	if err := query.ID.In(requestedIDs...).Exec(); err != nil {
		core.Log("request errors by IDs query failed::", " ids::", len(requestedIDs), " err::", err)
		return req.MakeErr("No se pudieron obtener los errores solicitados.", err)
	}

	return req.MakeResponse(core.StaticCachedPartition(makeRequestErrorsByID(requestedIDs, rows)))
}

func makeRequestErrorsByID(requestedIDs []int32, rows []coreTypes.RequestError) []requestErrorByID {
	entriesByID := make(map[int32][]requestErrorEntry, len(requestedIDs))
	for _, row := range rows {
		entriesByID[row.ID] = append(entriesByID[row.ID], requestErrorEntry{
			CodeLine: row.CodeLine,
			Text:     row.Text,
			Updated:  row.Updated,
		})
	}
	result := make([]requestErrorByID, 0, len(requestedIDs))
	for _, requestedID := range requestedIDs {
		entries := entriesByID[requestedID]
		slices.SortFunc(entries, func(left, right requestErrorEntry) int {
			return cmp.Compare(left.CodeLine, right.CodeLine)
		})
		result = append(result, requestErrorByID{ID: requestedID, Entries: entries})
	}

	return result
}
