//! The shared raw-TCP port: transport, authentication, and opcode routing.
//!
//! Services reachable here own their payload codecs and their business logic in their own
//! modules (`limiter`, and later `lock`). This tree owns only what they have in common: the
//! listener, the connection handshake, the frame HMAC, and the opcode table that decides which
//! module parses the bytes.

pub mod auth;
pub mod protocol;
pub mod server;
