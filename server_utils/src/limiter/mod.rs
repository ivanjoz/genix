//! Credit rate limiter: atomic CPU/inference quota enforcement over raw TCP.
//!
//! The Go backend opens one persistent connection and sends a fixed 19-byte frame per charge,
//! authenticated by a sequence-bound HMAC. Each frame is admitted or rejected against company
//! and user quotas for three windows (ten seconds, UTC hour, UTC day), atomically, in memory.
//!
//! Accepted charges are aggregated per API group into five-minute and daily records and
//! flushed to ScyllaDB's `credit_usage` as absolute snapshots every 15 seconds — only the rows
//! that actually changed. It fails closed: usage that cannot be loaded blocks admission.
//!
//! Version one must run as a single active process. Two instances would hold independent
//! quota state and would overwrite each other's absolute rows.

pub mod aggregation;
pub mod auth;
pub mod credits_blob;
pub mod protocol;
pub mod quota;
pub mod server;
pub mod storage;
pub mod time_frame;
