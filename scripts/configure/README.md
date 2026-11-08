# Unified server configuration

Run the public entrypoint from the repository root:

```bash
sudo python3 scripts/configure.py
```

The selection combines component digits with exactly one binary-source digit:

| Digit | Selection |
| --- | --- |
| `1` | Database host: MariaDB and Qdrant |
| `2` | Backend service and/or its Nginx proxy |
| `3` | Fareward service and SSE bridge proxy |
| `7` | Compile selected Genix services from source |
| `8` | Download selected Genix services from the latest public GitHub release |

For example, `238` configures Backend and Fareward from precompiled binaries. The same
selection can be passed non-interactively as `sudo python3 scripts/configure.py 238`; Backend
asks whether this host owns systemd, Nginx, or both unless `--backend-mode 1|2|3` answers it.
`--staged-binaries` (with `8`) installs the binaries already in `tmp/` instead of downloading.

The order is fixed: database, backend, then `fn-homologate` (the backend's tables, run with the
installed binary as the service user), then fareward, which exits at startup without its tables.

## From your machine: `configure_remote.py`

```bash
python3 scripts/configure_remote.py 123                         # the only [[servers]] entry
python3 scripts/configure_remote.py 23 --server 150.136.42.240  # pick one; --backend-mode 1 is the default
```

It does the same install on a `[[servers]]` VPS without logging in to it:

1. Builds the binaries **here** from what is on disk (unpushed submodule commits included): the
   backend with `-tags lambda.norpc` (and `GOARM64=v8.2` on arm64), fareward with
   `cargo zigbuild` against glibc 2.39. Needs Go, rustup, `cargo-zigbuild` and `zig` locally; the
   VPS needs no toolchain.
2. Clones the repository on the VPS on first use (`repository`, default `/home/<user>/genix`), then
   uploads this machine's `scripts/configure.py` and `scripts/configure/` over it.
3. Uploads the server's own config file (the entry's `config`, e.g. `config.oracle.toml`,
   gitignored) as `<repository>/config.toml`, mode 0600.
4. Runs `sudo python3 scripts/configure.py <selection>8 --staged-binaries` there over SSH.

The SSH user needs **passwordless sudo**, which the default `ubuntu`/`opc`/`ec2-user` of the
Oracle and AWS images have: packages, systemd units, nginx and firewall rules are root's.

```toml
[[servers]]
host   = "150.136.42.240"
user   = "ubuntu"
key    = "/home/me/keys/server.pem"
arch   = "arm64"
bin    = "/usr/local/bin/genix/genix_app"   # deploy_vps.go
config = "config.oracle.toml"
```

When Backend mode `1` or `2` places the backend systemd service on this host, Fareward is
installed in VPS service-only mode: the backend serves `/agent/stream` itself, so
`sse_bridge.url` is optional and no Fareward Nginx vhost is generated. A standalone Server
Utils selection (`38`) remains the Lambda companion mode and requires `sse_bridge.url` for its
public SSE bridge.

Option `8` maps `x86_64` to the `amd64` assets and `aarch64`/`arm64` to the `arm64` assets. It
downloads `SHA256SUMS` and the required files from the release that publishes each component —
the backend from `https://github.com/ivanjoz/genix/releases/latest/download`, Fareward from
`https://github.com/ivanjoz/fareward/releases/latest/download` — verifies each SHA-256
checksum against the manifest of the release it came from, and
only then invokes the nested installer. A matching asset already under `tmp/` is reused after
verification, so rerunning configuration does not download the large binaries again. A verified
latest asset takes precedence over an older installed binary. Option `7` requires the
corresponding source tree and never falls back to a binary.

The `7`/`8` choice applies only to Backend and Fareward. Database configuration already owns
the package/release strategy for MariaDB and Qdrant, so the selected digit is ignored
when component `1` runs.

Detailed component behavior:

- [Database](CONFIGURE_DB.md)
- [Backend Service](CONFIGURE_SERVER.md)
- [Fareward](CONFIGURE_FAREWARD.md)
