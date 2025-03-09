package types

import "app/db"

type GaleriaImagen struct {
	EmpresaID   int32
	Image       string
	Description string
	Status      int8
}

type gi = GaleriaImagen

func (e gi) EmpresaID_() db.CoI32   { return db.CoI32{"empresa_id"} }
func (e gi) Image_() db.CoStr       { return db.CoStr{"image"} }
func (e gi) Description_() db.CoStr { return db.CoStr{"description"} }
func (e gi) Status_() db.CoI8       { return db.CoI8{"status"} }

func (e GaleriaImagen) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "galeria_imagenes",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.Image_()},
		Views: []db.View{
			{Cols: []db.Coln{e.Status_()}, KeepPart: true},
		},
	}
}
