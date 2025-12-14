package types

import "app/db2"

type GaleriaImagen struct {
	db2.TableStruct[GaleriaImagenTable, GaleriaImagen]
	EmpresaID   int32  `db:"empresa_id,pk"`
	Image       string `db:"image,pk"`
	Description string `db:"description"`
	Status      int8   `json:"ss" db:"status,view"`
	Updated     int32  `json:"upd" db:"updated,view.1"`
}

type GaleriaImagenTable struct {
	db2.TableStruct[GaleriaImagenTable, GaleriaImagen]
	EmpresaID   db2.Col[GaleriaImagenTable, int32]
	Image       db2.Col[GaleriaImagenTable, string]
	Description db2.Col[GaleriaImagenTable, string]
	Status      db2.Col[GaleriaImagenTable, int8]
	Updated     db2.Col[GaleriaImagenTable, int32]
}

func (e GaleriaImagenTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "galeria_imagenes",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.Image},
		Views: []db2.View{
			{Cols: []db2.Coln{e.Status}, KeepPart: true},
			{Cols: []db2.Coln{e.Updated}, KeepPart: true},
		},
	}
}
