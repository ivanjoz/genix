package comercial

import (
	"app/core"
	"slices"
)

const (
	// Trace 1: order created without payment and without delivery.
	SaleOrderTraceStage1Generated int8 = 1
	// Trace 2: order created with payment already registered.
	SaleOrderTraceStage1Paid int8 = 2
	// Trace 3: order created with delivery already registered.
	SaleOrderTraceStage1Delivered int8 = 3
	// Trace 4: order completed during creation because payment and delivery were both registered.
	SaleOrderTraceStage1Completed int8 = 4
	// Trace 5: order started as generated and then payment was registered in a later step.
	SaleOrderTraceStage2GeneratedPaid int8 = 5
	// Trace 6: order started as generated and then delivery was registered in a later step.
	SaleOrderTraceStage2GeneratedDelivered int8 = 6
	// Trace 7: order started as generated+delivered and then payment was registered in a later step.
	SaleOrderTraceStage2CompletedFromGeneratedDelivered int8 = 7
	// Trace 8: order started as generated+paid and then delivery was registered in a later step.
	SaleOrderTraceStage2CompletedFromGeneratedPaid int8 = 8
	// Trace 9: order started as generated and was fully completed after creation.
	SaleOrderTraceStage3GeneratedDeliveredPaid int8 = 9
)

// The key is {previousTrace, hasPaymentAction, hasDeliveryAction}.
// A missing key means the transition is invalid for the current flow.
var saleOrderStatusTraceTransitions = map[[3]int8]int8{
	{0, 0, 0}: SaleOrderTraceStage1Generated,
	{0, 1, 0}: SaleOrderTraceStage1Paid,
	{0, 0, 1}: SaleOrderTraceStage1Delivered,
	{0, 1, 1}: SaleOrderTraceStage1Completed,

	{SaleOrderTraceStage1Generated, 1, 0}: SaleOrderTraceStage2GeneratedPaid,
	{SaleOrderTraceStage1Generated, 0, 1}: SaleOrderTraceStage2GeneratedDelivered,
	{SaleOrderTraceStage1Generated, 1, 1}: SaleOrderTraceStage3GeneratedDeliveredPaid,

	{SaleOrderTraceStage1Paid, 0, 1}:      SaleOrderTraceStage2CompletedFromGeneratedPaid,
	{SaleOrderTraceStage1Delivered, 1, 0}: SaleOrderTraceStage2CompletedFromGeneratedDelivered,

	{SaleOrderTraceStage2GeneratedPaid, 0, 1}:      SaleOrderTraceStage3GeneratedDeliveredPaid,
	{SaleOrderTraceStage2GeneratedDelivered, 1, 0}: SaleOrderTraceStage3GeneratedDeliveredPaid,
}

var saleOrderStatusTracePendingPayment = []int8{
	SaleOrderTraceStage1Generated,
	SaleOrderTraceStage1Delivered,
	SaleOrderTraceStage2GeneratedDelivered,
}

var saleOrderStatusTracePendingDelivery = []int8{
	SaleOrderTraceStage1Generated,
	SaleOrderTraceStage1Paid,
	SaleOrderTraceStage2GeneratedPaid,
}

var saleOrderStatusTraceCompleted = []int8{
	SaleOrderTraceStage1Completed,
	SaleOrderTraceStage2CompletedFromGeneratedDelivered,
	SaleOrderTraceStage2CompletedFromGeneratedPaid,
	SaleOrderTraceStage3GeneratedDeliveredPaid,
}

// GetSaleOrderStatusTracesByPendingStatus returns the trace states that still require the
// requested action. The API contract keeps using 2 = pending payment and 3 = pending delivery.
func GetSaleOrderStatusTracesByPendingStatus(pendingStatus int8) []int8 {
	if pendingStatus == 2 {
		return saleOrderStatusTracePendingPayment
	}
	if pendingStatus == 3 {
		return saleOrderStatusTracePendingDelivery
	}
	return nil
}

// GetSaleOrderStatusTracesByOrderStatus expands high-level order status filters into the
// concrete trace states used by the backend indexes.
func GetSaleOrderStatusTracesByOrderStatus(orderStatus int8) []int8 {
	if orderStatus == 4 {
		return saleOrderStatusTraceCompleted
	}
	return nil
}

// CalculateSaleOrderStatusTrace maps the order flow to the compact 1..9 trace states used for filtering.
//
// Transition summary:
//   - 0 + no actions          => 1
//   - 0 + payment             => 2
//   - 0 + delivery            => 3
//   - 0 + payment + delivery  => 4
//   - 1 + payment             => 5
//   - 1 + delivery            => 6
//   - 1 + payment + delivery  => 9
//   - 2 + delivery            => 8
//   - 3 + payment             => 7
//   - 5 + delivery            => 9
//   - 6 + payment             => 9
//
// Any other transition is treated as invalid because it would imply repeating an already
// executed action or mutating a completed trace.
func CalculateSaleOrderStatusTrace(previousTrace int8, actionIDs []int8) (int8, error) {
	hasPaymentAction := int8(0)
	hasDeliveryAction := int8(0)

	if slices.Contains(actionIDs, 2) {
		hasPaymentAction = 1
	}
	if slices.Contains(actionIDs, 3) {
		hasDeliveryAction = 1
	}

	transitionKey := [3]int8{previousTrace, hasPaymentAction, hasDeliveryAction}
	if nextTrace, ok := saleOrderStatusTraceTransitions[transitionKey]; ok {
		return nextTrace, nil
	}

	return 0, core.Err(
		"transicion invalida de StatusTrace. previousTrace:",
		previousTrace,
		" payment:",
		hasPaymentAction,
		" delivery:",
		hasDeliveryAction,
	)
}
