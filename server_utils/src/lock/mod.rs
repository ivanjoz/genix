//! Ephemeral distributed locks, reached through opcodes `0x02` and `0x03`.
//!
//! One holder per `(action, identifier)`, no exceptions: every lock here is mutual exclusion.
//! The daemon interprets neither field — the Go call sites decide what is being serialized —
//! which is what lets one service cover every use case in the project.
//!
//! Ownership is bound to the TCP connection rather than to a lease token. The permit lives in
//! the connection task, so a disconnect, a crash, a killed Lambda, and a client that simply goes
//! silent all free the lock through the same code path, with no sweeper and no token to
//! validate. The client-supplied lease is applied by `service::server` as the connection's read
//! deadline while holding.
//!
//! State is in memory only: a daemon restart drops every lock, and two instances would hand the
//! same key to two holders. This must run as a single active process, like the rate limiter.

pub mod protocol;
pub mod registry;
