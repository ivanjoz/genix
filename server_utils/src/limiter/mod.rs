//! Credit rate limiter: atomic CPU/inference quota enforcement, reached through opcode `0x01`.
//!
//! The Go backend opens one persistent connection and sends a 20-byte frame per charge, which
//! `service` authenticates and routes here as an 11-byte payload. Each charge is admitted or
//! rejected against company and user quotas for three windows (ten seconds, UTC hour, UTC day),
//! atomically, in memory.
//!
//! Accepted charges are aggregated per API group into five-minute and daily records and
//! flushed to ScyllaDB's `credit_usage` as absolute snapshots every 15 seconds — only the rows
//! that actually changed. It fails closed: usage that cannot be loaded blocks admission.
//!
//! Version one must run as a single active process. Two instances would hold independent
//! quota state and would overwrite each other's absolute rows.

pub mod aggregation;
pub mod credits_blob;
pub mod protocol;
pub mod quota;
pub mod storage;
pub mod time_frame;
