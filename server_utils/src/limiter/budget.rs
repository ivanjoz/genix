//! Company credit-budget mutation codec for opcode `0x05`.

use thiserror::Error;

use crate::limiter::credits_blob::Credits;

pub const MUTATE_BUDGET_PAYLOAD_SIZE: usize = 20;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum BudgetOperation {
    SetDaily = 1,
    SetCurrent = 2,
    IncreaseCurrent = 3,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct BudgetMutation {
    pub company_id: i32,
    pub operation: BudgetOperation,
    pub credits: Credits,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum BudgetMutationReply {
    Ok = 0,
    CurrentMonthNotConfigured = 1,
    Overflow = 2,
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum BudgetProtocolError {
    #[error("company_id must be positive")]
    InvalidCompany,
    #[error("unknown budget operation {0}")]
    InvalidOperation(u8),
    #[error("credit budget values must fit signed 64-bit database columns")]
    CreditOverflow,
}

pub fn parse_budget_mutation(
    payload: &[u8; MUTATE_BUDGET_PAYLOAD_SIZE],
) -> Result<BudgetMutation, BudgetProtocolError> {
    let company_id =
        ((i32::from(payload[0])) << 16) | ((i32::from(payload[1])) << 8) | i32::from(payload[2]);
    if company_id <= 0 {
        return Err(BudgetProtocolError::InvalidCompany);
    }

    let operation = match payload[3] {
        1 => BudgetOperation::SetDaily,
        2 => BudgetOperation::SetCurrent,
        3 => BudgetOperation::IncreaseCurrent,
        value => return Err(BudgetProtocolError::InvalidOperation(value)),
    };
    let cpu = u64::from_be_bytes(payload[4..12].try_into().expect("fixed CPU width"));
    let inference = u64::from_be_bytes(payload[12..20].try_into().expect("fixed inference width"));
    if cpu > i64::MAX as u64 || inference > i64::MAX as u64 {
        return Err(BudgetProtocolError::CreditOverflow);
    }

    Ok(BudgetMutation {
        company_id,
        operation,
        credits: Credits { cpu, inference },
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_exact_budget_wire_offsets() {
        let mut payload = [0_u8; MUTATE_BUDGET_PAYLOAD_SIZE];
        payload[0..3].copy_from_slice(&[0x12, 0x34, 0x56]);
        payload[3] = BudgetOperation::IncreaseCurrent as u8;
        payload[4..12].copy_from_slice(&300_u64.to_be_bytes());
        payload[12..20].copy_from_slice(&25_u64.to_be_bytes());

        assert_eq!(
            parse_budget_mutation(&payload).unwrap(),
            BudgetMutation {
                company_id: 0x12_34_56,
                operation: BudgetOperation::IncreaseCurrent,
                credits: Credits {
                    cpu: 300,
                    inference: 25,
                },
            }
        );
    }
}
