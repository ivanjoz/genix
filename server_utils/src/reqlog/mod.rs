//! Opcode `0x04`: the end-of-request record.
//!
//! One row per finished request in `user_logs`, and one row per distinct failing code line in
//! `request_errors`. What is deliberately *not* here is the message and the stack — those are
//! already in CloudWatch under the request id, and the only thing this side keeps of them is which
//! code line failed and roughly what it said.
//!
//! Unlike the limiter, every failure in this tree is dropped rather than propagated: a missing log
//! row is a missing log row, and taking the process down over one would stop the limiter and the
//! bridge too.

pub mod errors;
pub mod protocol;
pub mod writer;
