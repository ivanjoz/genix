package core

import (
	"encoding/binary"
	"sort"
)

// The wire format of a user's authorization grants, shared byte for byte with
// fareward/src/limiter/access.rs and with genix-ui/security/accesos.ts. This file is the only
// encoder in Go; nothing else may write these bytes.
//
// Grants live in TWO columns, and which column an access is in *is* a bit of information:
//
//	users.accesos_computed      accesses with no granted sub-access. Nothing but grant words, so
//	                            the stride is a fixed 2 bytes and the array is binary searchable.
//	users.accesos_sub_computed  accesses with at least one. Every grant word is followed by its
//	                            sub bytes, and no flag says so because the column says so.
//
// That split is what lets the grant word keep all 14 of its id bits. A single self-delimiting
// column would have had to spend one on a "sub bytes follow" flag, dropping the id ceiling to
// 8191 for no gain — see docs/SUB_ACCESSES_PLAN.md.
//
//	grant : u16 BIG-endian   bits 15..2  accesoID (1..16383)
//	                         bits  1..0  nivel - 1 (nivel 1..4)
//	sub   : u8               bit      7  MORE, another sub byte follows
//	                         bits  6..0  flags: byte 0 = sub 1..7, byte 1 = sub 8..14
//
// BIG-endian, unlike the little-endian []uint16 column this replaced. Every other integer crossing
// the fareward wire is big-endian and this column used to be the one exception, which access.rs
// carried a standing warning about: reading it backwards does not fail, it silently authorizes the
// wrong things. One endianness everywhere removes the trap.
const (
	// MaxAccesoID is what 14 bits of grant word hold. access.toml's highest id is 36, so this is
	// headroom rather than a constraint — but ids are permanent and never reused, so it is a real
	// ceiling on how many accesses the catalog may ever declare.
	MaxAccesoID = 16383

	// subAccesoFlagBits is how many sub-access flags ride in one sub byte; the eighth bit is MORE.
	subAccesoFlagBits = 7
	subAccesoFlagMask = 0x7F
	subAccesoMoreBit  = 0x80
)

// AccesoGrant is one access as a user actually holds it, after merging every profile assigned to
// them: the highest nivel any profile granted, and the union of the sub-accesses they granted.
type AccesoGrant struct {
	AccesoID int32
	Nivel    uint8
	// SubMask is bit (subID-1) per granted sub-access. Zero means the access carries none, which
	// is also what decides the column it is written to.
	SubMask uint16
}

// MakeAccesoNivelPacked empaqueta acceso + nivel en el uint16 que las dos columnas de grants
// guardan y que el frame del limitador transporta. Es el único codificador que los dos procesos
// comparten: fareward/src/limiter/access.rs decodifica exactamente esto.
func MakeAccesoNivelPacked(accesoID int32, nivel uint8) uint16 {
	if nivel < 1 || nivel > 4 {
		nivel = 1
	}
	return uint16(accesoID<<2) | uint16(nivel-1)
}

// UnpackAccesoNivel is the inverse. Kept next to its encoder so the two cannot drift.
func UnpackAccesoNivel(packedAccesoNivel uint16) (int32, uint8) {
	return int32(packedAccesoNivel >> 2), uint8(packedAccesoNivel&0b11) + 1
}

