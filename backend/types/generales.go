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
	TAGS        `table:"lista_compartida_registros"`
	EmpresaID   int32    `db:"empresa_id,pk"`
	ID          int32    `db:"id,pk"`
	ListaID     int32    `db:"lista_id,view,view.1,view.2"`
	Nombre      string   `json:",omitempty" db:"nombre"`
	Images      []string `json:",omitempty" db:"images"`
	Descripcion string   `json:",omitempty" db:"descripcion"`
	// Propiedades generales
	Status    int8  `json:"ss,omitempty" db:"status,view.1"`
	Updated   int64 `json:"upd,omitempty" db:"updated,view.2"`
	UpdatedBy int32 `json:",omitempty" db:"updated_by"`
}

func (e *ListaCompartidaRegistro) GetView(view int8) any {
	if view == 1 {
		return e.ListaID*100 + int32(e.Status)
	} else if view == 2 {
		return int64(e.ListaID)*10_000_000_000 + e.Updated
	} else {
		return 0
	}
}
