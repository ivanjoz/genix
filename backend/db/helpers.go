package db

import "hash/fnv"

func BasicHashInt(s string) int32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int32(h.Sum32())
}
