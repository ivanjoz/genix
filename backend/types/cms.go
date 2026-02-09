package types

import "app/db"

type GaleriaImagen struct {
	db.TableStruct[GaleriaImagenTable, GaleriaImagen]
	EmpresaID   int32  `db:"empresa_id,pk"`
	Image       string `db:"image,pk"`
	Description string `db:"description"`
	Status      int8   `json:"ss" db:"status,view"`
	Updated     int32  `json:"upd" db:"updated,view.1"`
}

type GaleriaImagenTable struct {
	db.TableStruct[GaleriaImagenTable, GaleriaImagen]
	EmpresaID   db.Col[GaleriaImagenTable, int32]
	Image       db.Col[GaleriaImagenTable, string]
	Description db.Col[GaleriaImagenTable, string]
	Status      db.Col[GaleriaImagenTable, int8]
	Updated     db.Col[GaleriaImagenTable, int32]
}

func (e GaleriaImagenTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "galeria_imagenes",
		Partition: e.EmpresaID,
		Keys:      []db.Coln{e.Image},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Status}, KeepPart: true},
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}
