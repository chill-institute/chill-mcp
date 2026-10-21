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

| Change | Check | CI |
| --- | --- | --- |
| Go, docs, workflows | `mise run verify`: format, module tidiness, lint, 80% coverage, govulncheck, actionlint, zizmor; tests drive an in-process MCP client against a fake API | [Verify](./.github/workflows/verify.yml), [Main](./.github/workflows/main.yml) |
| Transport, bearer gate, lifecycle | `mise run smoke`: built binary, `health`, `401` without bearer | Verify, Main |
| Image | `mise run docker:prove`: read-only boot, version, health, `401` | Verify, Main `release` on the published image ID |
| Release rules | `npm ci --prefix .github/release --ignore-scripts && node .github/release/smoke.mjs` | Verify |
| Release | push to `main`; only a release publishes and deploys an image | Main `release`, `publish` |

Gaps:

- No MCP client calls `tools/call` against the binary, image, or hosted API.
- No check covers Markdown links.
