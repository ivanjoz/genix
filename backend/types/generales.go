package types

import "app/db"

type Increment struct {
	TAGS         `table:"sequences"`
	TableName    string `db:"name,pk"`
	CurrentValue int64  `db:"current_value,counter"`
}

func (e Increment) TableName_() db.CoStr    { return db.CoStr{"name"} }
func (e Increment) CurrentValue_() db.CoI64 { return db.CoI64{"current_value"} }

func (e Increment) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:           "sequences",
		Keys:           []db.Coln{e.TableName_()},
		SequenceColumn: e.CurrentValue_(),
	}
}

type PaisCiudad struct {
	TAGS         `table:"pais_ciudades"`
	PaisID       int32       `json:",omitempty" db:"pais_id,pk"`
	CiudadID     string      `json:"ID" db:"ciudad_id,pk"`
	Nombre       string      `db:"nombre"`
	PadreID      string      `db:"padre_id"`
	Jerarquia    int8        `json:",omitempty" db:"jerarquia"`
	Updated      int64       `json:"upd,omitempty" db:"updated,view"`
	Departamento *PaisCiudad `json:"-"`
	Provincia    *PaisCiudad `json:"-"`
}

type _a = PaisCiudad

func (e _a) PaisID_() db.CoI32   { return db.CoI32{"pais_id"} }
func (e _a) CiudadID_() db.CoStr { return db.CoStr{"ciudad_id"} }
func (e _a) Nombre_() db.CoStr   { return db.CoStr{"nombre"} }
func (e _a) PadreID_() db.CoI32  { return db.CoI32{"padre_id"} }
func (e _a) Jerarquia_() db.CoI8 { return db.CoI8{"jerarquia"} }
func (e _a) Updated_() db.CoI64  { return db.CoI64{"updated"} }

func (e PaisCiudad) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "pais_ciudades",
		Partition: e.PaisID_(),
		Keys:      []db.Coln{e.CiudadID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.Updated_()}, KeepPart: true},
		},
	}
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

type _b = ListaCompartidaRegistro

func (e _b) EmpresaID_() db.CoI32   { return db.CoI32{"empresa_id"} }
func (e _b) ID_() db.CoI32          { return db.CoI32{"id"} }
func (e _b) ListaID_() db.CoI32     { return db.CoI32{"lista_id"} }
func (e _b) Nombre_() db.CoStr      { return db.CoStr{"nombre"} }
func (e _b) Images_() db.CsStr      { return db.CsStr{"images"} }
func (e _b) Descripcion_() db.CoStr { return db.CoStr{"descripcion"} }
func (e _b) Status_() db.CoI8       { return db.CoI8{"status"} }
func (e _b) Updated_() db.CoI64     { return db.CoI64{"updated"} }
func (e _b) UpdatedBy_() db.CoI32   { return db.CoI32{"updated_by"} }

func (e ListaCompartidaRegistro) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "lista_compartida_registro",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
		Views: []db.View{
			{Cols: []db.Coln{e.ListaID_(), e.Status_()}, KeepPart: true},
			{Cols: []db.Coln{e.ListaID_(), e.Status_()}, ConcatI32: []int8{2}},
			{Cols: []db.Coln{e.ListaID_(), e.Updated_()}, ConcatI64: []int8{10}},
		},
	}
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

type NewIDToID struct {
	NewID  int32
	TempID int32
}
