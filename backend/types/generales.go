package types

import (
	"app/core"
	"app/db"
	"encoding/binary"
)

type Increment struct {
	db.TableStruct[IncrementTable, Increment]
	Name         string
	CurrentValue int64
}

type IncrementTable struct {
	db.TableStruct[IncrementTable, Increment]
	Name         db.Col[IncrementTable, string]
	CurrentValue db.Col[IncrementTable, int64]
}

func (e IncrementTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:           "sequences",
		Keys:           []db.Coln{e.Name},
		SequenceColumn: &e.CurrentValue,
	}
}

type PaisCiudad struct {
	db.TableStruct[PaisCiudadTable, PaisCiudad]
	PaisID       int32       `json:",omitempty" db:"pais_id,pk"`
	CiudadID     string      `json:"ID" db:"ciudad_id,pk"`
	Nombre       string      `db:"nombre"`
	PadreID      string      `db:"padre_id"`
	Jerarquia    int8        `json:",omitempty" db:"jerarquia"`
	Updated      int64       `json:"upd,omitempty" db:"updated,view"`
	Departamento *PaisCiudad `json:"-"`
	Provincia    *PaisCiudad `json:"-"`
}

type PaisCiudadTable struct {
	db.TableStruct[PaisCiudadTable, PaisCiudad]
	PaisID    db.Col[PaisCiudadTable, int32]
	CiudadID  db.Col[PaisCiudadTable, string]
	Nombre    db.Col[PaisCiudadTable, string]
	PadreID   db.Col[PaisCiudadTable, string]
	Jerarquia db.Col[PaisCiudadTable, int8]
	Updated   db.Col[PaisCiudadTable, int64]
}

func (e PaisCiudadTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "pais_ciudades",
		Partition: e.PaisID,
		Keys:      []db.Coln{e.CiudadID},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

type ListaCompartidaRegistro struct {
	db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
	EmpresaID   int32
	ID          int32
	ListaID     int32
	Nombre      string   `json:",omitempty"`
	Images      []string `json:",omitempty"`
	Descripcion string   `json:",omitempty"`
	NombreHash  int32    `json:",omitempty"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty"`
	Updated   int64 `json:"upd,omitempty"`
	UpdatedBy int32 `json:",omitempty"`
}

type ListaCompartidaRegistroTable struct {
	db.TableStruct[ListaCompartidaRegistroTable, ListaCompartidaRegistro]
	EmpresaID   db.Col[ListaCompartidaRegistroTable, int32]
	ID          db.Col[ListaCompartidaRegistroTable, int32]
	ListaID     db.Col[ListaCompartidaRegistroTable, int32]
	Nombre      db.Col[ListaCompartidaRegistroTable, string]
	Images      db.ColSlice[ListaCompartidaRegistroTable, string]
	Descripcion db.Col[ListaCompartidaRegistroTable, string]
	NombreHash  db.Col[ListaCompartidaRegistroTable, int32]
	Status      db.Col[ListaCompartidaRegistroTable, int8]
	Updated     db.Col[ListaCompartidaRegistroTable, int64]
	UpdatedBy   db.Col[ListaCompartidaRegistroTable, int32]
}

func (e ListaCompartidaRegistroTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "lista_compartida_registro",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
		Indexes:      [][]db.Coln{{e.NombreHash}},
		ViewsDeprecated: []db.View{
			//{Cols: []db.Coln{e.ListaID_(), e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.ListaID, e.Status}, ConcatI32: []int8{2}},
			{Cols: []db.Coln{e.ListaID, e.Updated}, ConcatI64: []int8{10}},
		},
	}
}

func (e *ListaCompartidaRegistro) SelfParse() {
	name := core.Concatn(e.ListaID, e.Nombre)
	e.NombreHash = core.BasicHashInt(core.NormalizeString(&name))
}

type NewIDToID struct {
	ID     int32
	TempID int32
}

type Parametros struct {
	db.TableStruct[ParametrosTable, Parametros]
	EmpresaID int32
	Grupo     int32
	Key       string
	Valor     string
	ValorInt  int32
	Valores   []int32
	// Propiedades generales
	Status    int8
	Updated   int64
	UpdatedBy int32
}

