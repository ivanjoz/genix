# Unified server configuration

Run the public entrypoint from the repository root:

```bash
sudo python3 scripts/configure.py
```

The selection combines component digits with exactly one binary-source digit:

| Digit | Selection |
| --- | --- |
| `1` | Database host: ScyllaDB, GenixSearch and Qdrant |
| `2` | Backend service and/or its Nginx proxy |
| `3` | Server Utils service and SSE bridge proxy |
| `7` | Compile selected Genix services from source |
| `8` | Download selected Genix services from the latest public GitHub release |

For example, `238` configures Backend and Server Utils from precompiled binaries. The same
selection can be passed non-interactively as `sudo python3 scripts/configure.py 238`; Backend
still asks whether this host owns systemd, Nginx, or both.

When Backend mode `1` or `2` places the backend systemd service on this host, Server Utils is
installed in VPS service-only mode: the backend serves `/agent/stream` itself, so
`sse_bridge.url` is optional and no Server Utils Nginx vhost is generated. A standalone Server
Utils selection (`38`) remains the Lambda companion mode and requires `sse_bridge.url` for its
public SSE bridge.

Option `8` maps `x86_64` to the `amd64` assets and `aarch64`/`arm64` to the `arm64` assets. It
downloads `SHA256SUMS` and the required files from
`https://github.com/ivanjoz/genix/releases/latest/download`, verifies each SHA-256 checksum, and
only then invokes the nested installer. A matching asset already under `tmp/` is reused after
verification, so rerunning configuration does not download the large binaries again. A verified
latest asset takes precedence over an older installed binary. Option `7` requires the
corresponding source tree and never falls back to a binary.

The `7`/`8` choice applies only to Backend and Server Utils. Database configuration already owns
the package/release strategy for ScyllaDB, GenixSearch and Qdrant, so the selected digit is ignored
when component `1` runs.

Detailed component behavior:

- [Database](CONFIGURE_DB.md)
- [Backend Service](CONFIGURE_SERVER.md)
- [Server Utils](CONFIGURE_SERVER_UTILS.md)
