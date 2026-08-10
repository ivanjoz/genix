//! Shared implementation for the Genix server-utilities process.
//!
//! Two transports: `service` is the raw-TCP port (routing and authentication only, with each
//! opcode's logic living in its own module such as `limiter`), and `bridge` is the HTTP SSE
//! relay. `config` is the only thing they share besides the process and the tokio runtime.

pub mod bridge;
pub mod config;
pub mod limiter;
pub mod lock;
pub mod service;