type ParametrosTable struct {
	db.TableStruct[ParametrosTable, Parametros]
	EmpresaID db.Col[ParametrosTable, int32]
	Grupo     db.Col[ParametrosTable, int32]
	Key       db.Col[ParametrosTable, string]
	Valor     db.Col[ParametrosTable, string]
	ValorInt  db.Col[ParametrosTable, int32]
	Valores   db.ColSlice[ParametrosTable, int32]
	Status    db.Col[ParametrosTable, int8]
	Updated   db.Col[ParametrosTable, int64]
	UpdatedBy db.Col[ParametrosTable, int32]
}

func (e ParametrosTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:            "parametros",
		Partition:       e.EmpresaID,
		UseSequences:    true,
		Keys:            []db.Coln{e.Grupo, e.Key},
		ViewsDeprecated: []db.View{},
	}
}

func EncodeIDs(ids []int32) []byte {
	// Header-only payload means "empty ID list".
	if len(ids) == 0 {
		return []byte{0}
	}

	maxBytesPerID := 1
	for _, currentID := range ids {
		requiredBytes := 1
		// Negative IDs are preserved using full int32 width.
		if currentID < 0 {
			requiredBytes = 4
		} else {
			unsignedID := uint32(currentID)
			if unsignedID > 0xFFFFFF {
				requiredBytes = 4
			} else if unsignedID > 0xFFFF {
				requiredBytes = 3
			} else if unsignedID > 0xFF {
				requiredBytes = 2
			}
		}
		if requiredBytes > maxBytesPerID {
			maxBytesPerID = requiredBytes
		}
	}

	bitWidth := byte(maxBytesPerID * 8)
	encodedIDs := make([]byte, 1+len(ids)*maxBytesPerID)
	encodedIDs[0] = bitWidth

	for idIndex, currentID := range ids {
		writeOffset := 1 + idIndex*maxBytesPerID
		unsignedID := uint32(currentID)
		switch maxBytesPerID {
		case 1:
			encodedIDs[writeOffset] = byte(unsignedID)
		case 2:
			binary.LittleEndian.PutUint16(encodedIDs[writeOffset:], uint16(unsignedID))
		case 3:
			encodedIDs[writeOffset] = byte(unsignedID)
			encodedIDs[writeOffset+1] = byte(unsignedID >> 8)
			encodedIDs[writeOffset+2] = byte(unsignedID >> 16)
		case 4:
			binary.LittleEndian.PutUint32(encodedIDs[writeOffset:], unsignedID)
		}
	}

	return encodedIDs
}

func DecodeIDs(encodedIDs []byte) []int32 {
	if len(encodedIDs) == 0 {
		return nil
	}

	bitWidth := encodedIDs[0]
	// Header-only payload for empty lists.
	if bitWidth == 0 {
		return []int32{}
	}
	if bitWidth != 8 && bitWidth != 16 && bitWidth != 24 && bitWidth != 32 {
		return nil
	}

	bytesPerID := int(bitWidth / 8)
	encodedValues := encodedIDs[1:]
	// Protect against truncated payloads.
	if len(encodedValues)%bytesPerID != 0 {
		return nil
	}

	decodedIDs := make([]int32, len(encodedValues)/bytesPerID)
	for idIndex := range decodedIDs {
		readOffset := idIndex * bytesPerID
		var unsignedID uint32
		switch bytesPerID {
		case 1:
			unsignedID = uint32(encodedValues[readOffset])
		case 2:
			unsignedID = uint32(binary.LittleEndian.Uint16(encodedValues[readOffset:]))
		case 3:
			unsignedID = uint32(encodedValues[readOffset]) |
				uint32(encodedValues[readOffset+1])<<8 |
				uint32(encodedValues[readOffset+2])<<16
		case 4:
			unsignedID = binary.LittleEndian.Uint32(encodedValues[readOffset:])
		}
		decodedIDs[idIndex] = int32(unsignedID)
	}

	return decodedIDs
}

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