// EncodeAccesosGrants splits merged grants into the two blobs the user row stores.
//
// It sorts rather than requiring sorted input: ascending accesoID is what every reader relies on
// to early-exit and to binary search, and making the encoder responsible for it means no caller
// can get it wrong. It refuses invalid grants instead of clamping them — an out-of-range access id
// or a sub bit past the ceiling is a bug upstream, and writing a repaired blob would hide it.
func EncodeAccesosGrants(accesoGrants []AccesoGrant) ([]byte, []byte, error) {
	sortedGrants := append([]AccesoGrant{}, accesoGrants...)
	sort.Slice(sortedGrants, func(leftIndex, rightIndex int) bool {
		return sortedGrants[leftIndex].AccesoID < sortedGrants[rightIndex].AccesoID
	})

	accesosBlob := make([]byte, 0, len(sortedGrants)*2)
	accesosSubBlob := make([]byte, 0, 8)
	previousAccesoID := int32(0)

	for _, accesoGrant := range sortedGrants {
		if accesoGrant.AccesoID < 1 || accesoGrant.AccesoID > MaxAccesoID {
			return nil, nil, Err("El acceso", accesoGrant.AccesoID, "está fuera del rango codificable (1 a", MaxAccesoID, ").")
		}
		// Duplicates would make the same access answer twice with possibly different levels, and
		// which answer wins would depend on the reader's scan direction.
		if accesoGrant.AccesoID == previousAccesoID {
			return nil, nil, Err("El acceso", accesoGrant.AccesoID, "aparece más de una vez en los grants del user.")
		}
		if accesoGrant.Nivel < 1 || accesoGrant.Nivel > 4 {
			return nil, nil, Err("El acceso", accesoGrant.AccesoID, "tiene un nivel inválido:", accesoGrant.Nivel)
		}
		if accesoGrant.SubMask >= 1<<MaxSubAccesoID {
			return nil, nil, Err("El acceso", accesoGrant.AccesoID, "tiene sub-accesos por encima del máximo", MaxSubAccesoID, ".")
		}
		previousAccesoID = accesoGrant.AccesoID

		packedAccesoNivel := binary.BigEndian.AppendUint16(nil,
			MakeAccesoNivelPacked(accesoGrant.AccesoID, accesoGrant.Nivel))

		if accesoGrant.SubMask == 0 {
			accesosBlob = append(accesosBlob, packedAccesoNivel...)
			continue
		}
		accesosSubBlob = append(accesosSubBlob, packedAccesoNivel...)
		accesosSubBlob = appendSubAccesoBytes(accesosSubBlob, accesoGrant.SubMask)
	}

	return accesosBlob, accesosSubBlob, nil
}

// appendSubAccesoBytes writes a mask as 7-bit groups, low group first, with MORE set on every byte
// but the last. Trailing empty groups are dropped, so a user holding only "Todos" costs one byte.
func appendSubAccesoBytes(destination []byte, subMask uint16) []byte {
	for {
		subAccesoByte := byte(subMask & subAccesoFlagMask)
		subMask >>= subAccesoFlagBits
		if subMask != 0 {
			subAccesoByte |= subAccesoMoreBit
		}
		destination = append(destination, subAccesoByte)
		if subMask == 0 {
			return destination
		}
	}
}

