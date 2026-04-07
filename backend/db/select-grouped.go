package db

type RecordGroup[T any] struct {
	IndexGroupID int64
	IndexGroupIDValues []int64
	Records []T
	
}
