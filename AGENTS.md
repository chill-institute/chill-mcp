# Agent guide

`chill-mcp` is the stateless MCP server for chill.institute; tools, transports,
and configuration are in [README.md](./README.md).

## Work

```bash
mise install
mise run hooks   # pre-push runs mise run verify
mise run verify
mise run smoke
```

## Rules

- One tool maps to one hosted procedure. Validate input with
  `chill-cli/v2/pkg/chill` before building a request; never build URLs from raw
  arguments. Add missing validators in `chill-cli` first.
- No sessions, storage, token cache, OAuth, or discovery endpoints. The bearer
  comes from the request (`http`) or the `chilly` profile (`stdio`); a missing
  bearer gets `401` with `WWW-Authenticate: Bearer`.
- Tool failures are `isError` results with a stable `code: message` prefix;
  protocol errors are for transport faults.
- Never log headers, tokens, request bodies, or API responses.
- `add_transfer` and `update_user_setting` are the only mutations; keep their
  `dry_run` and annotations.
- Tools live in `internal/server/`; transports and lifecycle in `cmd/chill-mcp/`.
- `releaseRules` in [`.releaserc.json`](./.releaserc.json) match every
  chill.institute package repo; `.github/release/smoke.mjs` fails on drift.

## Proof map

| Change | Check |
| --- | --- |
| Go, docs, workflows | `mise run verify`: module tidiness, golangci-lint with gofmt and goimports, tests that drive an in-process MCP client against a fake API, actionlint, zizmor |
| Dependencies | `mise run security` (govulncheck; needs network, runs in CI) |
| Transport, bearer gate, lifecycle | `mise run smoke`: built binary, `health`, `401` without bearer |
| Image | `mise run docker:prove`: read-only boot, version, health, `401` (needs Docker) |
| Release rules | `mise run release:smoke` |

Gaps:

- No MCP client calls `tools/call` against the binary, image, or hosted API.
- No check covers Markdown links.
