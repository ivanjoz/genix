package types

type Producto struct {
	TAGS        `table:"productos"`
	EmpresaID   int32   `json:"empresaID" db:"empresa_id,pk"`
	ID          int32   `json:"id" db:"id,pk"`
	Nombre      string  `json:"nombre" db:"nombre"`
	Descripcion string  `json:"descripcion" db:"descripcion"`
	GruposIDs   []int32 `json:"gruposIDs" db:"grupos_ids"`
	Status      int8    `json:"ss" db:"status,view"`
	Updated     int64   `json:"upd" db:"updated,view"`
	UpdatedBy   int32   `json:"updatedBy" db:"updated_by"`
	Created     int64   `json:"created" db:"created"`
	CreatedBy   int32   `json:"createdBy" db:"created_by"`
}

type Almacen struct {
	TAGS        `table:"almacenes"`
	EmpresaID   int32           `db:"empresa_id,pk"`
	ID          int32           `db:"id"`
	SedeID      int32           `db:"sede_id"`
	Nombre      string          `db:"nombre"`
	Descripcion string          `db:"descripcion"`
	Layout      []AlmacenLayout `db:"layout"`
	Status      int8            `json:"ss" db:"status"`
	Updated     int64           `json:"upd" db:"updated,view"`
	UpdatedBy   int32           `db:"updated_by"`
	Created     int64           `db:"created"`
	CreatedBy   int32           `db:"created_by"`
}

type ProductoStock struct {
	TAGS             `table:"producto_stock"`
	EmpresaID        int32   `db:"empresa_id,pk"`
	ProductoID       int32   `db:"id,pk"`
	AlmacenID        int32   `db:"almacen_id"`
	LayoutID         int16   `db:"layout_id"`
	LayoutPosicionID int16   `db:"layout_posicion_id"`
	Stock            float32 `db:"stock"`
	SKU              string  `db:"sku"`
	Updated          int64   `json:"upd" db:"updated,view"`
	UpdatedBy        int32   `db:"updated_by"`
}

type AlmacenLayout struct {
	ID      int16                 `ms:"i"`
	Name    string                `ms:"n"`
	RowCant int8                  `ms:"r"`
	ColCant int8                  `ms:"c"`
	Bloques []AlmacenLayoutBloque `ms:"b"`
}

type AlmacenLayoutBloque struct {
	Row    int8   `json:"rw" ms:"r"`
	Column int8   `json:"co" ms:"c"`
	Name   string `json:"nm" ms:"n"`
}

type Sede struct {
	TAGS        `table:"sedes"`
	EmpresaID   int32  `db:"empresa_id,pk"`
	ID          int32  `db:"id"`
	Nombre      string `db:"nombre"`
	Descripcion string `db:"descripcion"`
	Direccion   string `db:"direccion"`
	CiudadID    string `db:"pais_ciudad_id"`
	Status      int8   `json:"ss" db:"status"`
	Updated     int64  `json:"upd" db:"updated,view"`
	UpdatedBy   int32  `db:"updated_by"`
	Created     int64  `db:"created"`
	CreatedBy   int32  `db:"created_by"`
	Ciudad      string
}

/*
CREATE TABLE sedes
(
    empresa_id int,
    id         int,
    nombre     text,
    direccion     text,
    telefono     text,
    departamento int,
    distrito int,
    created     bigint,
    created_by  int,
    status      tinyint,
    updated     bigint,
    updated_by  int,
    PRIMARY KEY ((empresa_id), id)
)
    WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
     and compaction = {'class': 'SizeTieredCompactionStrategy'}
     and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
     and dclocal_read_repair_chance = 0
     and speculative_retry = '99.0PERCENTILE';

CREATE INDEX ON sedes((empresa_id),updated);

CREATE MATERIALIZED VIEW sedes__updated_view
AS
SELECT  * FROM sedes
WHERE empresa_id IS NOT null
  AND updated IS NOT null
  AND id IS NOT null
PRIMARY KEY ((empresa_id), updated, id)
WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
 and compaction = {'class': 'SizeTieredCompactionStrategy'}
 and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
 and dclocal_read_repair_chance = 0
 and speculative_retry = '99.0PERCENTILE';

CREATE MATERIALIZED VIEW sedes__status_view
AS
SELECT  * FROM sedes
WHERE empresa_id IS NOT null
  AND status IS NOT null
  AND id IS NOT null
PRIMARY KEY ((empresa_id), status, id)
WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
 and compaction = {'class': 'SizeTieredCompactionStrategy'}
 and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
 and dclocal_read_repair_chance = 0
 and speculative_retry = '99.0PERCENTILE';
*/
