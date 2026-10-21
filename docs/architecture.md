# Architecture

`chill-mcp` turns MCP tool calls into validated `POST /v4/{procedure}` calls to
the hosted `chill.institute` API. Procedure names, validators, and the RPC
client come from `chill-cli/v2/pkg`, so the CLI and MCP surfaces cannot drift.

| Path | Owns |
| --- | --- |
| `cmd/chill-mcp/` | Entrypoint: `stdio`, `http`, `health`, `version` |
| `internal/server/` | Tools, input validation, API mapping, bearer gate |
| `internal/buildinfo/` | Version and commit set at link time |
| `Dockerfile` | Non-root distroless image |

## Requests

The MCP endpoint is `/`, with `/mcp` as an alias; `GET /health` reports
liveness without calling the API. The MCP endpoint accepts only `POST`; other
methods get `405`. A `POST` without a well-formed bearer gets `401` before the
protocol layer. Each tool validates input, then forwards the same bearer with
`X-Request-Id`, `X-Chill-Client: mcp`, and the build version. In `stdio` mode
the bearer comes from the `chilly` profile.

The HTTP handler uses the [Go SDK](https://github.com/modelcontextprotocol/go-sdk)
stateless mode: no `Mcp-Session-Id`, server-initiated requests, or event store.
The SDK version in [`go.mod`](../go.mod) owns protocol negotiation.

Validation and API failures return `isError` with a `code: message` prefix such
as `invalid_url:` or `auth_error:`, never response bodies.

## Delivery

Pull requests run [Verify](../.github/workflows/verify.yml): verify, smoke, the
image proof, and the release-rules smoke. On `main`,
[Main](../.github/workflows/main.yml) reruns verify and smoke, then `release`
picks the next version, builds the image once with
[`scripts/image.sh`](../scripts/image.sh), boots it read-only with all
capabilities dropped, and checks version, health, and `401`; only then does it
tag and draft the release. `publish`, the only job with package, attestation,
and OIDC write access, fails unless the image ID matches the proven one, then
pushes, attests, records the digest, and publishes. Floating tags, the latest
release mark, and the production deploy move only for the newest `v*` tag on
the `main` tip, so rerunning an old run cannot roll them back.