// DecodeAccesosGrants reads both blobs back into merged grants, sorted by accesoID.
//
// Every reader of these bytes validates as it walks rather than trusting the writer, because the
// alternative failure is silent: a blob that is out of order or truncated does not error, it
// answers the wrong question about what a user may do. The []uint16 column this replaced was
// defensively re-sorted on load for the same reason; ordering is load-bearing here, so it is
// checked instead.
func DecodeAccesosGrants(accesosBlob []byte, accesosSubBlob []byte) ([]AccesoGrant, error) {
	if len(accesosBlob)%2 != 0 {
		return nil, Err("accesos_computed tiene", len(accesosBlob), "bytes, que no son grants completos de 2 bytes.")
	}

	accesoGrants := make([]AccesoGrant, 0, len(accesosBlob)/2+2)

	previousAccesoID := int32(0)
	for offset := 0; offset < len(accesosBlob); offset += 2 {
		accesoID, nivel := UnpackAccesoNivel(binary.BigEndian.Uint16(accesosBlob[offset : offset+2]))
		if accesoID <= previousAccesoID {
			return nil, Err("accesos_computed no está ordenado: el acceso", accesoID, "sigue a", previousAccesoID, ".")
		}
		previousAccesoID = accesoID
		accesoGrants = append(accesoGrants, AccesoGrant{AccesoID: accesoID, Nivel: nivel})
	}

	previousAccesoID = 0
	for offset := 0; offset < len(accesosSubBlob); {
		if offset+3 > len(accesosSubBlob) {
			return nil, Err("accesos_sub_computed termina a mitad de un grant en el byte", offset, ".")
		}
		accesoID, nivel := UnpackAccesoNivel(binary.BigEndian.Uint16(accesosSubBlob[offset : offset+2]))
		if accesoID <= previousAccesoID {
			return nil, Err("accesos_sub_computed no está ordenado: el acceso", accesoID, "sigue a", previousAccesoID, ".")
		}
		previousAccesoID = accesoID
		offset += 2

		subMask, subBytesRead, err := readSubAccesoBytes(accesosSubBlob[offset:])
		if err != nil {
			return nil, Err("accesos_sub_computed, acceso", accesoID, "::", err)
		}
		offset += subBytesRead

		// An access with no sub-accesses belongs in the other column. Finding one here means the
		// two blobs were written by something that does not agree with this encoder.
		if subMask == 0 {
			return nil, Err("accesos_sub_computed contiene el acceso", accesoID, "sin ningún sub-acceso.")
		}
		accesoGrants = append(accesoGrants, AccesoGrant{AccesoID: accesoID, Nivel: nivel, SubMask: subMask})
	}

	sort.Slice(accesoGrants, func(leftIndex, rightIndex int) bool {
		return accesoGrants[leftIndex].AccesoID < accesoGrants[rightIndex].AccesoID
	})
	return accesoGrants, nil
}

// readSubAccesoBytes consumes one MORE-terminated run and returns the mask and how many bytes it
// spanned. A run that would overflow the declared ceiling is refused rather than truncated.
func readSubAccesoBytes(subAccesoBytes []byte) (uint16, int, error) {
	subMask := uint16(0)
	for byteIndex := 0; byteIndex < len(subAccesoBytes); byteIndex++ {
		if byteIndex*subAccesoFlagBits >= 16 {
			return 0, 0, Err("la secuencia de sub-accesos excede los 16 bits de máscara.")
		}
		subMask |= uint16(subAccesoBytes[byteIndex]&subAccesoFlagMask) << (byteIndex * subAccesoFlagBits)
		if subAccesoBytes[byteIndex]&subAccesoMoreBit == 0 {
			if subMask >= 1<<MaxSubAccesoID {
				return 0, 0, Err("la secuencia de sub-accesos supera el máximo", MaxSubAccesoID, ".")
			}
			return subMask, byteIndex + 1, nil
		}
	}
	// The last byte had MORE set and nothing followed it.
	return 0, 0, Err("la secuencia de sub-accesos termina con el bit MORE activo.")
}

// DecodeSubAccesoBytes reads one MORE-terminated sub-access run back into its mask.
//
// This is the seam where the daemon's opacity ends: fareward hands back the bytes it read out of
// accesos_sub_computed without knowing what any bit means, and this turns them into the mask the
// catalog can name. Empty is a legitimate answer — an access granted with no sub-accesses — so it
// is a zero mask and not an error.
func DecodeSubAccesoBytes(subAccesoBytes []byte) (uint16, error) {
	if len(subAccesoBytes) == 0 {
		return 0, nil
	}
	subMask, _, err := readSubAccesoBytes(subAccesoBytes)
	return subMask, err
}

// HasSubAcceso applies the one catalog rule the daemon deliberately does not know: sub-access 1 is
// "Todos" and satisfies every check on that access. fareward returns the raw mask; the meaning of
// bit 0 is expanded here and in genix-ui/security/accesos.ts, and nowhere else.
func HasSubAcceso(subMask uint16, subAccesoID int32) bool {
	if subMask&(1<<(SubAccesoTodosID-1)) != 0 {
		return true
	}
	if subAccesoID < 1 || subAccesoID > MaxSubAccesoID {
		return false
	}
	return subMask&(1<<uint16(subAccesoID-1)) != 0
}
