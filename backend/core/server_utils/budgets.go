package server_utils

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const budgetMutationPayloadSize = 20

type BudgetOperation uint8

const (
	BudgetSetDaily        BudgetOperation = 1
	BudgetSetCurrent      BudgetOperation = 2
	BudgetIncreaseCurrent BudgetOperation = 3
)

var (
	ErrBudgetMonthNotConfigured = errors.New("company credit budget is not configured for the current month")
	ErrBudgetMutationOverflow   = errors.New("company credit budget overflow")
)

func MutateCompanyCreditBudget(
	ctx context.Context,
	companyID int32,
	operation BudgetOperation,
	cpuCredits, inferenceCredits uint64,
) error {
	client := serverUtils()
	if client == nil {
		return ErrCreditLimiterMissing
	}
	payload, err := encodeBudgetMutation(companyID, operation, cpuCredits, inferenceCredits)
	if err != nil {
		return err
	}
	reply, err := client.requestOnce(ctx, opcodeMutateCompanyBudget, payload, 3*time.Second)
	if err != nil {
		return err
	}
	switch reply.status {
	case 0:
		return nil
	case 1:
		return ErrBudgetMonthNotConfigured
	case 2:
		return ErrBudgetMutationOverflow
	default:
		return fmt.Errorf("%w: budget mutation returned status %d", ErrServerUtilsUnavailable, reply.status)
	}
}

func encodeBudgetMutation(
	companyID int32,
	operation BudgetOperation,
	cpuCredits, inferenceCredits uint64,
) ([]byte, error) {
	if companyID <= 0 || companyID > 0xFF_FFFF {
		return nil, errors.New("company ID must fit positive uint24")
	}
	if operation < BudgetSetDaily || operation > BudgetIncreaseCurrent {
		return nil, fmt.Errorf("unknown budget operation %d", operation)
	}
	if cpuCredits > uint64(^uint64(0)>>1) || inferenceCredits > uint64(^uint64(0)>>1) {
		return nil, errors.New("credit budget values must fit int64")
	}
	payload := make([]byte, budgetMutationPayloadSize)
	writeUint24(payload[0:3], uint32(companyID))
	payload[3] = byte(operation)
	binary.BigEndian.PutUint64(payload[4:12], cpuCredits)
	binary.BigEndian.PutUint64(payload[12:20], inferenceCredits)
	return payload, nil
}
