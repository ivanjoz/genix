package db

import (
	"github.com/ivanjoz/genix-orm/dataframe"
	"github.com/ivanjoz/genix-orm/dataframe/framesql"
	orm "github.com/ivanjoz/genix-orm/db"
)

// ─── FrameSQL (genix-orm/dataframe/framesql/FRAMESQL.md) ───────────────────────
// One SQL statement over one DataFrame, folded from the frame's rows: what the agent answers sales
// questions with.

type (
	// SQLFrame is a frame a statement may read: its columns, their kinds and its read (FrameSource).
	SQLFrame  = framesql.Frame
	SQLColumn = framesql.Column
	// SQLKind is what a column's integers mean: a day, money in cents, a record ID, a quantity or a number.
	SQLKind         = framesql.Kind
	SQLSource       = framesql.Source
	SQLResolvers    = framesql.Resolvers
	SQLOptions      = framesql.Options
	SQLResult       = framesql.Result
	SQLResultColumn = framesql.ResultColumn
	// SQLStatement is a statement bound to its frame, before its names are looked up and its rows read.
	SQLStatement = framesql.Statement
)

const (
	SQLInteger  = framesql.KindInteger
	SQLDay      = framesql.KindDay
	SQLCents    = framesql.KindCents
	SQLRef      = framesql.KindRef
	SQLDecimal  = framesql.KindDecimal
	SQLQuantity = framesql.KindQuantity
	// FrameMaxDays is the widest range of a frame's first Key one read covers.
	FrameMaxDays = dataframe.MaxFirstKeys
)

var (
	// PrepareFrameSQL parses a statement and binds it to one of the frames: Columns, then Run.
	PrepareFrameSQL = framesql.Prepare
	// FramePeriodStarts lists the days, weeks or months a range of days touches (a result column's Period).
	FramePeriodStarts = framesql.PeriodStarts
	// ErrFrameNotBuilt: the frame is being rebuilt after a change of shape and can't be read yet.
	ErrFrameNotBuilt = dataframe.ErrFrameNotBuilt
)

// FrameSource is the SQLSource of one partition of a table's DataFrame. It always reads fresh, and
// the read compacts the partition when the frame's CompactOn is due.
func FrameSource[RecordT Record[TableT, RecordT, D], TableT Schema[TableT], D Executor[TableT, RecordT]](
	partition int32, frameName string,
) SQLSource {
	return orm.FrameSource[RecordT, TableT, D](partition, frameName)
}
