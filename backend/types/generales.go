package types

type PaisCiudad struct {
	TAGS         `table:"pais_ciudades"`
	PaisID       int32       `db:"pais_id,pk"`
	CiudadID     string      `json:"ID" db:"ciudad_id,pk"`
	Nombre       string      `db:"nombre"`
	PadreID      string      `db:"padre_id"`
	Jerarquia    int8        `db:"jerarquia"`
	Updated      int64       `json:"upd" db:"updated,view"`
	Departamento *PaisCiudad `json:"-"`
	Provincia    *PaisCiudad `json:"-"`
}

type ListaCompartidaRegistro struct {
	TAGS      `table:"lista_compartida_registros"`
	EmpresaID int32  `db:"empresa_id,pk"`
	ListaID   int32  `db:"lista_id"`
	ID        string `db:"id,pk"`
	Nombre    string `db:"nombre"`
	TempID    int32  `json:",omitempty"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
	Created   int64 `json:",omitempty" db:"created"`
	CreatedBy int32 `json:",omitempty" db:"created_by"`
}
