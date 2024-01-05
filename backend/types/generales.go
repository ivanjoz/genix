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

/*
CREATE MATERIALIZED VIEW pais_ciudades__updated_view
AS
SELECT  * FROM pais_ciudades
WHERE pais_id IS NOT null
  AND ciudad_id IS NOT null
  AND updated IS NOT null
PRIMARY KEY ((pais_id), updated, ciudad_id)
WITH caching = {'keys': 'ALL', 'rows_per_partition': 'ALL'}
 and compaction = {'class': 'SizeTieredCompactionStrategy'}
 and compression = {'compression_level': '3', 'sstable_compression': 'org.apache.cassandra.io.compress.ZstdCompressor'}
 and dclocal_read_repair_chance = 0
 and speculative_retry = '99.0PERCENTILE';
*/
