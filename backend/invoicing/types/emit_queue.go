// Asking for pending documents to be sent.
//
// The request that creates a sale writes the document and returns; SUNAT is dealt
// with later, by the cron. The enqueue lives in the types leaf because `sales` is
// what creates the document and may not import the invoicing body — the handler
// this schedules is registered over in emit_worker.go.

package types

import "app/core"

const (
	// EmitPendingActionID is the cron action that sends one company's pending
	// documents. Ids 2 to 5 are taken by other modules.
	EmitPendingActionID = int16(6)
	// emitSweepFrameMinutes is how long a document waits, at most, before the
	// sweep that follows its creation runs.
	emitSweepFrameMinutes = int8(5)
)

// ScheduleEmitPendingSweep asks for a company's pending documents to be sent.
//
// One-shot rather than recurring, and enqueued by whoever wrote a document: the
// scheduler dedupes the same logical action inside a frame, so a hundred sales in
// five minutes enqueue one row, and a company that stops selling stops being
// swept. The handler re-schedules itself while anything is still pending, which is
// what carries a backlog across frames.
//
// Failures are swallowed. The document is already persisted and the report shows
// it as pending with a "send now" button, so a scheduler that could not write its
// row must not take down the sale that just succeeded — ScheduleCronAction panics
// on a database error.
func ScheduleEmitPendingSweep(companyID int32) {
	defer func() {
		if problem := recover(); problem != nil {
			core.Log("no se pudo encolar el envío de comprobantes de la empresa", companyID, ":", problem)
		}
	}()

	core.ScheduleCronAction(core.CronAction{
		ActionID:  EmitPendingActionID,
		CompanyID: companyID,
		Params:    core.ExecArgs{Param1: int64(companyID)},
	}, emitSweepFrameMinutes)
}
