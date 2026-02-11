The cache version feature is a way to save the change version of an object or record based on a group
The object will have a cache group that is a UInt8 value, the is the uint8(object_id). The version of the cache is also a uint8 value that will increment from 0 to 255 and then again from 0 (when overflow)

The schema will be declared like this: (real example)

func (e ProductoTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:          "productos",
		Partition:     e.EmpresaID,
		UseSequences:  true,
		SaveCacheVersion: true,
		Keys:          []db.Coln{e.ID.Autoincrement(0)},
		GlobalIndexes: [][]db.Coln{{ e.CategoriasConStock }},
		ViewsDeprecated: []db.View{
			{Cols: []db.Coln{e.Status}, KeepPart: true},
			{Cols: []db.Coln{e.StockStatus}, KeepPart: true},
			{Cols: []db.Coln{e.Updated}, KeepPart: true},
		},
	}
}

and It will have a SaveCacheVersion property. if SaveCacheVersion is true, when doing an Insert or Update operation, the ORM will fetch the cache version record that contains all the cache version off all the groups.

/* CacheVersion Table */
type CacheVersion struct {
	TableStruct[CacheVersionTable, CacheVersion]
	Partition 	 int32
	TableID      int32
	CachedValues []byte
}

type CacheVersionTable struct {
	TableStruct[CacheVersionTable, CacheVersion]
	Partition    Col[IncrementTable, int32]
	TableID 		Col[IncrementTable, int32]  
	CachedValues Col[IncrementTable, []byte]  
}

func (e CacheVersionTable) GetSchema() TableSchema {
	return TableSchema{
		Name:           "cache_version",
		Partition: 			e.GetSchema().Partition,
		Keys:           []Coln{e.TableID},
	}
}

The TableID is the int32 hash of the table.

The CachedValues are where all the key:value pairs of all cache groups are,
in the []byte array, the 1º value is the cache group id, and the 2º value if the cache version, the 3º value is the cache group id, the 4º the version, the 5º the id and so on.

So you have to convert this []byte in a map[uint8]uint8

then you have to identify all the cache groups by doing uint8(id) on the ids, and increment the cache groups that are being updated (or inserted). then, you have to convert the map[uint8]uint8 into a []byte and save the cache version record in the "cache_version" table, using the tableID and the partitionID. The partition must be int32 or int64 always.

Important validations:
The Table must have an int16, int32 or int64 single key. This is good: 

Keys:  []db.Coln{e.ID.Autoincrement(0)},

If there are multiple keys or there is no partition ID, just panic.

Also the object / struct definition must have a CacheVersion field or a field with a "ccv" json tag, if not, panic. The CacheVersion field must be uint8, if not, panic.

When you select from this table, you must get the "cache_version" record with all the cache version groups from the database, and assing the cache version to CacheVersion field to the  all records.
