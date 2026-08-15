# Golang Genix Lambda Compatible Backend
This is the backend for the Genix ERP + Ecommerce project

## Precompiled Linux binaries

Every versioned [Genix GitHub Release](https://github.com/ivanjoz/genix/releases) publishes static
backend executables for Linux amd64 and arm64. A version URL is immutable; `latest/download` follows
the newest stable release. Pin a version URL for production automation.

| Machine (`uname -m`) | Release asset |
| --- | --- |
| `x86_64` | `genix_app_linux_amd64` |
| `aarch64` or `arm64` | `genix_app_linux_arm64` |

```bash
# Map the host name to the stable architecture suffix used by release assets.
case "$(uname -m)" in
  x86_64) release_architecture=amd64 ;;
  aarch64|arm64) release_architecture=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# Replace latest/download with download/vX.Y.Z to pin an immutable production version.
release_base_url=https://github.com/ivanjoz/genix/releases/latest/download
curl --fail --location \
  --output "genix_app_linux_${release_architecture}" \
  "${release_base_url}/genix_app_linux_${release_architecture}"
curl --fail --location --output SHA256SUMS "${release_base_url}/SHA256SUMS"

# Verify the selected asset before granting execute permission.
grep " genix_app_linux_${release_architecture}$" SHA256SUMS | sha256sum --check --strict
chmod 0755 "genix_app_linux_${release_architecture}"
```
