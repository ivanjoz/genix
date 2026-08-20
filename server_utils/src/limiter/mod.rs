//! Credit rate limiter: atomic CPU/inference quota enforcement, reached through opcode `0x01`.
//!
//! The Go backend opens one persistent connection and sends a 20-byte frame per charge, which
//! `service` authenticates and routes here as an 11-byte payload. Each charge is admitted or
//! rejected against burst/hour policy plus per-company day/month entitlement — both counted
//! on the Lima business day the usage rows are bucketed by — atomically, in memory.
//!
//! Accepted charges are aggregated per API group into five-minute and daily records and
//! flushed to ScyllaDB as absolute snapshots every 15 seconds — company totals to
//! `credit_usage_company`, per-user totals to `credit_usage_user` — and only the rows
//! that actually changed. The same flush publishes each charged company's daily and
//! month-to-date counters into `company_credit_budget`, next to the entitlement they are
//! compared against, so a reader gets the figures admission was decided on instead of
//! re-summing the usage rows. It fails closed: usage that cannot be loaded blocks admission.
//!
//! Version one must run as a single active process. Two instances would hold independent
//! quota state and would overwrite each other's absolute rows.

pub mod access;
pub mod aggregation;
pub mod budget;
pub mod credits_blob;
pub mod protocol;
pub mod quota;
pub mod storage;
pub mod time_frame;
