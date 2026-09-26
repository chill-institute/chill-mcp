# Agent guide

`chill-mcp` is the MCP server for chill.institute. Read [README.md](./README.md)
for tools, transports, and configuration.

## Work

```bash
mise install
mise run hooks
```

The pre-push hook runs `mise run verify`.

## Proof map

| Change | Check | Runs | Leaves |
| --- | --- | --- | --- |
| Docs, config | `mise run verify` | local, pre-push, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `verify` | exit status |
| Tools, validation, error mapping | `mise run verify` (format, tidy, lint, tests at 80% coverage, govulncheck); tests drive an in-process MCP client against a fake API | local, pre-push, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `verify` | exit status, `coverage.out` |
| Transport, bearer gate, process lifecycle | `mise run smoke`: built binary on `127.0.0.1:7199`, `health`, `401` without bearer | local, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `verify` | exit status, `./chill-mcp` |
| Image | `mise run docker:prove`: `scripts/image.sh` builds, then boots read-only and checks version, health, and `401` | local, [PR](./.github/workflows/verify.yml), [main](./.github/workflows/main.yml) `release` on the exact image ID that `publish` loads and pushes | `chill-mcp:local` |
| Workflows | `mise run actions` (actionlint, zizmor; inside verify) | local, CI with verify | exit status |
| Pushed workflow changes | [shared scan](https://github.com/chill-institute/.github/tree/main/.github/actions/scan), last step of the `verify` job in [Main](./.github/workflows/main.yml) and [Verify](./.github/workflows/verify.yml): Actionlint and Zizmor when the pushed range touches workflows; secrets rely on GitHub secret scanning | CI on push to `main` (pushed range) and Verify dispatch (full history) | failed run |
| Release and deploy | push to `main` | [main](./.github/workflows/main.yml) `release` and `publish`, then chill-engine `deploy-mcp.yml`, which checks private and public `/health` | tag, GitHub release with image digest, `ghcr.io/chill-institute/chill-mcp` tags with build provenance, deployed `mcp.chill.institute` |

Each commit type listed in [`.releaserc.json`](./.releaserc.json), including
`docs`, publishes an image and deploys it to production.

Gaps:

- No MCP client drives `initialize`, `tools/list`, or `tools/call` against the
  running binary or image. Owner: chill-institute/chill-mcp.
- No lane calls a tool against the hosted API or `mcp.chill.institute` with a
  real bearer. Owner: operator.
- No Markdown or link check covers docs. Owner: chill-institute/chill-mcp.

## Contracts

- One tool maps to one hosted procedure; validate input with
  `chill-cli/v2/pkg/chill` before building a request. Never construct URLs from
  raw tool arguments.
- Stateless only: no sessions, no storage, no token cache. The bearer comes
  from the request in `http` mode and from the `chilly` profile in `stdio` mode.
- No OAuth flows or discovery endpoints. Missing bearer is `401` with
  `WWW-Authenticate: Bearer`; token validity is the API's decision.
- Tool failures return `isError` results with a stable `code: message` prefix;
  protocol errors are reserved for transport problems.
- Hosted strings are data. Do not log headers, tokens, request bodies, or API
  responses.
- `add_transfer` and `update_user_setting` are the only mutations; keep
  `dry_run` and their annotations intact.

## Ownership

- Entry point, transports, and process lifecycle: `cmd/chill-mcp/`
- Tool definitions, validation, and API mapping: `internal/server/`
- Image: `Dockerfile`, `scripts/image.sh`; publish flow: `.github/workflows/main.yml`
- Production hosting pins the published image digest and is owned outside
  this repository.
