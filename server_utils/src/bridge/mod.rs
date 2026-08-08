//! SSE bridge: relays events between the Genix backend and browser tabs.
//!
//! Why it exists: AWS Lambda cannot hold a stream open for a whole agent turn, and it cannot
//! receive the browser's answer inside the same invocation. This process runs on a normal
//! server, keeps the browser's connection, and is the rendezvous point in both directions —
//! the backend pushes events with `POST /publish` and issues blocking commands with
//! `POST /rpc`, while the browser reads `GET /sse` and answers on `POST /in`.
//!
//! It holds no business logic and no database connection. Messages are opaque JSON and
//! nothing is buffered: a message for a disconnected tab is dropped.

pub mod auth;
pub mod channel;
pub mod http;
pub mod token;
