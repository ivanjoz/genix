package types

import "strconv"

// SubAccesoIDs is a list of sub-access ids that survives a JSON round trip.
//
// It exists because `[]uint8` is `[]byte`, and encoding/json writes any byte slice as a base64
// string: the browser would receive `"Ag=="` where it expects `[2]`. The decoder accepts both
// spellings, so without this the write path worked and only the read path broke — silently, as a
// sub-access that renders as four characters of base64 rather than as a ticked checkbox.
type SubAccesoIDs []uint8

func (subAccesoIDs SubAccesoIDs) MarshalJSON() ([]byte, error) {
	jsonArray := make([]byte, 0, len(subAccesoIDs)*3+2)
	jsonArray = append(jsonArray, '[')
	for index, subAccesoID := range subAccesoIDs {
		if index > 0 {
			jsonArray = append(jsonArray, ',')
		}
		jsonArray = strconv.AppendUint(jsonArray, uint64(subAccesoID), 10)
	}
	return append(jsonArray, ']'), nil
}

// AccesoGrantRecord is one access exactly as an operator granted it. A profile and a user grant
// access the same way, so both store a `[]AccesoGrantRecord` in their `accesos_grants` column —
// a blob, serialized by colbin (genix-orm/scylla/converter.go).
//
// It replaced three flat integer arrays with hand-rolled packing: `profiles.accesos`
// (accesoID*10 + nivel), `profiles.sub_accesos` (accesoID*100 + subID) and
// `users.access_level_ids`. Nesting the sub-accesses inside the grant is what makes an orphan
// impossible to store: a sub-access cannot exist without the access it qualifies, because it lives
// inside it. See docs/ACCESO_GRANTS_STORAGE_PLAN.md.
//
// The `cb` tags are positional field ids, not names. They keep the blob compact and let a field be
// renamed without rewriting a single stored row — so an id is permanent, and a removed field's
// number is never reused.
type AccesoGrantRecord struct {
	AccesoID uint16 `cb:"1"`
	// Nivel is the single widest level granted on this access, 1..4. One level and not a list:
	// every consumer already collapses to the highest one, so a list only ever let the editor
	// express something the merge immediately threw away.
	Nivel uint8 `cb:"2"`
	// SubAccesos are the granted sub-access ids. Id 1 is "Todos" and satisfies every check on the
	// access; it is synthesized rather than declared, so no catalog lists it. Empty is the normal
	// case — most accesses declare no sub-access at all.
	SubAccesos SubAccesoIDs `cb:"3" json:",omitempty"`
}
