//! Shared implementation for the Genix server-utilities process.
//!
//! Two independent services live here, one module tree each. `config` is the only thing they
//! share besides the process itself and the tokio runtime.

pub mod bridge;
pub mod config;
pub mod limiter;
